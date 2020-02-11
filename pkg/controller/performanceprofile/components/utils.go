package components

import (
	"fmt"
	"strings"
)

// GetComponentName returns the component name for the specific performance profile
func GetComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// GetFirstKeyAndValue return the first key / value pair of a map
func GetFirstKeyAndValue(m map[string]string) (string, string) {
	for k, v := range m {
		return k, v
	}
	return "", ""
}

// SplitLabelKey returns the given label key splitted up in domain and role
func SplitLabelKey(s string) (domain, role string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Can't split %s", s)
	}
	return parts[0], parts[1], nil
}
