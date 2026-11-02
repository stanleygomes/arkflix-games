#!/usr/bin/env bash
# Próxima versão (semver) a partir dos Conventional Commits desde a última tag.
#   feat -> minor | fix, perf -> patch | tipo! ou footer "BREAKING CHANGE:" -> major
#   docs, chore, test, ci, refactor, style, build sozinhos: sem release
#
# Uso: bin/release.sh            # mostra a versão e o changelog, não cria nada
#      bin/release.sh --publish  # cria a tag vX.Y.Z e a release no GitHub (CI)
set -euo pipefail
cd "$(dirname "$0")/.."

last=$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || true)
range=${last:+$last..}HEAD
IFS=. read -r major minor patch <<<"${last#v}"
major=${major:-0} minor=${minor:-0} patch=${patch:-0}

declare -A section=([breaking]="" [feat]="" [fix]="" [perf]="" [other]="")
bump=0 # 0 nada, 1 patch, 2 minor, 3 major
re='^([a-z]+)(\([^)]*\))?(!)?: (.+)$'

while IFS=$'\x1f' read -r -d $'\x1e' hash subject body; do
  hash=${hash//$'\n'/}
  [[ -z $hash ]] && continue
  line="- ${subject} (${hash:0:7})"
  if [[ $subject =~ $re ]]; then
    type=${BASH_REMATCH[1]}
    # footer da spec: linha começando com "BREAKING CHANGE:", não a frase no meio
    if [[ -n ${BASH_REMATCH[3]} ]] || grep -qE '^BREAKING[ -]CHANGE: ' <<<"$body"; then
      section[breaking]+="$line"$'\n'
      bump=3
    fi
    case $type in
      feat) section[feat]+="$line"$'\n'; ((bump < 2)) && bump=2 ;;
      fix | perf) section[$type]+="$line"$'\n'; ((bump < 1)) && bump=1 ;;
      *) section[other]+="$line"$'\n' ;;
    esac
  else
    section[other]+="$line"$'\n'
  fi
done < <(git log --no-merges --format='%H%x1f%s%x1f%b%x1e' "$range")

if ((bump == 0)); then
  echo "nada a lançar desde ${last:-o começo} (só docs/chore/test/ci...)"
  exit 0
fi
case $bump in
  3) version="v$((major + 1)).0.0" ;;
  2) version="v${major}.$((minor + 1)).0" ;;
  1) version="v${major}.${minor}.$((patch + 1))" ;;
esac

notes=$(mktemp)
trap 'rm -f "$notes"' EXIT
{
  for key in breaking feat fix perf other; do
    [[ -z ${section[$key]} ]] && continue
    case $key in
      breaking) echo "### ⚠️ Breaking changes" ;;
      feat) echo "### Novidades" ;;
      fix) echo "### Correções" ;;
      perf) echo "### Performance" ;;
      other) echo "### Outros" ;;
    esac
    printf '%s\n' "${section[$key]}"
  done
  echo "### Downloads"
  echo
  echo "| Sistema | Arquivo |"
  echo "| --- | --- |"
  echo "| Windows 10/11 (64 bits) | \`arkflix-games-$version-windows-x86_64-setup.exe\` (instalador) ou \`.zip\` (portátil) |"
  echo "| macOS 11+ (Intel e Apple Silicon) | \`arkflix-games-$version-macos-universal.dmg\` |"
  echo "| Linux x86_64 | \`arkflix-games-$version-linux-x86_64.tar.gz\`, ou o instalador de um comando do README |"
  echo
  echo "Os emuladores (RetroArch, PPSSPP, PCSX2) são instalados à parte; veja o README."
  echo
  [[ -n $last ]] && echo "Desde ${last}."
} >"$notes"

echo "versão: ${last:-nenhuma} -> $version"
echo
cat "$notes"

[[ ${1:-} == --publish ]] || exit 0
git tag -a "$version" -m "release $version"
git push origin "$version"
gh release create "$version" --title "$version" --notes-file "$notes" --verify-tag
echo "✅ release $version publicado"
# o CI usa a versão pra anexar os instaladores
if [[ -n ${GITHUB_OUTPUT:-} ]]; then echo "tag=$version" >>"$GITHUB_OUTPUT"; fi
