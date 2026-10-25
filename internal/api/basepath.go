package api

import (
	"fmt"
	"strings"
)

// ParseBasePath checks the BASE_PATH a reverse proxy (sthings-portal) serves
// the UI under. The proxy forwards the path unchanged, so the service answers
// under the prefix itself and puts it on every URL it hands to the browser.
// "" (or "/") means no prefix; otherwise it is "/" plus path segments, without
// a trailing slash.
func ParseBasePath(v string) (string, error) {
	v = strings.TrimSuffix(strings.TrimSpace(v), "/")
	if v == "" {
		return "", nil
	}
	if !strings.HasPrefix(v, "/") {
		return "", fmt.Errorf("BASE_PATH %q must start with /", v)
	}
	for _, seg := range strings.Split(v[1:], "/") {
		if seg == "" || seg == "." || seg == ".." || strings.ContainsAny(seg, "?#%{}\\ \t") {
			return "", fmt.Errorf("BASE_PATH %q is not a clean path", v)
		}
	}
	return v, nil
}
