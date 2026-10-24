// Package kitutil holds the dependency-free helpers shared by the conversion
// kit packages (dto, types, relayconvert). It moved out of the host's common
// package as part of the relaykit extraction; common re-exports these for
// host code.
package kitutil

import (
	"bytes"
	"encoding/json"
)

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func GetJsonType(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// JsonRawMessageToString returns JSON strings as their decoded value and other JSON values as raw text.
func JsonRawMessageToString(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	if trimmed[0] != '"' {
		return string(trimmed)
	}
	var value string
	if err := Unmarshal(trimmed, &value); err != nil {
		return string(trimmed)
	}
	return value
}

func Any2Type[T any](data any) (T, error) {
	var zero T
	bytes, err := json.Marshal(data)
	if err != nil {
		return zero, err
	}
	var res T
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return zero, err
	}
	return res, nil
}
