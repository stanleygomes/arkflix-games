#!/usr/bin/env bash
# Empacota o app pra um sistema, em build/dist (o CI anexa na release).
# Uso: bin/package.sh linux|windows|macos <versão>
#   linux:   arkflix-games-<v>-linux-x86_64.tar.gz (o que o install.sh baixa)
#   windows: arkflix-games-<v>-windows-x86_64-setup.exe (NSIS) e .zip portátil
#   macos:   arkflix-games-<v>-macos-universal.dmg (Intel + Apple Silicon)
# A interface (app/web/dist) precisa estar compilada antes (make app-web).
set -euo pipefail
target=${1:?linux|windows|macos} version=${2:?versão}
root=$(cd "$(dirname "$0")/.." && pwd)
app=$root/app dist=$root/build/dist work=$root/build/package/$target
rm -rf "$work" && mkdir -p "$work" "$dist"
[[ -f $app/web/dist/index.html ]] || { echo "❌ interface não compilada: make app-web" >&2; exit 1; }
ldflags="-s -w -X main.version=$version"
name=arkflix-games-$version

checksum() { (cd "$dist" && sha256sum "$1" >"$1.sha256"); }

case $target in
linux)
  mkdir -p "$work/prefix/bin"
  (cd "$app" && CGO_ENABLED=1 go build -tags desktop,production,webkit2_41 -trimpath \
    -ldflags "$ldflags" -o "$work/prefix/bin/arkflix-games" .)
  tar -C "$work" -czf "$dist/$name-linux-x86_64.tar.gz" prefix/bin/arkflix-games
  checksum "$name-linux-x86_64.tar.gz"
  ;;
windows)
  (cd "$app" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags desktop,production -trimpath \
    -ldflags "$ldflags -H windowsgui" -o "$work/arkflix-games.exe" .)
  (cd "$work" && zip -q -9 "$dist/$name-windows-x86_64.zip" arkflix-games.exe)
  checksum "$name-windows-x86_64.zip"
  if command -v makensis >/dev/null; then
    makensis -V2 -DVERSION="${version#v}" -DSRC="$work" -DOUT="$dist/$name-windows-x86_64-setup.exe" \
      "$root/packaging/windows/installer.nsi"
    checksum "$name-windows-x86_64-setup.exe"
  else
    echo "⏭️  sem makensis: só o .zip (o CI gera o instalador)"
  fi
  ;;
macos)
  bundle=$work/arkflix-games.app
  mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
  for arch in arm64 amd64; do
    cc=clang; [[ $arch == amd64 ]] && cc="clang -arch x86_64"
    # o Wails usa UTType (UniformTypeIdentifiers): o CLI dele liga esse framework, aqui é à mão
    (cd "$app" && GOOS=darwin GOARCH=$arch CGO_ENABLED=1 CC="$cc" MACOSX_DEPLOYMENT_TARGET=11.0 \
      CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags desktop,production -trimpath \
      -ldflags "$ldflags" -o "$work/arkflix-games-$arch" .)
  done
  lipo -create -output "$bundle/Contents/MacOS/arkflix-games" "$work/arkflix-games-arm64" "$work/arkflix-games-amd64"
  sed "s/@VERSION@/${version#v}/g" "$root/packaging/macos/Info.plist" >"$bundle/Contents/Info.plist"
  ln -s /Applications "$work/Applications"
  rm -f "$work"/arkflix-games-arm64 "$work"/arkflix-games-amd64
  hdiutil create -quiet -volname "arkflix games" -srcfolder "$work" -ov -format UDZO "$dist/$name-macos-universal.dmg"
  checksum "$name-macos-universal.dmg"
  ;;
*)
  echo "sistema desconhecido: $target" >&2
  exit 1
  ;;
esac
ls -1 "$dist" | grep -- "$name" | sed 's/^/📦 /'
