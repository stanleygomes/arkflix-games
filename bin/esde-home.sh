#!/usr/bin/env bash
# Monta a pasta do ES-DE local: tudo é link pro ~/ES-DE (gamelists, settings,
# scripts do backup de saves, mídia), menos as regras de busca dos emuladores.
set -euo pipefail
src=$1 dst=$2 bin=$3 cores=$4
here=$(cd "$(dirname "$0")/.." && pwd)

mkdir -p "$src" "$dst/custom_systems" "$dst/logs"
for item in "$src"/*; do
  name=${item##*/}
  case $name in custom_systems | logs) continue ;; esac
  [ -e "$dst/$name" ] || ln -s "$item" "$dst/$name"
done
for item in "$src"/custom_systems/*; do
  [ -e "$item" ] || continue
  name=${item##*/}
  [ "$name" = es_find_rules.xml ] && continue
  [ -e "$dst/custom_systems/$name" ] || ln -s "$item" "$dst/custom_systems/$name"
done
sed -e "s|@BIN@|$bin|g" -e "s|@CORES@|$cores|g" \
  "$here/config/es_find_rules.xml.in" > "$dst/custom_systems/es_find_rules.xml"
echo "ES-DE local em $dst"
