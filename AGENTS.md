# arkflix-games: instruções pra agentes

Leia o [README](README.md) antes. Resumo do que importa:

- Fonte em `src/` (submodules do upstream, rasos, versão travada). Build em
  `build/`, estado local em `home/`: os dois fora do git.
- **Mudou código em `src/`?** Recompile o alvo (`make es-de`, `make core-x`...),
  rode `make check` (ou o `check-*` do que mexeu) e só então diga que funciona.
  Caso "mudou" (⚠️): abra o `build/check/<caso>.png` e confira antes de
  `make check-update`.
- Mudanças em `src/` viram `patches/<projeto>.patch` com `make patch-save`; é o
  patch que vai pro commit. Nunca commite dentro do submodule.
- Não mexa nas pastas dos Flatpaks (`~/.var/app/...`) nem no `~/ES-DE` além do
  que o `make home` já faz: são os saves e a config reais do dono.
- `make deps` usa sudo: peça pro dono rodar (`! make deps`).

## Git: commit + push sempre

Toda iteração que mudar arquivo termina com commit e push. Sem perguntar.

- Antes de responder: `git status`, `git add` só do que você mudou, commit,
  `git push` no `master`. Nada de `git add -A` às cegas.
- Mudou fonte em `src/`? `make patch-save` e commite o `patches/*.patch`, não o
  submodule. Bump de versão de submodule = commit próprio.
- Conventional Commits em português, imperativo, 72 colunas por linha, sem emoji,
  sem Co-Authored-By. Um commit por mudança lógica.
- Push falhou? Diga ao dono. Nunca `--force`.
- Versão e release são automáticos (CI, pelo tipo do commit): escolha o tipo
  certo, ele decide a versão. `feat`/`fix` geram release; `docs`/`chore` não.
- `make release-binaries` publica binários no GitHub: só quando o dono pedir.
