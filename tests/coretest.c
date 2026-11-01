/*
 * coretest: frontend libretro mínimo, sem janela, pra teste automático.
 *
 * Carrega um core, abre o jogo, roda N frames apertando START de vez em quando
 * (sempre igual, então o resultado é repetível) e imprime um JSON com o que viu:
 * quantos frames vieram, quantos diferentes, se tem imagem, hash do último frame.
 * O último frame vai em RGB cru pro arquivo de saída (o check.py vira PNG).
 *
 * Core que pede GPU (N64, por exemplo) ganha um contexto OpenGL sem janela
 * (EGL surfaceless) e um framebuffer; o frame é lido de volta com glReadPixels.
 * CORETEST_NO_GPU=1 recusa a GPU, pra testar o render por software.
 *
 * Uso: coretest <core.so> <jogo> <frames> <dir_sistema> <saida.rgb>
 */
#include <EGL/egl.h>
#include <EGL/eglext.h>
#include <GL/gl.h>
#include <GL/glext.h>
#include <dlfcn.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libretro.h"

static const char *sys_dir;
static unsigned pixel_format = RETRO_PIXEL_FORMAT_0RGB1555;
static unsigned frame_no, video_frames, distinct, blank_streak;
static uint64_t last_hash;
static unsigned last_w, last_h;
static uint8_t *last_rgb;
static bool hw_asked;

/* GPU: contexto EGL + framebuffer do frontend */
static struct retro_hw_render_callback hw;
static bool hw_on;
static EGLDisplay egl_dpy = EGL_NO_DISPLAY;
static EGLContext egl_ctx = EGL_NO_CONTEXT;
static GLuint fbo;
static PFNGLGENFRAMEBUFFERSPROC gl_gen_framebuffers;
static PFNGLBINDFRAMEBUFFERPROC gl_bind_framebuffer;
static PFNGLGENRENDERBUFFERSPROC gl_gen_renderbuffers;
static PFNGLBINDRENDERBUFFERPROC gl_bind_renderbuffer;
static PFNGLRENDERBUFFERSTORAGEPROC gl_renderbuffer_storage;
static PFNGLFRAMEBUFFERRENDERBUFFERPROC gl_framebuffer_renderbuffer;
static PFNGLCHECKFRAMEBUFFERSTATUSPROC gl_check_framebuffer_status;
static void (*gl_read_pixels)(GLint, GLint, GLsizei, GLsizei, GLenum,
                                          GLenum, void *);
static void (*gl_pixel_storei)(GLenum, GLint);

static void log_cb(enum retro_log_level level, const char *fmt, ...) {
  if (level < RETRO_LOG_WARN && !getenv("CORETEST_VERBOSE"))
    return;
  va_list ap;
  va_start(ap, fmt);
  vfprintf(stderr, fmt, ap);
  va_end(ap);
}

/* opções do core: guarda o padrão declarado e devolve no GET_VARIABLE, como o
 * RetroArch faz. Core sem resposta pode ficar com config sem valor (lixo). */
#define MAX_OPTS 1024
static struct { char *key, *value; } opts[MAX_OPTS];
static unsigned n_opts;

static void opt_set(const char *key, const char *value) {
  if (!key || !value)
    return;
  for (unsigned i = 0; i < n_opts; i++)
    if (!strcmp(opts[i].key, key))
      return; /* o primeiro registro vale */
  if (n_opts < MAX_OPTS) {
    opts[n_opts].key = strdup(key);
    opts[n_opts++].value = strdup(value);
  }
}

/* v0: "Descrição; padrão|outro|outro" */
static void opts_v0(const struct retro_variable *vars) {
  for (; vars && vars->key; vars++) {
    const char *list = vars->value ? strstr(vars->value, "; ") : NULL;
    if (!list)
      continue;
    char first[256];
    snprintf(first, sizeof first, "%s", list + 2);
    first[strcspn(first, "|")] = 0;
    opt_set(vars->key, first);
  }
}

static void opts_v1(const struct retro_core_option_definition *defs) {
  for (; defs && defs->key; defs++)
    opt_set(defs->key, defs->default_value ? defs->default_value : defs->values[0].value);
}

static void opts_v2(const struct retro_core_options_v2 *o) {
  const struct retro_core_option_v2_definition *defs = o ? o->definitions : NULL;
  for (; defs && defs->key; defs++)
    opt_set(defs->key, defs->default_value ? defs->default_value : defs->values[0].value);
}

static uintptr_t hw_framebuffer(void) { return fbo; }

static retro_proc_address_t hw_proc(const char *sym) {
  return (retro_proc_address_t)eglGetProcAddress(sym);
}

#define GLPROC(var, name)                                                      \
  if (!(*(void **)&var = (void *)eglGetProcAddress(name)))                     \
    return false;

/* contexto sem janela e sem config (EGL_KHR_no_config_context) */
static bool egl_init(void) {
  PFNEGLGETPLATFORMDISPLAYEXTPROC platform_display =
      (PFNEGLGETPLATFORMDISPLAYEXTPROC)eglGetProcAddress("eglGetPlatformDisplayEXT");
  if (platform_display)
    egl_dpy = platform_display(EGL_PLATFORM_SURFACELESS_MESA, EGL_DEFAULT_DISPLAY, NULL);
  if (egl_dpy == EGL_NO_DISPLAY)
    egl_dpy = eglGetDisplay(EGL_DEFAULT_DISPLAY);
  if (egl_dpy == EGL_NO_DISPLAY || !eglInitialize(egl_dpy, NULL, NULL))
    return false;

  EGLint attrs[16], n = 0;
  bool gles = hw.context_type == RETRO_HW_CONTEXT_OPENGLES2 ||
              hw.context_type == RETRO_HW_CONTEXT_OPENGLES3 ||
              hw.context_type == RETRO_HW_CONTEXT_OPENGLES_VERSION;
  eglBindAPI(gles ? EGL_OPENGL_ES_API : EGL_OPENGL_API);
  if (hw.context_type == RETRO_HW_CONTEXT_OPENGLES2) {
    attrs[n++] = EGL_CONTEXT_MAJOR_VERSION, attrs[n++] = 2;
  } else if (hw.context_type == RETRO_HW_CONTEXT_OPENGLES3) {
    attrs[n++] = EGL_CONTEXT_MAJOR_VERSION, attrs[n++] = 3;
  } else if (hw.context_type == RETRO_HW_CONTEXT_OPENGLES_VERSION ||
             hw.context_type == RETRO_HW_CONTEXT_OPENGL_CORE) {
    attrs[n++] = EGL_CONTEXT_MAJOR_VERSION, attrs[n++] = (EGLint)hw.version_major;
    attrs[n++] = EGL_CONTEXT_MINOR_VERSION, attrs[n++] = (EGLint)hw.version_minor;
    if (!gles) {
      attrs[n++] = EGL_CONTEXT_OPENGL_PROFILE_MASK;
      attrs[n++] = EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT;
    }
  }
  attrs[n] = EGL_NONE;
  egl_ctx = eglCreateContext(egl_dpy, EGL_NO_CONFIG_KHR, EGL_NO_CONTEXT, attrs);
  if (egl_ctx == EGL_NO_CONTEXT ||
      !eglMakeCurrent(egl_dpy, EGL_NO_SURFACE, EGL_NO_SURFACE, egl_ctx))
    return false;

  GLPROC(gl_gen_framebuffers, "glGenFramebuffers")
  GLPROC(gl_bind_framebuffer, "glBindFramebuffer")
  GLPROC(gl_gen_renderbuffers, "glGenRenderbuffers")
  GLPROC(gl_bind_renderbuffer, "glBindRenderbuffer")
  GLPROC(gl_renderbuffer_storage, "glRenderbufferStorage")
  GLPROC(gl_framebuffer_renderbuffer, "glFramebufferRenderbuffer")
  GLPROC(gl_check_framebuffer_status, "glCheckFramebufferStatus")
  GLPROC(gl_read_pixels, "glReadPixels")
  GLPROC(gl_pixel_storei, "glPixelStorei")
  return true;
}

/* framebuffer do tamanho máximo do core: cor + profundidade/stencil se pedir */
static bool fbo_init(unsigned w, unsigned h) {
  GLuint rb[2];
  gl_gen_framebuffers(1, &fbo);
  gl_bind_framebuffer(GL_FRAMEBUFFER, fbo);
  gl_gen_renderbuffers(2, rb);
  gl_bind_renderbuffer(GL_RENDERBUFFER, rb[0]);
  gl_renderbuffer_storage(GL_RENDERBUFFER, GL_RGBA8, (GLsizei)w, (GLsizei)h);
  gl_framebuffer_renderbuffer(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_RENDERBUFFER, rb[0]);
  if (hw.depth || hw.stencil) {
    gl_bind_renderbuffer(GL_RENDERBUFFER, rb[1]);
    gl_renderbuffer_storage(GL_RENDERBUFFER, GL_DEPTH24_STENCIL8, (GLsizei)w, (GLsizei)h);
    gl_framebuffer_renderbuffer(GL_FRAMEBUFFER, GL_DEPTH_STENCIL_ATTACHMENT,
                                GL_RENDERBUFFER, rb[1]);
  }
  return gl_check_framebuffer_status(GL_FRAMEBUFFER) == GL_FRAMEBUFFER_COMPLETE;
}

static bool environment(unsigned cmd, void *data) {
  switch (cmd) {
  case RETRO_ENVIRONMENT_GET_SYSTEM_DIRECTORY:
  case RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
    *(const char **)data = sys_dir;
    return true;
  case RETRO_ENVIRONMENT_SET_PIXEL_FORMAT:
    pixel_format = *(const unsigned *)data;
    return true;
  case RETRO_ENVIRONMENT_GET_LOG_INTERFACE:
    ((struct retro_log_callback *)data)->log = log_cb;
    return true;
  case RETRO_ENVIRONMENT_GET_CAN_DUPE:
    *(bool *)data = true;
    return true;
  case RETRO_ENVIRONMENT_GET_CORE_OPTIONS_VERSION:
    *(unsigned *)data = 2;
    return true;
  case RETRO_ENVIRONMENT_SET_VARIABLES:
    opts_v0(data);
    return true;
  case RETRO_ENVIRONMENT_SET_CORE_OPTIONS:
    opts_v1(data);
    return true;
  case RETRO_ENVIRONMENT_SET_CORE_OPTIONS_INTL:
    opts_v1(((const struct retro_core_options_intl *)data)->us);
    return true;
  case RETRO_ENVIRONMENT_SET_CORE_OPTIONS_V2:
    opts_v2(data);
    return true;
  case RETRO_ENVIRONMENT_SET_CORE_OPTIONS_V2_INTL:
    opts_v2(((const struct retro_core_options_v2_intl *)data)->us);
    return true;
  case RETRO_ENVIRONMENT_GET_VARIABLE: {
    struct retro_variable *var = data;
    for (unsigned i = 0; i < n_opts; i++)
      if (var->key && !strcmp(opts[i].key, var->key)) {
        var->value = opts[i].value;
        return true;
      }
    var->value = NULL;
    return false;
  }
  case RETRO_ENVIRONMENT_GET_VARIABLE_UPDATE:
    *(bool *)data = false;
    return true;
  case RETRO_ENVIRONMENT_SET_HW_RENDER: {
    struct retro_hw_render_callback *cb = data;
    hw_asked = true;
    if (getenv("CORETEST_NO_GPU") || hw_on)
      return false;
    switch (cb->context_type) {
    case RETRO_HW_CONTEXT_OPENGL:
    case RETRO_HW_CONTEXT_OPENGL_CORE:
    case RETRO_HW_CONTEXT_OPENGLES2:
    case RETRO_HW_CONTEXT_OPENGLES3:
    case RETRO_HW_CONTEXT_OPENGLES_VERSION:
      break;
    default:
      return false; /* Vulkan e outros: o core tenta o próximo */
    }
    hw = *cb;
    if (!egl_init()) {
      fprintf(stderr, "sem contexto OpenGL (EGL surfaceless)\n");
      return false;
    }
    cb->get_current_framebuffer = hw_framebuffer;
    cb->get_proc_address = hw_proc;
    hw = *cb;
    hw_on = true;
    return true;
  }
  default:
    return false;
  }
}

/* pixel -> RGB24, nos três formatos do libretro */
static void to_rgb(const uint8_t *row, unsigned x, uint8_t *out) {
  if (pixel_format == RETRO_PIXEL_FORMAT_XRGB8888) {
    uint32_t p = ((const uint32_t *)row)[x];
    out[0] = p >> 16, out[1] = p >> 8, out[2] = p;
  } else if (pixel_format == RETRO_PIXEL_FORMAT_RGB565) {
    uint16_t p = ((const uint16_t *)row)[x];
    out[0] = (p >> 11) << 3, out[1] = ((p >> 5) & 63) << 2, out[2] = (p & 31) << 3;
  } else {
    uint16_t p = ((const uint16_t *)row)[x];
    out[0] = ((p >> 10) & 31) << 3, out[1] = ((p >> 5) & 31) << 3, out[2] = (p & 31) << 3;
  }
}

/* registra um frame em RGB24 (fica com o buffer) */
static void take_frame(uint8_t *rgb, unsigned w, unsigned h) {
  uint64_t hash = 1469598103934665603ULL; /* FNV-1a */
  bool blank = true;
  size_t size = (size_t)w * h * 3;
  for (size_t i = 0; i < size; i++) {
    hash = (hash ^ rgb[i]) * 1099511628211ULL;
    if (rgb[i] != rgb[i % 3])
      blank = false;
  }
  video_frames++;
  if (hash != last_hash)
    distinct++;
  blank_streak = blank ? blank_streak + 1 : 0;
  last_hash = hash, last_w = w, last_h = h;
  free(last_rgb);
  last_rgb = rgb;
}

static void video_refresh(const void *data, unsigned w, unsigned h, size_t pitch) {
  if (!data || !w || !h)
    return; /* frame repetido (dupe) */
  uint8_t *rgb = malloc((size_t)w * h * 3);
  if (data == RETRO_HW_FRAME_BUFFER_VALID) {
    if (!hw_on) {
      free(rgb);
      return;
    }
    uint8_t *rgba = malloc((size_t)w * h * 4);
    gl_bind_framebuffer(GL_FRAMEBUFFER, fbo);
    gl_pixel_storei(GL_PACK_ALIGNMENT, 1);
    gl_read_pixels(0, 0, (GLsizei)w, (GLsizei)h, GL_RGBA, GL_UNSIGNED_BYTE, rgba);
    for (unsigned y = 0; y < h; y++) {
      /* OpenGL começa embaixo: vira a imagem, a não ser que o core diga o contrário */
      unsigned src = hw.bottom_left_origin ? h - 1 - y : y;
      for (unsigned x = 0; x < w; x++)
        memcpy(rgb + ((size_t)y * w + x) * 3, rgba + ((size_t)src * w + x) * 4, 3);
    }
    free(rgba);
  } else {
    for (unsigned y = 0; y < h; y++)
      for (unsigned x = 0; x < w; x++)
        to_rgb((const uint8_t *)data + y * pitch, x, rgb + ((size_t)y * w + x) * 3);
  }
  take_frame(rgb, w, h);
}

static void audio_sample(int16_t l, int16_t r) { (void)l, (void)r; }
static size_t audio_batch(const int16_t *d, size_t n) { (void)d; return n; }
static void input_poll(void) {}

/* START por 6 frames a cada 120, a partir do 90: passa abertura e menu */
static int16_t input_state(unsigned port, unsigned dev, unsigned idx, unsigned id) {
  (void)idx;
  if (port || dev != RETRO_DEVICE_JOYPAD)
    return 0;
  bool press = frame_no >= 90 && frame_no % 120 < 6;
  if (id == RETRO_DEVICE_ID_JOYPAD_START || id == RETRO_DEVICE_ID_JOYPAD_A)
    return press;
  return 0;
}

#define SYM(name)                                                              \
  name = dlsym(core, #name);                                                   \
  if (!name) {                                                                 \
    fprintf(stderr, "símbolo %s não existe no core\n", #name);                \
    return 2;                                                                  \
  }

int main(int argc, char **argv) {
  if (argc != 6) {
    fprintf(stderr, "uso: coretest <core.so> <jogo> <frames> <dir> <saida.rgb>\n");
    return 2;
  }
  const char *core_path = argv[1], *rom = argv[2], *out = argv[5];
  unsigned frames = (unsigned)atoi(argv[3]);
  sys_dir = argv[4];

  void *core = dlopen(core_path, RTLD_NOW | RTLD_LOCAL);
  if (!core) {
    fprintf(stderr, "dlopen: %s\n", dlerror());
    return 2;
  }
  void (*retro_set_environment)(retro_environment_t);
  void (*retro_set_video_refresh)(retro_video_refresh_t);
  void (*retro_set_audio_sample)(retro_audio_sample_t);
  void (*retro_set_audio_sample_batch)(retro_audio_sample_batch_t);
  void (*retro_set_input_poll)(retro_input_poll_t);
  void (*retro_set_input_state)(retro_input_state_t);
  void (*retro_init)(void);
  void (*retro_deinit)(void);
  unsigned (*retro_api_version)(void);
  void (*retro_get_system_info)(struct retro_system_info *);
  void (*retro_get_system_av_info)(struct retro_system_av_info *);
  bool (*retro_load_game)(const struct retro_game_info *);
  void (*retro_unload_game)(void);
  void (*retro_run)(void);
  SYM(retro_set_environment) SYM(retro_set_video_refresh)
  SYM(retro_set_audio_sample) SYM(retro_set_audio_sample_batch)
  SYM(retro_set_input_poll) SYM(retro_set_input_state) SYM(retro_init)
  SYM(retro_deinit) SYM(retro_api_version) SYM(retro_get_system_info)
  SYM(retro_get_system_av_info) SYM(retro_load_game) SYM(retro_unload_game)
  SYM(retro_run)

  struct retro_system_info info = {0};
  retro_get_system_info(&info);
  /* ordem do RetroArch: environment, init, o resto (o Mesen depende disso) */
  retro_set_environment(environment);
  retro_init();
  retro_set_video_refresh(video_refresh);
  retro_set_audio_sample(audio_sample);
  retro_set_audio_sample_batch(audio_batch);
  retro_set_input_poll(input_poll);
  retro_set_input_state(input_state);

  struct retro_game_info game = {.path = rom};
  void *buf = NULL;
  if (!info.need_fullpath) { /* core quer o jogo na memória */
    FILE *f = fopen(rom, "rb");
    if (!f) {
      perror(rom);
      return 2;
    }
    fseek(f, 0, SEEK_END);
    game.size = (size_t)ftell(f);
    rewind(f);
    buf = malloc(game.size);
    if (fread(buf, 1, game.size, f) != game.size) {
      fprintf(stderr, "leitura incompleta de %s\n", rom);
      return 2;
    }
    fclose(f);
    game.data = buf;
  }

  bool loaded = retro_load_game(&game);
  struct retro_system_av_info av = {0};
  if (loaded) {
    retro_get_system_av_info(&av);
    if (hw_on) {
      if (!fbo_init(av.geometry.max_width, av.geometry.max_height)) {
        fprintf(stderr, "framebuffer OpenGL incompleto\n");
        return 2;
      }
      if (hw.context_reset)
        hw.context_reset();
    }
    for (frame_no = 0; frame_no < frames; frame_no++)
      retro_run();
    if (hw_on && hw.context_destroy)
      hw.context_destroy();
    retro_unload_game();
  }
  retro_deinit();

  if (last_rgb) {
    FILE *f = fopen(out, "wb");
    if (f) {
      fwrite(last_rgb, 3, (size_t)last_w * last_h, f);
      fclose(f);
    }
  }
  printf("{\"core\":\"%s\",\"version\":\"%s\",\"api\":%u,\"loaded\":%s,"
         "\"hw_asked\":%s,\"gpu\":%s,\"frames\":%u,\"video_frames\":%u,"
         "\"distinct\":%u,\"blank_at_end\":%s,\"width\":%u,\"height\":%u,"
         "\"fps\":%.3f,\"hash\":\"%016llx\"}\n",
         info.library_name, info.library_version, retro_api_version(),
         loaded ? "true" : "false", hw_asked ? "true" : "false",
         hw_on ? "true" : "false", loaded ? frames : 0, video_frames, distinct,
         blank_streak > 0 ? "true" : "false", last_w, last_h, av.timing.fps,
         (unsigned long long)last_hash);
  free(buf);
  free(last_rgb);
  if (egl_dpy != EGL_NO_DISPLAY) {
    eglMakeCurrent(egl_dpy, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
    eglTerminate(egl_dpy);
  }
  return loaded ? 0 : 1;
}
