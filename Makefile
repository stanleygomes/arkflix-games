# arkflix-games: app de jogos (biblioteca, saves, controle, perfis) que abre os
# jogos nos Flatpaks do RetroArch (8 cores), PPSSPP e PCSX2. Só o app é
# compilado aqui; os emuladores vêm prontos do Flathub.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

ROOT    := $(CURDIR)
BUILD   := $(ROOT)/build
PREFIX  := $(BUILD)/prefix
APP     := $(ROOT)/app
WEB     := $(APP)/web
APP_BIN := $(PREFIX)/bin/arkflix-games
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Emuladores (Flathub) e os cores que o catálogo do app usa
FLATPAKS  := org.libretro.RetroArch org.ppsspp.PPSSPP net.pcsx2.PCSX2
CORE_LIST := mesen snes9x gambatte genesis_plus_gx mupen64plus_next mednafen_psx \
             fbneo melondsds
RA_CORES  := $(HOME)/.var/app/org.libretro.RetroArch/config/retroarch/cores
BUILDBOT  := https://buildbot.libretro.com/nightly/linux/x86_64/latest

APT_DEPS := build-essential pkg-config curl unzip flatpak \
  libgtk-3-dev libwebkit2gtk-4.1-dev

.PHONY: help deps emulators build app-deps app-web app run run-web \
        check check-app check-app-desktop check-emulators clean \
        release-notes release-binaries release-install package

##@ Começo

help: ## Lista os comandos
	@awk 'BEGIN{FS=":.*##"} /^##@/{printf "\n\033[1m%s\033[0m\n",substr($$0,5)} \
	  /^[a-zA-Z0-9_%-]+:.*##/{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)

deps: ## Pacotes do sistema: compilar o app e rodar Flatpak (apt, pede sudo)
	sudo apt-get install -y --no-install-recommends $(APT_DEPS) \
	  $$(command -v go >/dev/null || echo golang-go) \
	  $$(command -v npm >/dev/null || echo nodejs npm)

emulators: ## RetroArch, PPSSPP e PCSX2 do Flathub + os 8 cores (sem sudo)
	@command -v flatpak >/dev/null || { echo "❌ falta o Flatpak: make deps"; exit 1; }
	flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
	@for id in $(FLATPAKS); do \
	  flatpak info --show-ref $$id >/dev/null 2>&1 && echo "ok $$id" || \
	    flatpak install --user -y --noninteractive flathub $$id; done
	# o PCSX2 do Flathub não enxerga a home: libera leitura pros jogos de PS2
	flatpak override --user --filesystem=host:ro net.pcsx2.PCSX2
	@mkdir -p $(RA_CORES)
	@for c in $(CORE_LIST); do \
	  f=$(RA_CORES)/$${c}_libretro.so; \
	  if [ -f $$f ]; then echo "ok core $$c"; continue; fi; \
	  echo "baixando core $$c"; tmp=$$(mktemp); \
	  curl -fsSL $(BUILDBOT)/$${c}_libretro.so.zip -o $$tmp; \
	  unzip -qo $$tmp -d $(RA_CORES); rm -f $$tmp; done
	@echo "✅ emuladores prontos"

##@ Build

build: app ## Compila o app

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

##@ Rodar

run: ## Abre o app: biblioteca, saves, controles, perfis e configurações
	@test -x $(APP_BIN) || { echo "❌ app não compilado: make app"; exit 1; }
	$(APP_BIN) $(ARGS)

run-web: app-web ## O mesmo app no navegador (desenvolvimento): http://127.0.0.1:8484
	cd $(APP) && go run . --web 127.0.0.1:8484

##@ Validar

check: ## Valida tudo: app (Go, interface, ponta a ponta, janela) e emuladores
	@$(MAKE) --no-print-directory -k check-app check-emulators \
	  && echo "✅ check passou" || { echo "❌ check falhou (detalhe acima)"; exit 1; }

check-app: app-web ## App: Go (vet, testes, race), tipos, unitários, ponta a ponta e janela
	cd $(APP) && go vet ./... && go test -race -count=1 ./...
	cd $(WEB) && npx playwright install chromium >/dev/null
	cd $(WEB) && npm run check && npm test && npx playwright test
	@if [ -x $(APP_BIN) ]; then $(MAKE) --no-print-directory check-app-desktop; \
	  else echo "⏭️  app desktop não compilado (make app): teste da janela pulado"; fi

check-app-desktop: ## Abre a janela de verdade, escondida, e espera a interface responder
	cd $(APP) && go vet -tags desktop,production,webkit2_41 .
	rm -rf $(BUILD)/smoke && $(APP_BIN) --smoke --data $(BUILD)/smoke

check-emulators: ## Os 3 Flatpaks e os 8 cores estão instalados
	@ok=1; for id in $(FLATPAKS); do \
	  if flatpak info --show-ref $$id >/dev/null 2>&1; then echo "ok $$id"; \
	  else echo "❌ falta $$id (make emulators)"; ok=0; fi; done; \
	for c in $(CORE_LIST); do \
	  if [ -f $(RA_CORES)/$${c}_libretro.so ]; then echo "ok core $$c"; \
	  else echo "❌ falta o core $$c (make emulators)"; ok=0; fi; done; \
	[ $$ok = 1 ]

clean: ## Apaga o build
	rm -rf $(BUILD)

##@ Release

# Instaladores: bin/package.sh (o CI gera Linux, Windows e macOS a cada release).
DIST := $(BUILD)/dist
GH_REPO ?= stanleygomes/arkflix-games
RELEASE ?=
RELEASE_FILES := prefix/bin/arkflix-games

package: app-web ## Empacota o app pra um sistema em build/dist (OS=linux|windows|macos)
	bin/package.sh $(or $(OS),linux) $(VERSION)


release-notes: ## Mostra a próxima versão e o changelog (o CI publica no push)
	@bin/release.sh

release-binaries: ## Plano B: gera os pacotes de Linux e Windows daqui e anexa na tag do HEAD (o CI já faz os 3 sistemas)
	@test -z "$$(git status --porcelain)" || { echo "❌ commit/push antes"; exit 1; }
	@git fetch -q --tags origin
	@tag=$$(git describe --tags --exact-match HEAD 2>/dev/null) || { \
	  echo "❌ HEAD sem tag: dê push e espere o CI (ou é só docs/chore)"; exit 1; }
	$(MAKE) --no-print-directory app check
	@set -e; tag=$$(git describe --tags --exact-match HEAD); \
	bin/package.sh linux $$tag; bin/package.sh windows $$tag; \
	gh release upload $$tag $(DIST)/arkflix-games-$$tag-* --clobber; \
	echo "✅ pacotes anexados na $$tag"

release-install: ## Baixa o app do último release que tiver (RELEASE=vX fixa)
	@bin/release-download.sh "$(GH_REPO)" "$(RELEASE)" "$(DIST)/download"
	tar -C $(BUILD) -xzf $(DIST)/download/*.tar.gz $(RELEASE_FILES)
	@echo "✅ pronto: make run"
