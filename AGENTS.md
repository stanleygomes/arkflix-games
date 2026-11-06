# arkflix-games: instruções pra agentes

Leia o [README](README.md) antes. Resumo do que importa:

- Só o app é código deste repo. Os emuladores são os Flatpaks do Flathub
  (RetroArch, PPSSPP, PCSX2) e os cores vêm do buildbot do libretro
  (`make emulators`). Não volte a compilar emulador aqui.
- Build em `build/`, fora do git.
- Não mexa nas pastas dos Flatpaks (`~/.var/app/...`) nem no `~/ES-DE`: são os
  saves e a config reais do dono (o app só lê o `~/ES-DE`). Exceções: o
  `make emulators` e o `internal/emuinstall` (botão "Instalar emuladores")
  instalam emuladores e cores na pasta de cores do RetroArch (no `--fixture`,
  só nas pastas de teste), e o
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
  RomM falso (`fixture.FakeRomM`, também em `/fake-romm` no `--fixture --test`;
  aceita Range e fica lento com `ARKFLIX_FAKE_ROMM_SLOW=<ms>`). A fila de
  downloads fica em `internal/core/downloads.go`.
- Mudou o app? `make check-app` (Go com race, svelte-check, Vitest, Playwright
  com axe). Tela nova ou mudada = teste de ponta a ponta + `axe(page, ...)` nela.
- Acessibilidade é requisito: contraste AA (paletas testadas no Go), nome
  acessível contendo o texto visível, foco sempre visível, tudo navegável por
  controle e teclado (`data-nav`).
- Dependência nova (Go ou npm): versão exata, publicada há pelo menos duas semanas.

## Visual

Segue a skill `retro-design` (Noemi), com estas decisões já tomadas:

- **Monitor CRT no app inteiro** (`components/Crt.svelte`): moldura, cantos,
  bojo, reflexo, grão e linhas que somem no centro. Separação de cor e brilho
  de fósforo só nos títulos (`h1`) e no foco, nunca no texto corrido.
- **Chão synthwave** (`components/Synthwave.svelte`) só no Início, Perfis e
  Boas-vindas (lista `FLOOR` no `App.svelte`). Nunca atrás de formulário.
- **Glitch analógico** (`lib/glitch.ts`): por evento (erro = estática,
  trocar perfil = desmagnetizar, fechar jogo = rolagem, trocar tela =
  chuvisco) e ambiente (títulos a cada 4-9 s, tela a cada 12-20 s).
  Só em títulos (h1/h2/h3, `.pixel`): texto corrido fica limpo.
- **Neobrutalismo**: botão pílula com sombra dura que sobe/afunda, painel como
  janela de RPG 16-bit, um selo "NOVO" por tela, um número gigante (Estatísticas).
- Fontes do projeto mantidas (Atkinson no texto). Só tema escuro. Paleta neon
  por console mantida; sombras duras na cor dela, sem blur.
- Alto contraste desliga tubo, chão e glitch; "reduzir animações" congela tudo.

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
