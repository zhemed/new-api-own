#!/bin/sh
set -eu
# Forbid direct encoding/json Marshal/Unmarshal in business code
# Allowed: common/json.go (wrapper itself), relaykit/, pkg/, *_test.go
if grep -rn --include="*.go" -E 'json\.(Marshal|Unmarshal|NewDecoder|NewEncoder|MarshalIndent)' . \
  | grep -v "common/json.go" | grep -v "relaykit/" | grep -v "pkg/" | grep -v "_test.go" | grep -q .; then
  echo "❌ Direct encoding/json usage found (must use common.Marshal/Unmarshal/DecodeJson):"
  grep -rn --include="*.go" -E 'json\.(Marshal|Unmarshal|NewDecoder|NewEncoder|MarshalIndent)' . | grep -v "common/json.go" | grep -v "relaykit/" | grep -v "pkg/" | grep -v "_test.go"
  exit 1
fi
echo "✅ No forbidden encoding/json usage"
