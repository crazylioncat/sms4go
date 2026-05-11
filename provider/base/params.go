package base

import (
	"sort"
	"strconv"
	"strings"
)

func ValuesBySortedKey(values map[string]string) []string {
	if len(values) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func ParamsByAmpersand(message string) map[string]string {
	if strings.TrimSpace(message) == "" {
		return map[string]string{}
	}
	parts := strings.Split(message, "&")
	values := make(map[string]string, len(parts))
	for i, part := range parts {
		values[strconv.Itoa(i)] = part
	}
	return values
}
