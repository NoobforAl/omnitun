#!/usr/bin/env bash
# Creates the Android release keystore with a random password, once.
#   - the keystore:   app/android/omnitun-release.jks          (git-ignored)
#   - the passwords:  .env at the repository root              (git-ignored)
#   - for Gradle:     app/android/key.properties               (git-ignored; local release builds use it)
# Refuses to replace an existing key: the key signs every update of the app, and a new one cannot update
# an installed app. BACK UP .env AND THE KEYSTORE (a password manager is a good place).
# Run from Git Bash. Needs the JDK's keytool.
set -euo pipefail
cd "$(dirname "$0")/.."

KS=app/android/omnitun-release.jks
ENVF=.env

if [[ -f "$ENVF" ]] && grep -q '^ANDROID_KEYSTORE_PASSWORD=' "$ENVF"; then
  echo ".env already holds a keystore password: nothing to do."
  exit 0
fi
if [[ -f "$KS" ]]; then
  echo "$KS exists but .env has no password for it. Move the old keystore away first." >&2
  exit 1
fi

if [[ -z "${JAVA_HOME:-}" && -d "/c/Program Files/Java/jdk-17" ]]; then export JAVA_HOME="/c/Program Files/Java/jdk-17"; fi
KEYTOOL="${JAVA_HOME:+$JAVA_HOME/bin/}keytool"

# 32 random letters and digits (from the system's random source, never printed).
PASS="$(head -c 96 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 32)"
[[ ${#PASS} -eq 32 ]] || { echo "could not make a password" >&2; exit 1; }

# PKCS12 keystores use one password for the store and the key.
"$KEYTOOL" -genkeypair -keystore "$KS" -storetype PKCS12 -alias omnitun \
  -keyalg RSA -keysize 4096 -validity 10000 -storepass "$PASS" -keypass "$PASS" \
  -dname "CN=Omnitun, O=Omnitun" >/dev/null

umask 077
{
  echo "# Android release signing. Secret: never commit, never share. Back it up with the keystore."
  echo "ANDROID_KEYSTORE_FILE=$KS"
  echo "ANDROID_KEY_ALIAS=omnitun"
  echo "ANDROID_KEYSTORE_PASSWORD=$PASS"
  echo "ANDROID_KEY_PASSWORD=$PASS"
} > "$ENVF"
printf 'storeFile=omnitun-release.jks\nstorePassword=%s\nkeyAlias=omnitun\nkeyPassword=%s\n' "$PASS" "$PASS" > app/android/key.properties

echo "Created $KS and $ENVF (random 32-character password), and app/android/key.properties."
echo "Fingerprint:"
"$KEYTOOL" -list -keystore "$KS" -storepass "$PASS" 2>/dev/null | grep -i "SHA-256\|SHA256" | head -1 || true
