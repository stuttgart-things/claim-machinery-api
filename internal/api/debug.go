package api

import (
	"log"
	"os"
	"strings"
)

// debugEnabled caches the debug mode check at startup
var debugEnabled = isDebugEnabled()

func isDebugEnabled() bool {
	val := strings.ToLower(os.Getenv("DEBUG"))
	return val == "1" || val == "true" || val == "yes"
}

// debugf prints a debug message if DEBUG mode is enabled
func debugf(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// debugParams logs parameter information for order requests
func debugParams(action string, params map[string]interface{}) {
	if !debugEnabled {
		return
	}
	debugf("%s: %d parameters", action, len(params))
	for k, v := range params {
		debugf("  %s = %v", k, v)
	}
}

// IsDebugEnabled returns whether debug mode is enabled
func IsDebugEnabled() bool {
	return debugEnabled
}
