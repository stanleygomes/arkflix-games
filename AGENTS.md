# arkflix-games: instruções pra agentes

Leia o [README](README.md) antes. Resumo do que importa:

- Só o app é código deste repo. Os emuladores são os Flatpaks do Flathub
  (RetroArch, PPSSPP, PCSX2) e os cores vêm do buildbot do libretro
  (`make emulators`). Não volte a compilar emulador aqui.
- Build em `build/`, fora do git.
- Não mexa nas pastas dos Flatpaks (`~/.var/app/...`) nem no `~/ES-DE`: são os
  saves e a config reais do dono (o app só lê o `~/ES-DE`). Exceções: o
  `make emulators` instala cores que faltam na pasta de cores do RetroArch, e o
  `internal/emucfg` ajusta só os atalhos de menu/saída do PPSSPP e do PCSX2
  (com backup). No RetroArch, nada vai no arquivo dele: tudo por `--appendconfig`,
  com `config_save_on_exit = "false"` (senão ele grava o acrescentado por cima).
- `make deps` usa sudo: peça pro dono rodar (`! make deps`).

## App (app/)

- Backend Go em `app/internal`, interface Svelte 5 + TypeScript em `app/web`.
  Métodos exportados de `core.Service` são a API (Wails no desktop, HTTP no web):
  não exporte método que não deva ser chamado pela interface.
- Fonte de jogos, metadados e saves entram por interface (`GameSource`,
  `MetaProvider`, `SaveStore`). O servidor RomM fica em `internal/romm` (cliente)
  e `internal/core/remote.go`; não acople nada remoto nos pacotes locais.
- Nunca guarde senha: o login do RomM vira um token de dispositivo. Teste com o
  RomM falso (`fixture.FakeRomM`, também em `/fake-romm` no `--fixture --test`).
- Mudou o app? `make check-app` (Go com race, svelte-check, Vitest, Playwright
  com axe). Tela nova ou mudada = teste de ponta a ponta + `axe(page, ...)` nela.
- Acessibilidade é requisito: contraste AA (paletas testadas no Go), nome
  acessível contendo o texto visível, foco sempre visível, tudo navegável por
  controle e teclado (`data-nav`).
- Dependência nova (Go ou npm): versão exata, publicada há pelo menos duas semanas.

## Git: commit + push sempre

Toda iteração que mudar arquivo termina com commit e push. Sem perguntar.

- Antes de responder: `git status`, `git add` só do que você mudou, commit,
  `git push` no `master`. Nada de `git add -A` às cegas.
- Conventional Commits em português, imperativo, 72 colunas por linha, sem emoji,
  sem Co-Authored-By. Um commit por mudança lógica.
- Push falhou? Diga ao dono. Nunca `--force`.
- Versão e release são automáticos (CI, pelo tipo do commit): escolha o tipo
  certo, ele decide a versão. `feat`/`fix` geram release; `docs`/`chore` não.
- `make release-binaries` publica binários no GitHub: só quando o dono pedir.
