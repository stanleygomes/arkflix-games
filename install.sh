#!/usr/bin/env bash
# Instala o arkflix-games: curl -fsSL <raw>/install.sh | bash
#
# Clona (ou atualiza) o repo, instala as dependências (apt, pede sudo), baixa os
# binários do último release (sem release com binários: compila, ~40 min) e cria
# o comando `arkflix-games`. Rodar de novo = atualizar.
#
# Variáveis: ARKFLIX_GAMES_DIR (padrão ~/.local/share/arkflix-games),
#            ARKFLIX_GAMES_BUILD=1 (compila mesmo tendo release),
#            ARKFLIX_GAMES_NO_DEPS=1 (pula o apt).
set -euo pipefail

repo_url=https://github.com/stanleygomes/arkflix-games.git
dir=${ARKFLIX_GAMES_DIR:-$HOME/.local/share/arkflix-games}
launcher=$HOME/.local/bin/arkflix-games

die() { echo "❌ $*" >&2; exit 1; }
step() { printf '\n\033[1m▶ %s\033[0m\n' "$*"; }

[[ $(uname -sm) == "Linux x86_64" ]] || die "só Linux x86_64"
command -v apt-get >/dev/null || die "precisa de Ubuntu/Debian (apt)"

if [[ -z ${ARKFLIX_GAMES_NO_DEPS:-} ]]; then
  step "pacotes básicos (sudo)"
  sudo apt-get update -qq
  sudo apt-get install -y -qq git make curl ca-certificates python3 >/dev/null
fi

step "código em $dir"
if [[ -d $dir/.git ]]; then
  git -C "$dir" pull -q --ff-only
else
  mkdir -p "$(dirname "$dir")"
  git clone -q --depth 1 "$repo_url" "$dir"
fi
cd "$dir"

if [[ -z ${ARKFLIX_GAMES_NO_DEPS:-} ]]; then
  step "dependências (sudo)"
  make deps
fi

if [[ -z ${ARKFLIX_GAMES_BUILD:-} ]] && make release-install; then
  :
else
  step "sem binários prontos: compilando (demora)"
  make fetch build
fi

step "comando arkflix-games"
mkdir -p "$(dirname "$launcher")"
cat >"$launcher" <<LAUNCHER
#!/bin/sh
# gerado pelo install.sh do arkflix-games
exec make --no-print-directory -C "$dir" run "\$@"
LAUNCHER
chmod +x "$launcher"

echo
echo "✅ instalado. Abrir: arkflix-games"
case :$PATH: in *":$HOME/.local/bin:"*) ;; *) echo "   (~/.local/bin fora do PATH: abra um terminal novo)" ;; esac
