#!/usr/bin/env bash
# Baixa o pacote de binários de um release (API pública, sem gh nem login) e
# confere o sha256. Sem VERSION: o release mais novo que tiver binários.
# Uso: bin/release-download.sh <dono/repo> [vX.Y.Z] <pasta>
set -euo pipefail
repo=$1 version=$2 out=$3
api="https://api.github.com/repos/$repo/releases"
[[ -n $version ]] && api+="/tags/$version"

json=$(curl -fsSL "$api" 2>/dev/null) || {
  echo "❌ release ${version:-mais novo} não achado em $repo" >&2
  exit 1
}
urls=$(python3 -I -c '
import json, sys
data = json.load(sys.stdin)
for release in data if isinstance(data, list) else [data]:
  assets = {a["name"]: a["browser_download_url"] for a in release["assets"]}
  pkg = [n for n in assets if n.endswith("-linux-x86_64.tar.gz")]
  if pkg and pkg[0] + ".sha256" in assets:
    print(release["tag_name"], assets[pkg[0]], assets[pkg[0] + ".sha256"])
    break
' <<<"$json")
if [[ -z $urls ]]; then
  echo "❌ nenhum release com binários${version:+ em $version} ($repo)" >&2
  exit 1
fi
read -r tag pkg sum <<<"$urls"
rm -rf "$out" && mkdir -p "$out"
echo "baixando $tag..."
curl -fL --progress-bar -o "$out/${pkg##*/}" "$pkg"
curl -fsSL -o "$out/${sum##*/}" "$sum"
(cd "$out" && sha256sum -c --quiet "${sum##*/}")
echo "✅ $tag baixado e conferido"
