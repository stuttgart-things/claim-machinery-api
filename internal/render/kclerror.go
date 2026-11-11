package render

import (
	"regexp"
	"strings"
)

// AssertError is a KCL run that failed on an `assert` or a schema `check`
// of the module: the parameters break one of its rules. Message is only
// the rule's message (no stack, no ANSI codes); the API answers it with
// 400 (#135). Other KCL failures stay ordinary errors (500).
type AssertError struct {
	Message string
	// Stderr is kcl's whole error output, ANSI codes removed, for the log.
	Stderr string
}

func (e *AssertError) Error() string { return e.Message }

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// StripANSI removes terminal colour codes from kcl's output.
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

var (
	// "3 | assert x, "msg"": the source line of a diagnostic.
	kclSourceLine = regexp.MustCompile(`^\s*\d+\s*\|\s?(.*)$`)
	// "  |   ^ message": the marker line below it.
	kclMarkerLine = regexp.MustCompile(`^\s*\|(.*)$`)
)

const kclCheckFailed = "Check failed on the condition:"

// parseAssert finds a failed assert or schema check in kcl's error output
// (ANSI already stripped) and returns its message. kcl 0.x prints
//
//	EvaluationError
//	 --> /path/main.k:89:1
//	   |
//	89 | assert _useCase ..., "use_case dev needs size large or xlarge, got small"
//	   |  use_case dev needs size large or xlarge, got small
//
// and for a schema check "Check failed on the condition: <msg>" below the
// check's line. An assert without a message gets "assertion failed: <expr>".
func parseAssert(stderr string) (string, bool) {
	if !strings.Contains(stderr, "EvaluationError") {
		return "", false
	}
	lines := strings.Split(stderr, "\n")
	for i, line := range lines {
		m := kclSourceLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		src := strings.TrimSpace(m[1])
		msg := ""
		if i+1 < len(lines) {
			if mm := kclMarkerLine.FindStringSubmatch(lines[i+1]); mm != nil {
				msg = strings.TrimSpace(strings.TrimLeft(mm[1], " ^~"))
			}
		}
		if rest, ok := strings.CutPrefix(msg, kclCheckFailed); ok {
			return strings.TrimSpace(rest), true
		}
		if src == "assert" || strings.HasPrefix(src, "assert ") {
			if msg == "" {
				return "assertion failed: " + strings.TrimSpace(strings.TrimPrefix(src, "assert")), true
			}
			return msg, true
		}
	}
	return "", false
}

// kclRunError returns kcl's error output without ANSI codes and, when a
// rule of the module failed, the *AssertError for it (else nil, and the
// caller keeps its own error).
func kclRunError(stderr string) (clean string, assert *AssertError) {
	clean = StripANSI(stderr)
	if msg, ok := parseAssert(clean); ok {
		return clean, &AssertError{Message: msg, Stderr: clean}
	}
	return clean, nil
}
