package sim

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var reOptionKey = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// BuildSubstatOptionsString builds the value for the engine CLI -options flag.
//
// The CLI expects a semicolon-separated list: "key=value;key2=value2".
// We sort keys for deterministic output.
func BuildSubstatOptionsString(options map[string]any) (string, error) {
	if len(options) == 0 {
		return "", nil
	}

	keys := make([]string, 0, len(options))
	for k := range options {
		k = strings.TrimSpace(k)
		if k == "" {
			return "", fmt.Errorf("substat optimizer option key is empty")
		}
		if !reOptionKey.MatchString(k) {
			return "", fmt.Errorf("substat optimizer option key %q is invalid (allowed: [a-zA-Z0-9_])", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := options[k]
		if v == nil {
			return "", fmt.Errorf("substat optimizer option %q has null value", k)
		}
		vs := fmt.Sprint(v)
		vs = strings.TrimSpace(vs)
		if vs == "" {
			return "", fmt.Errorf("substat optimizer option %q has empty value", k)
		}
		if strings.ContainsAny(vs, `;"`) {
			return "", fmt.Errorf("substat optimizer option %q has unsupported value %q (contains ';' or '\"')", k, vs)
		}
		parts = append(parts, k+"="+vs)
	}

	return strings.Join(parts, ";"), nil
}
