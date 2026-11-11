package render

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Output of kcl 0.13 (stderr, with its colours) for a failing assert, as
// xr-native-vm prints it (#135).
const kclAssertStderr = "EvaluationError\n" +
	" \x1b[1;38;5;12m-->\x1b[0m /root/.kcl/kpm/xr-native-vm_0.2.0/main.k:89:1\n" +
	"\x1b[1;38;5;12m   |\x1b[0m\n" +
	"\x1b[1;38;5;12m89 |\x1b[0m assert _useCase not in [\"dev\", \"github-runner\"] or _sizeName in [\"large\", \"xlarge\"], \"use_case ${_useCase} needs size large or xlarge, got ${_sizeName}\"\n" +
	"\x1b[1;38;5;12m   |\x1b[0m\x1b[1;38;5;9m \x1b[0m \x1b[1;38;5;9muse_case dev needs size large or xlarge, got small\x1b[0m\n" +
	"\x1b[1;38;5;12m   |\x1b[0m\n"

func TestKCLRunErrorAssert(t *testing.T) {
	clean, aerr := kclRunError(kclAssertStderr)
	assert.NotContains(t, clean, "\x1b")
	if assert.NotNil(t, aerr) {
		assert.Equal(t, "use_case dev needs size large or xlarge, got small", aerr.Message)
		assert.Equal(t, aerr.Message, aerr.Error())
		var target *AssertError
		assert.True(t, errors.As(error(aerr), &target))
	}
}

func TestParseAssert(t *testing.T) {
	cases := []struct {
		name, stderr, want string
		ok                 bool
	}{
		{"assert without message", "EvaluationError\n --> main.k:1:1\n  |\n1 | assert False\n  |  \n  |\n", "assertion failed: False", true},
		{"schema check", "EvaluationError\n --> s.k:5:5\n  |\n5 | s = S {a = 0}\n  |     ^ Instance check failed\n  |\n\n --> s.k:4:1\n  |\n4 |         a > 1, \"a must be > 1\"\n  |  Check failed on the condition: a must be > 1\n  |\n", "a must be > 1", true},
		{"other evaluation error", "EvaluationError\n --> idx.k:1:1\n  |\n1 | a = [1][5]\n  |  list index out of range: 5\n  |\n", "", false},
		{"syntax error", "error[E1001]: InvalidSyntax\n --> bad.k:1:7\n  |\n1 | assert x +\n  |       ^ expected one of [\"identifier\"] got newline\n  |\n", "", false},
		{"pull failure", "Error: failed to pull oci://ghcr.io/x: not found\n", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseAssert(c.stderr)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}
