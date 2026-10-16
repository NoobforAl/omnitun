#!/usr/bin/env bash
# Builds signed release APKs (one per ABI) into app/build/app/outputs/flutter-apk/.
# Creates a release keystore on first run: app/android/omnitun-release.jks and
# app/android/key.properties (both git-ignored). BACK THEM UP: updates to an
# installed app must be signed with the same key.
# Run from Git Bash. Usage: scripts/build-android-release.sh [--skip-core]
set -euo pipefail
cd "$(dirname "$0")/.."

FLUTTER="${FLUTTER:-flutter}"
command -v "$FLUTTER" >/dev/null || FLUTTER="$HOME/flutter/bin/flutter.bat"
if [[ -z "${JAVA_HOME:-}" && -d "/c/Program Files/Java/jdk-17" ]]; then export JAVA_HOME="/c/Program Files/Java/jdk-17"; fi
KEYTOOL="${JAVA_HOME:+$JAVA_HOME/bin/}keytool"

PROPS=app/android/key.properties
if [[ ! -f "$PROPS" ]]; then
  PASS="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24)"
  "$KEYTOOL" -genkeypair -keystore app/android/omnitun-release.jks -alias omnitun \
    -keyalg RSA -keysize 4096 -validity 10000 -storepass "$PASS" -keypass "$PASS" \
    -dname "CN=Omnitun, O=Omnitun" >/dev/null
  printf 'storeFile=omnitun-release.jks\nstorePassword=%s\nkeyAlias=omnitun\nkeyPassword=%s\n' "$PASS" "$PASS" > "$PROPS"
  echo "Created release keystore app/android/omnitun-release.jks — back it up together with $PROPS."
fi

[[ "${1:-}" == "--skip-core" ]] || { scripts/build-android-core.sh; scripts/build-android-tor.sh arm64-v8a armeabi-v7a x86_64; }
cd app
"$FLUTTER" build apk --release --split-per-abi
ls -lh build/app/outputs/flutter-apk/*-release.apk
