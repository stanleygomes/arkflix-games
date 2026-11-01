# arkflix-games: app de jogos + RetroArch (8 cores) + PPSSPP + PCSX2 compilados do fonte.
# Fonte em src/ (submodules travados em versão), build em build/, nada vai pro sistema.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

ROOT    := $(CURDIR)
SRC     := $(ROOT)/src
BUILD   := $(ROOT)/build
PREFIX  := $(BUILD)/prefix
BIN     := $(BUILD)/bin
CORES   := $(PREFIX)/lib/libretro
STATE   := $(ROOT)/home
VAR_APP := $(HOME)/.var/app
GAMES   := $(HOME)/Games/arkflix
JOBS    ?= $(shell nproc)
CMAKE   := cmake -G Ninja -DCMAKE_BUILD_TYPE=Release
APP     := $(ROOT)/app
WEB     := $(APP)/web
APP_BIN := $(PREFIX)/bin/arkflix-games
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# ccache, se instalado: recompilar do zero leva minutos (cache em ~/.cache/ccache)
CCACHE := $(shell command -v ccache 2>/dev/null)
ifneq ($(CCACHE),)
CMAKE += -DCMAKE_C_COMPILER_LAUNCHER=ccache -DCMAKE_CXX_COMPILER_LAUNCHER=ccache
CC_VARS := CC="ccache gcc" CXX="ccache g++"
endif

CORE_LIST := mesen snes9x gambatte genesis_plus_gx mupen64plus_next mednafen_psx \
             fbneo melondsds

# submodules do PPSSPP que o build Linux não usa (iOS, Mac, Android, VR, testes)
PPSSPP_SKIP := SDL/macOS ext/libadrenotools ios/MoltenVK \
               libretro/libretro-common

APT_DEPS := build-essential cmake ninja-build pkg-config nasm gettext ccache \
  libsdl2-dev libsdl2-ttf-dev libsdl3-dev libfreetype-dev libfontconfig-dev \
  libharfbuzz-dev libicu-dev libcurl4-openssl-dev \
  libavcodec-dev libavfilter-dev libavformat-dev libavutil-dev libswresample-dev \
  libswscale-dev \
  libasound2-dev libpulse-dev libpipewire-0.3-dev libjack-jackd2-dev \
  libgl-dev libgl1-mesa-dev libglu1-mesa-dev libegl-dev libgles-dev libvulkan-dev \
  libdrm-dev libgbm-dev \
  libx11-dev libx11-xcb-dev libxext-dev libxrandr-dev libxi-dev libxinerama-dev \
  libxxf86vm-dev \
  libxkbcommon-dev libwayland-dev wayland-protocols \
  libudev-dev libusb-1.0-0-dev libdbus-1-dev zlib1g-dev libzstd-dev liblz4-dev \
  libpng-dev libjpeg-dev libwebp-dev libpcap-dev libbacktrace-dev \
  qt6-base-dev qt6-base-private-dev qt6-tools-dev qt6-tools-dev-tools \
  qt6-l10n-tools qt6-svg-dev libkddockwidgets-qt6-dev libplutosvg-dev \
  libplutovg-dev directx-headers-dev libshaderc-dev extra-cmake-modules \
  libgtk-3-dev libwebkit2gtk-4.1-dev

.PHONY: help deps fetch build retroarch ppsspp pcsx2 rapidyaml cores \
        $(addprefix core-,$(CORE_LIST)) wrappers run status \
        patch-save patch-apply clean check check-cores check-update \
        check-ppsspp check-pcsx2 release-notes release-binaries \
        release-install app-deps app-web app run-web check-app check-app-desktop

##@ Começo

help: ## Lista os comandos
	@awk 'BEGIN{FS=":.*##"} /^##@/{printf "\n\033[1m%s\033[0m\n",substr($$0,5)} \
	  /^[a-zA-Z0-9_%-]+:.*##/{printf "  \033[36m%-16s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)

deps: ## Instala os pacotes de compilação (apt, pede sudo)
	sudo apt-get install -y --no-install-recommends $(APT_DEPS) \
	  $$(command -v go >/dev/null || echo golang-go) \
	  $$(command -v npm >/dev/null || echo nodejs npm)

fetch: ## Baixa os fontes (submodules rasos, nas versões travadas)
	git submodule update --init --depth 1 --jobs 4
	git -C src/deps/rapidyaml submodule update --init --depth 1 --recursive
	cd src/ppsspp && git submodule update --init --recursive --depth 1 --jobs 4 \
	  $$(git config -f .gitmodules --get-regexp '\.path$$' | awk '{print $$2}' \
	     | grep -vxF $(addprefix -e ,$(PPSSPP_SKIP)))
	$(MAKE) --no-print-directory patch-apply

##@ Build

build: retroarch cores ppsspp pcsx2 app wrappers $(BUILD)/coretest ## Compila tudo

retroarch: ## RetroArch (build dentro de src/retroarch, é o jeito dele)
	cd src/retroarch && $(CC_VARS) ./configure --prefix=$(PREFIX) --disable-update_cores \
	  --disable-update_assets --disable-update_core_info
	$(MAKE) -C src/retroarch -j $(JOBS) $(CC_VARS)
	$(MAKE) -C src/retroarch install

ppsspp: ## PPSSPP (PSP)
	$(CMAKE) -S src/ppsspp -B $(BUILD)/ppsspp -DUSING_QT_UI=OFF -DHEADLESS=ON \
	  -DUNITTEST=ON
	cmake --build $(BUILD)/ppsspp -j $(JOBS)

rapidyaml: ## Única dependência do PCSX2 que o apt não tem
	$(CMAKE) -S src/deps/rapidyaml -B $(BUILD)/rapidyaml \
	  -DCMAKE_INSTALL_PREFIX=$(BUILD)/deps -DBUILD_SHARED_LIBS=OFF \
	  -DCMAKE_POSITION_INDEPENDENT_CODE=ON
	cmake --build $(BUILD)/rapidyaml -j $(JOBS)
	cmake --install $(BUILD)/rapidyaml

# O PCSX2 abre "libshaderc_shared.so.1" em tempo de execução; o Ubuntu chama de
# libshaderc. O shim é só um link, usado no build e no lançador (LD_LIBRARY_PATH).
SHADERC := $(shell pkg-config --variable=libdir shaderc 2>/dev/null)/libshaderc.so

pcsx2: rapidyaml ## PCSX2 (PS2)
	@mkdir -p $(BUILD)/shim
	ln -sf $(SHADERC) $(BUILD)/shim/libshaderc_shared.so
	ln -sf $(SHADERC).1 $(BUILD)/shim/libshaderc_shared.so.1
	$(CMAKE) -S src/pcsx2 -B $(BUILD)/pcsx2 -DCMAKE_PREFIX_PATH=$(BUILD)/deps \
	  -DSHADERC_LIBRARY=$(BUILD)/shim/libshaderc_shared.so \
	  -DENABLE_TESTS=ON -DPACKAGE_MODE=OFF
	cmake --build $(BUILD)/pcsx2 -j $(JOBS)

cores: $(addprefix core-,$(CORE_LIST)) ## Os 8 cores do RetroArch

core-mesen:
	$(MAKE) -C src/cores/mesen/Libretro -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/mesen/Libretro/mesen_libretro.so -t $(CORES)
core-snes9x:
	$(MAKE) -C src/cores/snes9x/libretro -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/snes9x/libretro/snes9x_libretro.so -t $(CORES)
core-gambatte:
	$(MAKE) -C src/cores/gambatte -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/gambatte/gambatte_libretro.so -t $(CORES)
core-genesis_plus_gx:
	$(MAKE) -C src/cores/genesis_plus_gx -f Makefile.libretro -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/genesis_plus_gx/genesis_plus_gx_libretro.so -t $(CORES)
core-mupen64plus_next:
	$(MAKE) -C src/cores/mupen64plus_next -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/mupen64plus_next/mupen64plus_next_libretro.so -t $(CORES)
core-mednafen_psx:
	$(MAKE) -C src/cores/mednafen_psx -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/mednafen_psx/mednafen_psx_libretro.so -t $(CORES)
core-fbneo:
	$(MAKE) -C src/cores/fbneo/src/burner/libretro -j $(JOBS) $(CC_VARS)
	install -Dm644 src/cores/fbneo/src/burner/libretro/fbneo_libretro.so -t $(CORES)
core-melondsds:
	$(CMAKE) -S src/cores/melondsds -B $(BUILD)/cores/melondsds
	cmake --build $(BUILD)/cores/melondsds -j $(JOBS)
	install -Dm644 $(BUILD)/cores/melondsds/src/libretro/melondsds_libretro.so -t $(CORES)

##@ Rodar

wrappers: ## Lançadores em build/bin (saves no mesmo lugar dos Flatpaks)
	@mkdir -p $(BIN) $(STATE)
	@for w in retroarch ppsspp pcsx2; do \
	  sed -e 's|@PREFIX@|$(PREFIX)|g' -e 's|@BUILD@|$(BUILD)|g' -e 's|@STATE@|$(STATE)|g' \
	      -e 's|@VAR_APP@|$(VAR_APP)|g' config/$$w.sh.in > $(BIN)/$$w; \
	  chmod +x $(BIN)/$$w; done
	@sed -e 's|@SRC@|$(SRC)|g' -e 's|@CORES@|$(CORES)|g' -e 's|@VAR_APP@|$(VAR_APP)|g' \
	  config/retroarch-paths.cfg.in > $(STATE)/retroarch-paths.cfg
	@touch $(STATE)/retroarch.cfg
	@echo "wrappers em $(BIN)"

run: wrappers ## Abre o app: biblioteca, saves, controles e configurações
	@test -x $(APP_BIN) || { echo "❌ app não compilado: make app"; exit 1; }
	$(APP_BIN) --bin $(BIN) --cores $(CORES)

run-web: wrappers app-web ## O mesmo app no navegador (desenvolvimento): http://127.0.0.1:8484
	cd $(APP) && go run . --web 127.0.0.1:8484 --bin $(BIN) --cores $(CORES)

##@ App (frontend)

app-deps: ## Dependências da interface (npm; precisa de Node 20+ e Go 1.26+)
	@command -v npm >/dev/null || { echo "❌ falta Node.js (npm) pra compilar a interface"; exit 1; }
	@command -v go >/dev/null || { echo "❌ falta Go pra compilar o app"; exit 1; }
	cd $(WEB) && npm ci --no-fund --no-audit

app-web: ## Compila a interface (Svelte + TypeScript) em app/web/dist
	@test -d $(WEB)/node_modules || $(MAKE) --no-print-directory app-deps
	cd $(WEB) && npm run build

app: app-web ## Compila o app desktop (Go + Wails) em build/prefix/bin/arkflix-games
	@pkg-config --exists webkit2gtk-4.1 || { echo "❌ falta o WebKitGTK de desenvolvimento: make deps"; exit 1; }
	cd $(APP) && CGO_ENABLED=1 go build -tags desktop,production,webkit2_41 -trimpath \
	  -ldflags "-s -w -X main.version=$(VERSION)" -o $(APP_BIN) .

##@ Validar

# CASOS=nes snes ... restringe os cores testados (padrão: todos + retroarch)
check: ## Valida tudo: app, cores com jogo de verdade, PPSSPP, PCSX2
	@$(MAKE) --no-print-directory -k check-app check-cores check-ppsspp check-pcsx2 \
	  && echo "✅ check passou" || { echo "❌ check falhou (detalhe acima)"; exit 1; }

check-app: app-web ## App: Go (vet, testes, race), tipos, unitários, ponta a ponta e janela
	cd $(APP) && go vet ./... && go test -race -count=1 ./...
	cd $(WEB) && npx playwright install chromium >/dev/null
	cd $(WEB) && npm run check && npm test && npx playwright test
	@if [ -x $(APP_BIN) ]; then $(MAKE) --no-print-directory check-app-desktop; \
	  else echo "⏭️  app desktop não compilado (make app): teste da janela pulado"; fi

check-app-desktop: ## Abre a janela de verdade, escondida, e espera a interface responder
	cd $(APP) && go vet -tags desktop,production,webkit2_41 .
	rm -rf $(BUILD)/smoke && $(APP_BIN) --smoke --data $(BUILD)/smoke --bin $(BIN) --cores $(CORES)

check-cores: $(BUILD)/coretest ## Cada core roda um jogo seu sem janela (CASOS=...)
	python3 -I tests/check.py $(CASOS)

check-update: $(BUILD)/coretest ## Igual ao check-cores, e grava o resultado como baseline
	python3 -I tests/check.py --update $(CASOS)

check-ppsspp: ## Suíte oficial do PPSSPP (pspautotests) + testes unitários
	ln -sfn $(BUILD)/ppsspp src/ppsspp/build-arkflix
	cd src/ppsspp && python3 test.py -g --graphics=software
	cd src/ppsspp && $(BUILD)/ppsspp/PPSSPPUnitTest ALL

check-pcsx2: ## Testes unitários do PCSX2
	cmake --build $(BUILD)/pcsx2 --target unittests -j $(JOBS)
	ctest --test-dir $(BUILD)/pcsx2/tests/ctest --output-on-failure --no-tests=error -j $(JOBS)

$(BUILD)/coretest: tests/coretest.c
	@mkdir -p $(BUILD)
	cc -O2 -Wall -Wextra -I src/retroarch/libretro-common/include -o $@ $< -ldl -lEGL

##@ Release

# Pacote = só o que o make run usa (sem objetos de compilação). Roda em
# qualquer pasta: os lançadores são gerados na instalação.
DIST := $(BUILD)/dist
GH_REPO ?= stanleygomes/arkflix-games
RELEASE_FILES := prefix ppsspp/PPSSPPSDL ppsspp/assets pcsx2/bin shim coretest

release-notes: ## Mostra a próxima versão e o changelog (o CI publica no push)
	@bin/release.sh

release-binaries: ## Valida, empacota o build e anexa na release da tag do HEAD
	@test -z "$$(git status --porcelain)" || { echo "❌ commit/push antes"; exit 1; }
	@git fetch -q --tags origin
	@tag=$$(git describe --tags --exact-match HEAD 2>/dev/null) || { \
	  echo "❌ HEAD sem tag: dê push e espere o CI (ou é só docs/chore)"; exit 1; }
	$(MAKE) --no-print-directory check
	@set -e; tag=$$(git describe --tags --exact-match HEAD); \
	file=arkflix-games-$$tag-linux-x86_64.tar.gz; mkdir -p $(DIST); \
	tar -C $(BUILD) -czf $(DIST)/$$file $(RELEASE_FILES); \
	( cd $(DIST) && sha256sum $$file > $$file.sha256 ); \
	gh release upload $$tag $(DIST)/$$file $(DIST)/$$file.sha256 --clobber; \
	echo "✅ binários anexados na $$tag"

release-install: ## Baixa os binários do último release que tiver (VERSION=vX fixa)
	@bin/release-download.sh "$(GH_REPO)" "$(VERSION)" "$(DIST)/download"
	tar -C $(BUILD) -xzf $(DIST)/download/*.tar.gz
	git submodule update --init --depth 1 src/retroarch-assets \
	  src/retroarch-joypad-autoconfig src/libretro-core-info
	$(MAKE) --no-print-directory wrappers
	@echo "✅ pronto: make run"

##@ Fonte

status: ## Versão de cada fonte e o que você mudou
	@git submodule foreach --quiet 'printf "%-36s %-12s %s\n" "$$sm_path" \
	  "$$(git describe --tags --always)" "$$(git status --porcelain -uno | wc -l | \
	  sed "s/^0$$//; s/[0-9]\+/& arquivo(s) mudado(s)/")"'

patch-save: ## Suas mudanças em src/* viram patches/<projeto>.patch
	@mkdir -p patches
	@git submodule foreach --quiet 'f=$(ROOT)/patches/$$(echo $$sm_path | sed "s|^src/||; s|/|-|g").patch; \
	  if git diff --quiet; then rm -f "$$f"; else git diff > "$$f"; echo "salvo $$f"; fi'

patch-apply: ## Aplica patches/*.patch (pula o que já está aplicado)
	@shopt -s nullglob; for p in patches/*.patch; do \
	  d=src/$$(basename $$p .patch | sed 's|^cores-|cores/|; s|^deps-|deps/|'); \
	  if git -C $$d apply --reverse --check $(ROOT)/$$p 2>/dev/null; then echo "já aplicado $$p"; \
	  else git -C $$d apply $(ROOT)/$$p && echo "aplicado $$p"; fi; done

clean: ## Apaga o build (fonte e patches ficam)
	rm -rf $(BUILD)
	-$(MAKE) -C src/retroarch clean
	-for c in mesen/Libretro snes9x/libretro gambatte mupen64plus_next mednafen_psx \
	  fbneo/src/burner/libretro; do $(MAKE) -C src/cores/$$c clean; done
	-$(MAKE) -C src/cores/genesis_plus_gx -f Makefile.libretro clean
