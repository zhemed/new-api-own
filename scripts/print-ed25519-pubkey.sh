#!/bin/sh
# Print the raw 32-byte ed25519 public key (standard base64, padded) derived from
# an ed25519 PKCS#8 PEM private key.
#
# Used when rotating the release-signing key: the output must be written into
# BOTH trust anchors — `UPDATE_SIGNING_PUBLIC_KEY` in
# .github/workflows/release.yml and the key embedded in the panel binary.
# See docs/release-signing.md §4.
#
# Usage: scripts/print-ed25519-pubkey.sh <ed25519-private-key.pem>
set -eu

[ "$#" -eq 1 ] || {
  echo "usage: $0 <ed25519-private-key.pem>" >&2
  exit 2
}

key="$1"
[ -f "$key" ] || {
  echo "no such file: $key" >&2
  exit 2
}

spki="$(mktemp)"
trap 'rm -f "$spki"' EXIT

openssl pkey -in "$key" -pubout -outform DER -out "$spki" 2>/dev/null

size="$(wc -c < "$spki" | tr -d ' ')"
if [ "$size" -ne 44 ]; then
  echo "unexpected public key encoding: ${size} bytes (want 44 bytes of ed25519 SPKI)" >&2
  exit 1
fi

# Last 32 bytes of the SPKI structure are the raw ed25519 public key.
tail -c 32 "$spki" | base64 | tr -d '\n'
echo
