# arkflix-games

[![release](https://img.shields.io/github/v/release/stanleygomes/arkflix-games?sort=semver)](https://github.com/stanleygomes/arkflix-games/releases)
[![workflow](https://github.com/stanleygomes/arkflix-games/actions/workflows/release.yml/badge.svg)](https://github.com/stanleygomes/arkflix-games/actions/workflows/release.yml)

Ambiente de emulação para Linux compilado a partir do código-fonte: o frontend
[ES-DE](https://es-de.org), o [RetroArch](https://www.retroarch.com) com oito
cores, o [PPSSPP](https://www.ppsspp.org) e o [PCSX2](https://pcsx2.net). Todos
ficam em versões fixas, com um único `Makefile` para compilar, executar, testar
e publicar.

O objetivo é poder alterar o código de qualquer um desses projetos, validar a
alteração com testes automatizados que rodam jogos de verdade e usar o
resultado no dia a dia, sem afetar as versões já instaladas no sistema.

## Sumário

- [Recursos](#recursos)
- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Uso](#uso)
- [Compilação a partir do código-fonte](#compilação-a-partir-do-código-fonte)
- [Componentes](#componentes)
- [Como funciona](#como-funciona)
- [Desenvolvimento](#desenvolvimento)
- [Testes](#testes)
- [Versionamento e releases](#versionamento-e-releases)
- [Estrutura do repositório](#estrutura-do-repositório)
- [Solução de problemas](#solução-de-problemas)
- [Contribuindo](#contribuindo)
- [Licença](#licença)

## Recursos

- Código-fonte de todos os componentes como submódulos Git, em versões fixas.
- Compilação completa ou por componente, sem instalar nada fora da pasta do
  projeto.
- Convivência com as versões instaladas (Flatpak ou AppImage): mesma pasta de
  saves, BIOS e configuração, sem alterar o frontend instalado.
- Alterações locais guardadas como patches, reaplicáveis sobre novas versões.
- Testes automatizados que executam jogos sem janela e verificam a imagem, além
  das suítes oficiais do PPSSPP e do PCSX2.
- Versão semântica e changelog gerados a partir de Conventional Commits.
- Instalação em um comando, com binários pré-compilados quando disponíveis.

## Requisitos

| Item | Detalhe |
| --- | --- |
| Sistema | Ubuntu 26.04 ou Debian equivalente, x86_64 |
| Disco | cerca de 1 GB com binários pré-compilados; cerca de 12 GB para compilar |
| Ferramentas | `git`, `make`, `curl` e `python3` (o instalador cuida do resto) |
| Permissão | `sudo` para instalar os pacotes do apt |
| Opcional | BIOS do PlayStation e do PlayStation 2, que não são distribuídas aqui |

## Instalação

```bash
curl -fsSL https://raw.githubusercontent.com/stanleygomes/arkflix-games/master/install.sh | bash
```

O instalador:

1. instala as dependências pelo apt;
2. clona o repositório em `~/.local/share/arkflix-games` ou o atualiza, se já
   existir;
3. baixa os binários do release mais recente que os tenha e confere o SHA-256;
   se nenhum release tiver binários, compila tudo, o que leva cerca de 40
   minutos;
4. cria o comando `arkflix-games` em `~/.local/bin`.

Executar o instalador de novo atualiza a instalação. As opções são variáveis de
ambiente, passadas antes do `bash`:

| Variável | Efeito |
| --- | --- |
| `ARKFLIX_GAMES_DIR` | pasta da instalação (padrão: `~/.local/share/arkflix-games`) |
| `ARKFLIX_GAMES_BUILD=1` | compila mesmo que exista release com binários |
| `ARKFLIX_GAMES_NO_DEPS=1` | não instala pacotes do apt |

```bash
curl -fsSL https://raw.githubusercontent.com/stanleygomes/arkflix-games/master/install.sh \
  | ARKFLIX_GAMES_DIR=~/emulacao bash
```

## Uso

```bash
arkflix-games       # após a instalação
make run            # dentro do repositório
```

Na primeira execução, o ES-DE pede a pasta das ROMs, e a escolha fica gravada
nas configurações dele. As BIOS ficam nas pastas usuais de cada emulador (veja
[Como funciona](#como-funciona)).

## Compilação a partir do código-fonte

```bash
git clone https://github.com/stanleygomes/arkflix-games.git
cd arkflix-games
make deps      # pacotes de compilação via apt (requer sudo)
make fetch     # código-fonte dos submódulos, em clone raso
make build     # compila tudo em build/
make check     # executa os testes
make run       # abre o ES-DE compilado
```

Também é possível compilar um componente por vez:

| Alvo | Componente |
| --- | --- |
| `make es-de` | frontend ES-DE |
| `make retroarch` | RetroArch |
| `make cores` | os oito cores; `make core-<nome>` compila um só |
| `make ppsspp` | PPSSPP, incluindo `PPSSPPHeadless` e testes unitários |
| `make pcsx2` | PCSX2 e as dependências dele que não existem no apt |

`make help` lista todos os alvos. A variável `JOBS` controla o paralelismo
(padrão: número de CPUs).

## Componentes

| Caminho | Projeto | Sistema | Versão | Licença |
| --- | --- | --- | --- | --- |
| `src/es-de` | ES-DE | frontend | v3.5.0 | MIT |
| `src/retroarch` | RetroArch | frontend libretro | v1.22.2 | GPL-3.0 |
| `src/ppsspp` | PPSSPP | PlayStation Portable | v1.20.4 | GPL-2.0-or-later |
| `src/pcsx2` | PCSX2 | PlayStation 2 | v2.8.2 | GPL-3.0 |
| `src/cores/mesen` | Mesen | NES | master | GPL-3.0 |
| `src/cores/snes9x` | Snes9x | SNES | master | não comercial |
| `src/cores/gambatte` | Gambatte | Game Boy e Game Boy Color | master | GPL-2.0 |
| `src/cores/genesis_plus_gx` | Genesis Plus GX | Mega Drive | master | não comercial |
| `src/cores/mupen64plus_next` | Mupen64Plus-Next | Nintendo 64 | develop | GPL-2.0 |
| `src/cores/melondsds` | melonDS DS | Nintendo DS | v1.4.0 | GPL-3.0 |
| `src/cores/mednafen_psx` | Beetle PSX | PlayStation | master | GPL-2.0 |
| `src/cores/fbneo` | FinalBurn Neo | Arcade | master | não comercial |
| `src/retroarch-assets` | RetroArch Assets | menus | master | CC-BY-4.0 |
| `src/retroarch-joypad-autoconfig` | Joypad Autoconfig | controles | master | MIT |
| `src/libretro-core-info` | Core Info | metadados dos cores | master | MIT |
| `src/deps/rapidyaml` | rapidyaml | dependência do PCSX2 | v0.12.1 | MIT |

As versões de ES-DE, RetroArch, PPSSPP e PCSX2 acompanham as dos pacotes
Flatpak correspondentes, para manter a configuração e os saves compatíveis. O
melonDS DS baixa o melonDS e algumas bibliotecas durante o `cmake`
(FetchContent), nas versões fixadas pelo próprio projeto.

## Como funciona

O `make run` executa o ES-DE compilado com os emuladores compilados, sem
alterar a instalação existente:

- **ES-DE.** Usa `home/ES-DE` como pasta de dados, por meio de
  `ESDE_APPDATA_DIR`. O conteúdo dessa pasta são links para `~/ES-DE`
  (gamelists, configurações, mídia e scripts), exceto
  `custom_systems/es_find_rules.xml`, que aponta para os binários de
  `build/bin`.
- **RetroArch.** Usa uma configuração própria (`home/retroarch.cfg`), porque a
  do Flatpak referencia caminhos internos do sandbox. Saves, save states e BIOS
  continuam nas pastas do Flatpak.
- **PPSSPP e PCSX2.** Usam a pasta de configuração do Flatpak por meio de
  `XDG_CONFIG_HOME`. Como as versões são as mesmas, configuração, controles,
  BIOS e saves são compartilhados.

| Emulador | Saves e BIOS |
| --- | --- |
| RetroArch | `~/.var/app/org.libretro.RetroArch/config/retroarch/{saves,states,system}` |
| PPSSPP | `~/.var/app/org.ppsspp.PPSSPP/config/ppsspp/PSP` |
| PCSX2 | `~/.var/app/net.pcsx2.PCSX2/config/PCSX2/{memcards,sstates,bios}` |

Na prática, um jogo salvo na versão Flatpak continua na versão compilada, e
vice-versa.

## Desenvolvimento

Edite o código diretamente em `src/<projeto>`, recompile o alvo correspondente
e execute os testes:

```bash
make core-snes9x
make check-cores CASOS=snes
```

Os submódulos apontam para os repositórios originais. Alterações locais não são
commitadas dentro dos submódulos: são exportadas como patches, versionados
neste repositório.

| Comando | Função |
| --- | --- |
| `make status` | versão de cada submódulo e quantidade de arquivos alterados |
| `make patch-save` | exporta as alterações para `patches/<projeto>.patch` |
| `make patch-apply` | reaplica os patches; o `make fetch` já faz isso |

Para atualizar a versão de um componente:

```bash
git -C src/pcsx2 fetch --depth 1 origin tag v2.8.3
git -C src/pcsx2 checkout v2.8.3
make patch-apply pcsx2 check
git add src/pcsx2
git commit -m "build: atualiza PCSX2 para v2.8.3"
```

## Testes

`make check` executa todas as verificações e falha se qualquer uma falhar.

| Alvo | Verificação |
| --- | --- |
| `check-cores` | executa cada core com um jogo, sem janela, por cerca de 15 segundos |
| `check-es-de` | versão do ES-DE e existência de todos os caminhos das regras de busca |
| `check-ppsspp` | suíte oficial [pspautotests](https://github.com/hrydgard/pspautotests) e testes unitários do PPSSPP |
| `check-pcsx2` | testes unitários do PCSX2 |

O `check-cores` usa um frontend libretro mínimo (`tests/coretest.c`) que
carrega o core, abre o jogo e pressiona START em intervalos fixos, o que torna o
resultado reproduzível. Um caso falha quando o core não carrega o jogo, trava,
encerra com erro, não produz imagem, termina com a tela de uma única cor ou
mantém a imagem estática. O teste também executa o RetroArch completo, com
drivers nulos, por 300 frames.

Cada caso salva o último frame em `build/check/<caso>.png` e compara o hash
dele com `tests/baseline.json`. Uma diferença aparece como "mudou", sem falhar
o teste, porque pode ser o efeito esperado da alteração. Depois de conferir a
imagem, `make check-update` grava o novo baseline. `CASOS="nes snes"` restringe
os casos executados.

As ROMs vêm de `~/Games/arkflix/roms` ou de `ARKFLIX_ROMS`, e os nomes dos jogos
de cada caso estão em `tests/check.py`. Casos sem ROM ou sem a BIOS necessária
são pulados, com o motivo informado.

## Versionamento e releases

O projeto segue [Versionamento Semântico](https://semver.org/lang/pt-BR/) e
[Conventional Commits](https://www.conventionalcommits.org/pt-br/). A cada push
no `master`, o workflow `release.yml` executa `bin/release.sh --publish`, que
analisa os commits desde a última tag:

| Commits desde a última tag | Incremento |
| --- | --- |
| `tipo!:` ou rodapé `BREAKING CHANGE:` | major |
| `feat` | minor |
| `fix` ou `perf` | patch |
| apenas `docs`, `chore`, `test`, `ci`, `refactor`, `style` ou `build` | nenhum release |

O release é criado com a tag `vX.Y.Z` e um changelog agrupado por tipo. Os
binários são anexados depois, a partir de uma máquina com as ROMs de teste,
porque a publicação exige que o `make check` passe:

| Comando | Função |
| --- | --- |
| `make release-notes` | mostra a próxima versão e o changelog, sem publicar |
| `make release-binaries` | executa os testes, empacota o build e anexa ao release da tag do HEAD |
| `make release-install` | baixa e verifica os binários do release mais recente que os tenha |

O pacote contém apenas o que o `make run` usa e funciona em qualquer pasta. O
código-fonte de cada binário é o da tag correspondente (submódulos mais
`patches/`).

## Estrutura do repositório

```text
.
├── .github/workflows/  release automático
├── bin/                scripts: versão, download de release, pasta do ES-DE
├── config/             modelos dos lançadores e das configurações
├── patches/            alterações locais sobre os submódulos
├── src/                código-fonte dos componentes (submódulos)
├── tests/              frontend de teste, casos e baseline
├── install.sh          instalador em um comando
└── Makefile            ponto de entrada de todas as tarefas
```

`build/` (compilação) e `home/` (estado local do ES-DE e do RetroArch) são
gerados e ignorados pelo Git.

## Solução de problemas

**O ES-DE informa que não encontrou o emulador ou o core.** O componente não foi
compilado ou os lançadores estão desatualizados. Execute `make build` (ou o
alvo específico) e depois `make check-es-de`, que lista os caminhos ausentes.

**Um caso do `check-cores` foi pulado.** A ROM não está no caminho esperado ou
falta a BIOS. O motivo aparece na saída do teste.

**Jogos de PlayStation não iniciam.** O Beetle PSX exige uma BIOS original, como
a `scph5501.bin`, em
`~/.var/app/org.libretro.RetroArch/config/retroarch/system`.

**O build falha porque não encontra um cabeçalho ou uma biblioteca.** Execute
`make deps` novamente; a lista de pacotes é atualizada junto com o projeto. O
PCSX2 2.8 requer Qt 6.10 ou mais recente, presente no Ubuntu 26.04.

**Logs.** Cada caso do `check-cores` grava `build/check/<caso>.log`. O ES-DE
grava em `home/ES-DE/logs`.

## Contribuindo

1. Faça um fork e crie um branch.
2. Altere o código e exporte as mudanças com `make patch-save`.
3. Execute `make check` e confira as imagens em `build/check/`.
4. Escreva commits no padrão Conventional Commits, em português e no modo
   imperativo, com no máximo 72 caracteres por linha. O tipo do commit define a
   próxima versão.
5. Abra um pull request descrevendo a mudança e o resultado dos testes.

## Licença

Cada componente em `src/` é distribuído sob a própria licença, listada em
[Componentes](#componentes) e no repositório de origem. Snes9x, Genesis Plus GX
e FinalBurn Neo proíbem uso comercial. Quem redistribui binários de componentes
GPL precisa disponibilizar o código-fonte correspondente; neste projeto, é a tag
do release.

Os arquivos próprios deste repositório (Makefile, scripts, testes e
configurações) ainda não têm uma licença definida.

ROMs e BIOS não são distribuídas com este projeto.
