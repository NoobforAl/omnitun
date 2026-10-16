#!/usr/bin/env bash
# Builds the Tor client (Arti, tor/) for Android and copies it into the app as
# jniLibs/<abi>/libomnitun_tor.so; Android only lets an app execute files from
# its native library directory, and the core finds it there.
# Needs: rustup targets (aarch64/armv7/x86_64-linux-android), cargo-ndk, Android NDK.
# Run from Git Bash. Usage: scripts/build-android-tor.sh [abi...]  (default: arm64-v8a)
set -euo pipefail
cd "$(dirname "$0")/.."

ABIS=("$@")
[[ ${#ABIS[@]} -gt 0 ]] || ABIS=(arm64-v8a)
if [[ -z "${ANDROID_NDK_HOME:-}" ]]; then
  SDK="${ANDROID_HOME:-${LOCALAPPDATA:+$LOCALAPPDATA/Android/Sdk}}"
  SDK="${SDK:-$HOME/Android/Sdk}"
  ANDROID_NDK_HOME="$SDK/ndk/$(ls "$SDK/ndk" | sort -V | tail -1)"
  export ANDROID_NDK_HOME
fi

declare -A TRIPLE=([arm64-v8a]=aarch64-linux-android [armeabi-v7a]=armv7-linux-androideabi [x86_64]=x86_64-linux-android)
cd tor
for abi in "${ABIS[@]}"; do
  cargo ndk -t "$abi" -P 29 build --release --bin omnitun-tor
  dest="../app/android/app/src/main/jniLibs/$abi"
  mkdir -p "$dest"
  cp "target/${TRIPLE[$abi]}/release/omnitun-tor" "$dest/libomnitun_tor.so"
  ls -lh "$dest/libomnitun_tor.so"
done
