#!/usr/bin/env bash
# Uploads the Android signing secrets from .env to the GitHub repository (Settings → Secrets → Actions),
# where the release workflow reads them. Needs the GitHub CLI, logged in with access to the repository.
# The values are passed on standard input and are never printed.
# Usage: scripts/android-github-secrets.sh [owner/repo]   (default: the repository of the "origin" remote)
set -euo pipefail
cd "$(dirname "$0")/.."

[[ -f .env ]] || { echo ".env not found: run scripts/android-keystore.sh first" >&2; exit 1; }
set -a
# shellcheck disable=SC1091
source ./.env
set +a
: "${ANDROID_KEYSTORE_FILE:?}" "${ANDROID_KEYSTORE_PASSWORD:?}" "${ANDROID_KEY_PASSWORD:?}"

REPO="${1:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
echo "Setting the secrets on $REPO"

base64 -w0 "$ANDROID_KEYSTORE_FILE" | gh secret set ANDROID_KEYSTORE_BASE64 --repo "$REPO"
printf '%s' "$ANDROID_KEYSTORE_PASSWORD" | gh secret set ANDROID_KEYSTORE_PASSWORD --repo "$REPO"
printf '%s' "$ANDROID_KEY_PASSWORD" | gh secret set ANDROID_KEY_PASSWORD --repo "$REPO"
gh secret list --repo "$REPO"
