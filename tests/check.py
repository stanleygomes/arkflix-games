"""
check: roda cada core com um jogo de verdade (do mount do arkflix), sem janela,
e diz se quebrou. Também roda o RetroArch inteiro em modo null com um core.

Quebrou = não carregou, travou, estourou o tempo, nenhum frame, tela preta no
fim ou imagem parada (o teste aperta START, a tela tem que mudar).
Mudou = o último frame ficou diferente do baseline (tests/baseline.json). Não
é erro: pode ser a sua mudança. Confira o PNG em build/check/ e rode
`make check-update` se estiver certo.

Uso: python3 tests/check.py [--update] [caso ...]
"""

import json
import os
import struct
import subprocess
import sys
import tempfile
import zipfile
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BUILD = ROOT / "build"
CORES = BUILD / "prefix" / "lib" / "libretro"
OUT = BUILD / "check"
BASELINE = ROOT / "tests" / "baseline.json"
ROMS = Path(os.environ.get("ARKFLIX_ROMS", Path.home() / "Games/arkflix/roms"))
SYSTEM = Path.home() / ".var/app/org.libretro.RetroArch/config/retroarch/system"
TIMEOUT = 240

# caso: (core, jogo relativo a roms/, frames, BIOS aceitas: precisa de uma)
CASES = {
  "nes": ("mesen", "nes/Super Mario Bros. 3.zip", 900, []),
  "snes": ("snes9x", "snes/Super Mario World (U) [!].zip", 900, []),
  "gbc": ("gambatte", "gbc/Tetris DX.zip", 900, []),
  "genesis": (
    "genesis_plus_gx",
    "genesis/Sonic The Hedgehog 2 (World) (Rev A).zip",
    900,
    [],
  ),
  "n64": ("mupen64plus_next", "n64/Super Mario 64.zip", 900, []),
  "nds": ("melondsds", "nds/New Super Mario Bros..zip", 900, []),
  "psx": (
    "mednafen_psx",
    "psx/Crash Bandicoot (USA)",
    900,
    ["scph5501.bin", "scph5500.bin", "scph5502.bin", "scph1001.bin", "scph7001.bin"],
  ),
  "arcade": ("fbneo", "arcade/mspacman.zip", 900, []),
}


def png(path: Path, rgb: bytes, w: int, h: int) -> None:
  rows = b"".join(b"\0" + rgb[y * w * 3 : (y + 1) * w * 3] for y in range(h))

  def chunk(kind: bytes, data: bytes) -> bytes:
    body = kind + data
    return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

  head = struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)
  path.write_bytes(
    b"\x89PNG\r\n\x1a\n"
    + chunk(b"IHDR", head)
    + chunk(b"IDAT", zlib.compress(rows, 6))
    + chunk(b"IEND", b"")
  )


def game_file(core: str, src: Path, tmp: Path) -> Path:
  """Zip vira o arquivo de dentro (o fbneo quer o zip). Pasta de disco vira o .cue."""
  if src.is_dir():
    for ext in (".m3u", ".cue", ".chd", ".pbp"):
      found = sorted(src.glob(f"*{ext}"))
      if found:
        return found[0]
    raise FileNotFoundError(f"sem .cue/.chd em {src}")
  if src.suffix.lower() == ".zip" and core != "fbneo":
    with zipfile.ZipFile(src) as z:
      member = max(z.infolist(), key=lambda i: i.file_size)
      return Path(z.extract(member, tmp))
  return src


def run_case(name: str, baseline: dict) -> tuple[str, str, dict]:
  core, rel, frames, bios = CASES[name]
  so = CORES / f"{core}_libretro.so"
  if not so.is_file():
    return "falhou", f"{so.name} não compilado (make core-{core})", {}
  src = ROMS / rel
  if not src.exists():
    return "pulado", f"jogo não achado: {src} (mount montado?)", {}
  with tempfile.TemporaryDirectory(prefix=f"check-{name}-") as tmp:
    tmp = Path(tmp)
    sysdir = tmp / "system"
    sysdir.mkdir()
    if bios:
      have = [b for b in bios if (SYSTEM / b).is_file()]
      if not have:
        return "pulado", f"sem BIOS em {SYSTEM} ({bios[0]} ou outra)", {}
      for b in have:
        (sysdir / b).symlink_to(SYSTEM / b)
    try:
      game = game_file(core, src, tmp)
    except (OSError, zipfile.BadZipFile) as error:
      return "falhou", f"jogo ilegível: {error}", {}
    raw = OUT / f"{name}.rgb"
    cmd = [str(BUILD / "coretest"), str(so), str(game), str(frames), str(sysdir), str(raw)]
    try:
      res = subprocess.run(cmd, capture_output=True, text=True, timeout=TIMEOUT)
    except subprocess.TimeoutExpired:
      return "falhou", f"travou: passou de {TIMEOUT}s", {}
  (OUT / f"{name}.log").write_text(res.stderr)
  if res.returncode < 0:
    return "falhou", f"crash (sinal {-res.returncode}), ver build/check/{name}.log", {}
  try:
    info = json.loads(res.stdout.strip().splitlines()[-1])
  except (IndexError, json.JSONDecodeError):
    return "falhou", f"sem resultado (exit {res.returncode}), ver {name}.log", {}
  if raw.is_file() and info["width"]:
    png(OUT / f"{name}.png", raw.read_bytes(), info["width"], info["height"])
    raw.unlink()

  if not info["loaded"]:
    return "falhou", "o core recusou o jogo", info
  if info["video_frames"] == 0:
    why = " (pediu GPU e não tem render por software)" if info["hw_asked"] else ""
    return "falhou", f"nenhum frame de vídeo{why}", info
  if info["blank_at_end"]:
    return "falhou", "tela de uma cor só no fim", info
  if info["distinct"] < 2:
    return "falhou", "imagem parada: nunca mudou", info
  old = baseline.get(name)
  if old and old != info["hash"]:
    return "mudou", f"último frame ≠ baseline, ver build/check/{name}.png", info
  return "ok", f"{info['distinct']} frames diferentes de {info['video_frames']}", info


def retroarch_null() -> tuple[str, str]:
  """RetroArch inteiro (frontend + core), sem vídeo nem som, 300 frames."""
  exe = BUILD / "prefix" / "bin" / "retroarch"
  so = CORES / "snes9x_libretro.so"
  src = ROMS / CASES["snes"][1]
  if not exe.is_file() or not so.is_file():
    return "falhou", "retroarch ou snes9x não compilado"
  if not src.exists():
    return "pulado", "jogo do teste não achado (mount montado?)"
  with tempfile.TemporaryDirectory(prefix="check-ra-") as tmp:
    tmp = Path(tmp)
    cfg = tmp / "retroarch.cfg"
    cfg.write_text(
      'video_driver = "null"\naudio_driver = "null"\ninput_driver = "null"\n'
      f'savefile_directory = "{tmp}"\nsavestate_directory = "{tmp}"\n'
      f'system_directory = "{tmp}"\nconfig_save_on_exit = "false"\n'
    )
    game = game_file("snes9x", src, tmp)
    cmd = [str(exe), "--config", str(cfg), "-L", str(so), str(game), "--max-frames=300"]
    try:
      res = subprocess.run(cmd, capture_output=True, text=True, timeout=TIMEOUT)
    except subprocess.TimeoutExpired:
      return "falhou", f"travou: passou de {TIMEOUT}s"
  (OUT / "retroarch.log").write_text(res.stdout + res.stderr)
  if res.returncode != 0:
    return "falhou", f"exit {res.returncode}, ver build/check/retroarch.log"
  return "ok", "abriu, rodou 300 frames e fechou"


def main() -> int:
  args = [a for a in sys.argv[1:] if not a.startswith("--")]
  update = "--update" in sys.argv
  names = args or list(CASES)
  unknown = [n for n in names if n not in CASES and n != "retroarch"]
  if unknown:
    print(f"caso desconhecido: {unknown}. Existem: {list(CASES)} + retroarch")
    return 2
  OUT.mkdir(parents=True, exist_ok=True)
  baseline = json.loads(BASELINE.read_text()) if BASELINE.is_file() else {}
  icon = {"ok": "✅", "mudou": "⚠️ ", "pulado": "⏭️ ", "falhou": "❌"}
  failed = 0
  for name in names:
    if name == "retroarch":
      continue
    status, msg, info = run_case(name, baseline)
    print(f"{icon[status]} {name:<8} {CASES[name][0]:<17} {msg}", flush=True)
    failed += status == "falhou"
    if update and status in ("ok", "mudou"):
      baseline[name] = info["hash"]
  if not args or "retroarch" in args:
    status, msg = retroarch_null()
    print(f"{icon[status]} {'retroarch':<8} {'frontend':<17} {msg}", flush=True)
    failed += status == "falhou"
  if update:
    BASELINE.write_text(json.dumps(baseline, indent=2, sort_keys=True) + "\n")
    print(f"baseline salvo em {BASELINE.relative_to(ROOT)}")
  print(f"prints em {OUT.relative_to(ROOT)}/")
  return 1 if failed else 0


if __name__ == "__main__":
  sys.exit(main())
