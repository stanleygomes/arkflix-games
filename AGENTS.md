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
- Commit: Conventional Commits em português, 72 colunas, sem Co-Authored-By.
  Um commit por mudança lógica.
