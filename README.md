# arkflix-games

ES-DE, RetroArch (8 cores), PPSSPP e PCSX2 compilados do fonte, pra mexer no
código e jogar a biblioteca do arkflix com a versão modificada.

O ES-DE e os emuladores instalados (AppImage e Flatpaks, pelo genesis)
continuam iguais; isto aqui roda ao lado deles.

## Instalar

```bash
curl -fsSL https://raw.githubusercontent.com/stanleygomes/arkflix-games/master/install.sh | bash
```

Ubuntu/Debian x86_64. Pede sudo (pacotes do apt), clona em
`~/.local/share/arkflix-games`, baixa os binários do último release (sem
release com binários, compila: ~40 min) e cria o comando `arkflix-games`.
Rodar de novo atualiza. Opções vão antes do `bash`: `... | ARKFLIX_GAMES_DIR=~/x bash`
(outra pasta) ou `... | ARKFLIX_GAMES_BUILD=1 bash` (compila mesmo com release).

## Começo (desenvolvimento)

```bash
make deps     # pacotes de compilação (apt, pede sudo). Uma vez só
make fetch    # baixa os fontes (submodules rasos)
make build    # compila tudo em build/ (~20-40 min na primeira vez)
make run      # abre o ES-DE compilado (antes: make arkflix-mount no solomon)
make check    # valida que nada quebrou
```

Compilar só um: `make es-de`, `make retroarch`, `make ppsspp`, `make pcsx2`,
`make cores` ou `make core-snes9x`.

## O que tem

| Pasta | Projeto | Versão | Licença |
| --- | --- | --- | --- |
| `src/es-de` | ES-DE (frontend) | v3.5.0 | MIT |
| `src/retroarch` | RetroArch | v1.22.2 | GPLv3 |
| `src/ppsspp` | PPSSPP (PSP) | v1.20.4 | GPLv2+ |
| `src/pcsx2` | PCSX2 (PS2) | v2.8.2 | GPLv3 |
| `src/cores/mesen` | NES | master | GPLv3 |
| `src/cores/snes9x` | SNES | master | não comercial |
| `src/cores/gambatte` | GB/GBC | master | GPLv2 |
| `src/cores/genesis_plus_gx` | Mega Drive | master | não comercial |
| `src/cores/mupen64plus_next` | N64 | develop | GPLv2 |
| `src/cores/melondsds` | DS | v1.4.0 | GPLv3 |
| `src/cores/mednafen_psx` | PS1 | master | GPLv2 |
| `src/cores/fbneo` | Arcade | master | não comercial |
| `src/retroarch-*`, `src/libretro-core-info` | Menus, controles, info dos cores | master | — |
| `src/deps/rapidyaml` | Lib do PCSX2 que o apt não tem | v0.12.1 | MIT |

Uso pessoal: modificar é livre em todos. As licenças só pesam se distribuir.
O melonDS DS baixa o melonDS e algumas libs no `cmake` (FetchContent).

## Como o `make run` funciona

- **ES-DE local** em `home/ES-DE` (`ESDE_APPDATA_DIR`): tudo é link pro
  `~/ES-DE` (gamelists, settings, mídia, o script de backup dos saves), menos
  `custom_systems/es_find_rules.xml`, que aponta pros binários de `build/bin`.
- **RetroArch** com config própria (`home/retroarch.cfg`), porque a do Flatpak
  aponta pra `/app/...`. Saves, states e BIOS ficam nas pastas do Flatpak.
- **PPSSPP e PCSX2** usam a pasta do Flatpak inteira (`XDG_CONFIG_HOME`):
  mesma versão, então config, controles, BIOS e saves são os mesmos.

Resultado: o backup dos saves do solomon (`make arkflix-saves`) vale pros
dois, e jogar no Flatpak ou na versão compilada dá no mesmo save.

## Mexer no código

Edite direto em `src/<projeto>`, recompile o alvo e rode `make check`.

```bash
make status        # versão de cada fonte e quantos arquivos você mudou
make patch-save    # suas mudanças viram patches/<projeto>.patch (commite isso)
make patch-apply   # reaplica os patches (o make fetch já faz)
```

Os submodules apontam pro upstream; suas mudanças vivem nos patches, não em
commits dentro do submodule. Atualizar uma versão: `git -C src/x fetch --depth 1
origin tag vX && git -C src/x checkout vX`, `make patch-apply`, `make build check`.

## Validar (`make check`)

| Alvo | O que testa |
| --- | --- |
| `check-cores` | Cada core roda um jogo seu por ~15 s sem janela (`tests/coretest.c`), apertando START. Falha se não carregar, travar, der crash, ficar com tela de uma cor só ou imagem parada. Também roda o RetroArch inteiro em modo null por 300 frames |
| `check-es-de` | Versão do ES-DE e se todo caminho das regras de busca existe |
| `check-ppsspp` | Suíte oficial (pspautotests) e testes unitários do PPSSPP |
| `check-pcsx2` | Testes unitários do PCSX2 |

Os jogos vêm do mount (`~/Games/arkflix/roms`, ou `ARKFLIX_ROMS=`). Sem mount
ou sem BIOS, o caso é pulado com o motivo (PS1 precisa de `scph5501.bin` em
`~/.var/app/org.libretro.RetroArch/config/retroarch/system`).

Cada caso salva o último frame em `build/check/<caso>.png` e compara o hash com
`tests/baseline.json`. Diferente = ⚠️ "mudou" (não é erro: pode ser a sua
mudança). Conferiu o PNG e está certo? `make check-update` grava o novo
baseline. Só alguns casos: `make check-cores CASOS="nes snes"`.

## Release

Automático: a cada push no `master`, o GitHub Action (`.github/workflows/release.yml`)
roda `bin/release.sh --publish`, que lê os commits desde a última tag e:

| Commits desde a última tag | Versão |
| --- | --- |
| algum `tipo!:` ou linha `BREAKING CHANGE: ...` no corpo | major (`v1.4.2` → `v2.0.0`) |
| algum `feat` | minor (`v1.4.2` → `v1.5.0`) |
| algum `fix` ou `perf` | patch (`v1.4.2` → `v1.4.3`) |
| só `docs`, `chore`, `test`, `ci`, `refactor`... | nada |

Cria a tag `vX.Y.Z` e a release com o changelog agrupado (Breaking, Novidades,
Correções, Performance, Outros).

```bash
make release-notes      # prévia: próxima versão e changelog, sem criar nada
make release-binaries   # anexa os binários na release da tag do HEAD (depois do CI)
make release-install    # baixa os binários do último release que tiver (sem gh)
make release-install VERSION=v0.2.0
```

Os binários saem do seu PC, não do CI: o `release-binaries` exige o HEAD com
tag, roda o `make check` (que usa os seus jogos) e só então anexa. O pacote
tem só o que o `make run` usa, compilado pra Ubuntu, e roda em qualquer pasta
(os lançadores são gerados na instalação). O código de cada binário é o da tag
(submodules + `patches/`), o que cumpre as licenças GPL.
