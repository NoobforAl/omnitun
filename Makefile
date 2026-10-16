# On Windows, recipes run in Git's sh, never in WSL: a plain `bash` on PATH can be
# C:\Windows\System32\bash.exe, the WSL launcher, which fails with "execvpe(/bin/bash)".
ifeq ($(OS),Windows_NT)
GIT_SH := $(firstword $(wildcard C:/PROGRA~1/Git/usr/bin/sh.exe C:/PROGRA~2/Git/usr/bin/sh.exe $(subst \,/,$(USERPROFILE))/scoop/apps/git/current/usr/bin/sh.exe))
ifeq ($(GIT_SH),)
$(error Git for Windows (its sh.exe) is needed to run this Makefile; install it, or run make from Git Bash)
endif
SHELL := $(GIT_SH)
BASH := $(dir $(GIT_SH))bash.exe
BAT := .bat
else
BASH := bash
BAT :=
endif

# sing-box feature tags used by every Go build and test.
TAGS := with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_openvpn,with_openconnect,with_acme,badlinkname,tfogo_checklinkname0
LDFLAGS := -s -w -checklinkname=0
GOFLAGS_ALL := -tags "$(TAGS)" -ldflags "$(LDFLAGS)"

.PHONY: build test vet tor app-test server-release fix dev-build dev-server dev-tui dev-web dev-android dev-clean help docker-e2e notices

build:
	go build $(GOFLAGS_ALL) -o bin/ ./cmd/...

test:
	go test $(GOFLAGS_ALL) ./...

vet:
	go vet -tags "$(TAGS)" ./...

tor:
	cd tor && cargo test

app-test:
	cd app && flutter analyze && flutter test

# Static Linux server binaries (amd64, arm64) and SHA256SUMS in dist/.
server-release:
	$(BASH) scripts/build-server-release.sh

# Regenerates the license list the app shows (Settings > Open-source licenses).
# Run it after changing dependencies; CI checks that every license is known.
notices:
	python scripts/gen-notices.py --out app/assets/third_party_notices.md

# Real clients (OpenVPN, WireGuard, AmneziaWG, SSTP, IKEv2) against the Linux
# server, in Docker. Needs Docker; PROTO="sstp ikev2" picks some.
docker-e2e:
	python tools/dockertest/run.py $(PROTO)

# ---------------------------------------------------------------------------
# Development. `make help` lists these.
#
#   make dev-server    run a local server (creates .dev/server on first use, user "dev")
#   make dev-tui       the terminal UI of that server
#   make dev-web       the web API of that server (docs at http://127.0.0.1:<port>/docs)
#   make dev-android   build the core for the connected phone, then run the app on it
#   make fix           format and fix the code (gofmt, go fix, go vet, dart fix, cargo fmt)
#   make docker-e2e    real VPN clients against the Linux server in Docker (PROTO="sstp ikev2")
# ---------------------------------------------------------------------------

DEV_DATA ?= .dev/server
ifeq ($(OS),Windows_NT)
EXE := .exe
SDK := $(LOCALAPPDATA)/Android/Sdk
else
EXE :=
SDK := $(or $(ANDROID_HOME),$(HOME)/Android/Sdk)
endif
SERVER := bin/omnitun-server$(EXE) --data $(DEV_DATA)
ADB := $(SDK)/platform-tools/adb$(EXE)
# Flutter: the one on PATH, else ~/flutter/bin (resolved by the shell, so Windows paths survive).
# flutter.bat / dart.bat on Windows: the extensionless flutter and dart are bash scripts (`env bash`), which can find WSL.
FLUTTER_SH := FLUTTER="$$(command -v flutter$(BAT) || echo "$$HOME/flutter/bin/flutter$(BAT)")"

help:
	@sed -n '/^# Development/,/^# -----/p' Makefile | sed 's/^# \{0,1\}//' | grep -v '^---'

dev-build:
	go build $(GOFLAGS_ALL) -o bin/omnitun-server$(EXE) ./cmd/omnitun-server

# The first run creates the dev server: all protocols on 127.0.0.1, a user called "dev".
$(DEV_DATA)/state.json:
	$(SERVER) init --host 127.0.0.1 --first-user dev

dev-server: dev-build $(DEV_DATA)/state.json
	@echo "Subscription of user dev:"; $(SERVER) link dev --sub
	$(SERVER) run

dev-tui: dev-build $(DEV_DATA)/state.json
	$(SERVER) tui

dev-web: dev-build $(DEV_DATA)/state.json
	$(SERVER) api enable
	$(SERVER) run

# Needs a phone with USB debugging (or an emulator). A release build, because the installed app
# is signed with the release key (a debug build cannot replace it).
dev-android:
	@"$(ADB)" devices | grep -w device >/dev/null || { echo "no Android device: connect a phone with USB debugging"; exit 1; }
	$(BASH) scripts/build-android-core.sh android/arm64
	$(FLUTTER_SH); cd app && "$$FLUTTER" run --release --target-platform android-arm64

dev-clean:
	rm -rf .dev

# Formats and fixes what tools can: Go (gofmt, go fix, go vet), Dart (dart fix, analyze), Rust (cargo fmt).
# Dart's formatter is left out on purpose: it would rewrite every file in a different style.
fix:
	gofmt -s -w $$(git ls-files '*.go')
	go fix -tags "$(TAGS)" ./...
	go vet -tags "$(TAGS)" ./...
	$(FLUTTER_SH); cd app && "$$FLUTTER" pub get >/dev/null && "$$(dirname "$$FLUTTER")/dart$(BAT)" fix --apply && "$$FLUTTER" analyze --no-fatal-infos
	-cd tor && cargo fmt
