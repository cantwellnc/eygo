package ir

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Decode parses an EYG IR node from dag-json bytes.
func Decode(data []byte) (Node, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("ir: invalid json: %w", err)
	}
	return decodeNode(raw)
}

func decodeNode(raw map[string]json.RawMessage) (Node, error) {
	kindRaw, ok := raw["0"]
	if !ok {
		return nil, fmt.Errorf("ir: missing discriminant key \"0\"")
	}
	var kind string
	if err := json.Unmarshal(kindRaw, &kind); err != nil {
		return nil, fmt.Errorf("ir: invalid discriminant: %w", err)
	}

	switch kind {
	case "v":
		return &Var{Label: mustStr(raw, "l")}, nil

	case "f":
		body, err := decodeChild(raw, "b")
		if err != nil {
			return nil, err
		}
		return &Lambda{Label: mustStr(raw, "l"), Body: body}, nil

	case "a":
		fn, err := decodeChild(raw, "f")
		if err != nil {
			return nil, err
		}
		arg, err := decodeChild(raw, "a")
		if err != nil {
			return nil, err
		}
		return &Apply{Fn: fn, Arg: arg}, nil

	case "l":
		val, err := decodeChild(raw, "v")
		if err != nil {
			return nil, err
		}
		then, err := decodeChild(raw, "t")
		if err != nil {
			return nil, err
		}
		return &Let{Label: mustStr(raw, "l"), Value: val, Then: then}, nil

	case "i":
		var v int64
		if err := json.Unmarshal(raw["v"], &v); err != nil {
			return nil, fmt.Errorf("ir: int value: %w", err)
		}
		return &Int{Value: v}, nil

	case "s":
		var v string
		if err := json.Unmarshal(raw["v"], &v); err != nil {
			return nil, fmt.Errorf("ir: string value: %w", err)
		}
		return &String{Value: v}, nil

	case "x":
		b, err := decodeDagBytes(raw["v"])
		if err != nil {
			return nil, fmt.Errorf("ir: binary value: %w", err)
		}
		return &Binary{Value: b}, nil

	case "ta":
		return &Tail{}, nil

	case "c":
		return &Cons{}, nil

	case "u":
		return &Empty{}, nil

	case "e":
		return &Extend{Label: mustStr(raw, "l")}, nil

	case "o":
		return &Overwrite{Label: mustStr(raw, "l")}, nil

	case "g":
		return &Select{Label: mustStr(raw, "l")}, nil

	case "t":
		return &Tag{Label: mustStr(raw, "l")}, nil

	case "m":
		return &Case{Label: mustStr(raw, "l")}, nil

	case "n":
		return &NoCases{}, nil

	case "p":
		return &Perform{Label: mustStr(raw, "l")}, nil

	case "h":
		return &Handle{Label: mustStr(raw, "l")}, nil

	case "hs":
		return &Shallow{Label: mustStr(raw, "l")}, nil

	case "b":
		return &Builtin{Label: mustStr(raw, "l")}, nil

	case "#":
		cid, err := decodeDagLink(raw["l"])
		if err != nil {
			return nil, fmt.Errorf("ir: reference cid: %w", err)
		}
		return &Reference{CID: cid}, nil

	case "@":
		cid, err := decodeDagLink(raw["l"])
		if err != nil {
			return nil, fmt.Errorf("ir: release cid: %w", err)
		}
		var rel int64
		if err := json.Unmarshal(raw["r"], &rel); err != nil {
			return nil, fmt.Errorf("ir: release number: %w", err)
		}
		return &Release{
			Package: mustStr(raw, "p"),
			Release: rel,
			CID:     cid,
		}, nil

	case "z":
		comment := mustStr(raw, "c") // optional; empty string if absent
		return &Vacant{Comment: comment}, nil

	default:
		return nil, fmt.Errorf("ir: unknown node kind %q", kind)
	}
}

func mustStr(raw map[string]json.RawMessage, key string) string {
	r, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(r, &s)
	return s
}

func decodeChild(raw map[string]json.RawMessage, key string) (Node, error) {
	field, ok := raw[key]
	if !ok {
		return nil, fmt.Errorf("ir: missing field %q", key)
	}
	var child map[string]json.RawMessage
	if err := json.Unmarshal(field, &child); err != nil {
		return nil, fmt.Errorf("ir: field %q: %w", key, err)
	}
	return decodeNode(child)
}

// decodeDagBytes handles dag-json byte encoding: {"/":{"bytes":"<base64url>"}}
func decodeDagBytes(raw json.RawMessage) ([]byte, error) {
	var wrapper struct {
		Slash struct {
			Bytes string `json:"bytes"`
		} `json:"/"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Slash.Bytes != "" {
		return base64.RawURLEncoding.DecodeString(wrapper.Slash.Bytes)
	}
	// fallback: plain base64 string
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unrecognised bytes encoding")
	}
	return base64.StdEncoding.DecodeString(s)
}

// decodeDagLink extracts a CID string from a dag-json link: {"/": "<cid>"}
func decodeDagLink(raw json.RawMessage) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("missing link field")
	}
	// First try plain string (some encoders omit the link wrapper)
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var link struct {
		Slash string `json:"/"`
	}
	if err := json.Unmarshal(raw, &link); err != nil {
		return "", fmt.Errorf("unrecognised link encoding: %w", err)
	}
	return link.Slash, nil
}
