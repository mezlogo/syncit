package processhandlers

import (
	"sort"
	"strings"
)

const (
	placeholderAllUnique = "$ALL_UNIQUE_VALUES_JOIN_WHITESPACE"
	placeholderSingle    = "$UNIQUE_SINGLE_VALUE"
)

func substituteAllUnique(template string, values []string) string {
	return strings.ReplaceAll(template, placeholderAllUnique, strings.Join(values, " "))
}

func substituteSingle(template, value string) string {
	return strings.ReplaceAll(template, placeholderSingle, value)
}

// uniqueNonEmpty trims, deduplicates and sorts values for deterministic output.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
