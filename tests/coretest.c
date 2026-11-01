/*
 * coretest: frontend libretro mínimo, sem janela, pra teste automático.
 *
 * Carrega um core, abre o jogo, roda N frames apertando START de vez em quando
 * (sempre igual, então o resultado é repetível) e imprime um JSON com o que viu:
 * quantos frames vieram, quantos diferentes, se tem imagem, hash do último frame.
 * O último frame vai em RGB cru pro arquivo de saída (o check.py vira PNG).
 *
 * Uso: coretest <core.so> <jogo> <frames> <dir_sistema> <saida.rgb>
 */
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

static void log_cb(enum retro_log_level level, const char *fmt, ...) {
  if (level < RETRO_LOG_WARN && !getenv("CORETEST_VERBOSE"))
    return;
  va_list ap;
  va_start(ap, fmt);
  vfprintf(stderr, fmt, ap);
  va_end(ap);
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
  case RETRO_ENVIRONMENT_SET_HW_RENDER:
    hw_asked = true; /* sem GPU aqui: o core cai pro render por software, se tiver */
    return false;
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

static void video_refresh(const void *data, unsigned w, unsigned h, size_t pitch) {
  if (!data || data == RETRO_HW_FRAME_BUFFER_VALID || !w || !h)
    return; /* frame repetido (dupe) ou de GPU: nada pra olhar */
  video_frames++;
  uint8_t *rgb = malloc((size_t)w * h * 3);
  uint64_t hash = 1469598103934665603ULL; /* FNV-1a */
  bool blank = true;
  for (unsigned y = 0; y < h; y++) {
    const uint8_t *row = (const uint8_t *)data + y * pitch;
    for (unsigned x = 0; x < w; x++) {
      uint8_t *px = rgb + ((size_t)y * w + x) * 3;
      to_rgb(row, x, px);
      for (int c = 0; c < 3; c++)
        hash = (hash ^ px[c]) * 1099511628211ULL;
      if (px[0] != rgb[0] || px[1] != rgb[1] || px[2] != rgb[2])
        blank = false;
    }
  }
  if (hash != last_hash)
    distinct++;
  blank_streak = blank ? blank_streak + 1 : 0;
  last_hash = hash, last_w = w, last_h = h;
  free(last_rgb);
  last_rgb = rgb;
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
    for (frame_no = 0; frame_no < frames; frame_no++)
      retro_run();
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
         "\"hw_asked\":%s,\"frames\":%u,\"video_frames\":%u,\"distinct\":%u,"
         "\"blank_at_end\":%s,\"width\":%u,\"height\":%u,\"fps\":%.3f,"
         "\"hash\":\"%016llx\"}\n",
         info.library_name, info.library_version, retro_api_version(),
         loaded ? "true" : "false", hw_asked ? "true" : "false", loaded ? frames : 0,
         video_frames, distinct, blank_streak > 0 ? "true" : "false", last_w, last_h,
         av.timing.fps, (unsigned long long)last_hash);
  free(buf);
  free(last_rgb);
  return loaded ? 0 : 1;
}
