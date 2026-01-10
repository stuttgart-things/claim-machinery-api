package render

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceTripleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    `'''4'''`,
			expected: `'4'`,
		},
		{
			name:     "string value",
			input:    `'''helloopo'''`,
			expected: `'helloopo'`,
		},
		{
			name:     "with surrounding text",
			input:    `count: '''4'''`,
			expected: `count: '4'`,
		},
		{
			name:     "multiple replacements",
			input:    `count: '''4''', name: '''helloopo'''`,
			expected: `count: '4', name: 'helloopo'`,
		},
		{
			name:     "no replacement needed",
			input:    `count: '4'`,
			expected: `count: '4'`,
		},
		{
			name:     "empty value",
			input:    `count: ''''''`,
			expected: `count: ''`,
		},
		{
			name:     "mixed quotes",
			input:    `count: '''4''', name: "hello"`,
			expected: `count: '4', name: "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceTripleQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFixQuotesInMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "simple values",
			input: map[string]string{
				"count": "'''4'''",
				"name":  "'''helloopo'''",
			},
			expected: map[string]string{
				"count": "'4'",
				"name":  "'helloopo'",
			},
		},
		{
			name: "mixed quotes",
			input: map[string]string{
				"cpu":   "'''8'''",
				"ram":   "'''4'''",
				"disk":  "'1'",    // already single quoted
				"label": `"fast"`, // double quoted
			},
			expected: map[string]string{
				"cpu":   "'8'",
				"ram":   "'4'",
				"disk":  "'1'",
				"label": `"fast"`,
			},
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixQuotesInMap(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fixQuotesInMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("malformed triple quotes", func(t *testing.T) {
		input := `'''unmatched`
		result := replaceTripleQuotes(input)
		assert.Equal(t, input, result) // Should remain unchanged
	})

	t.Run("nested quotes", func(t *testing.T) {
		input := `'''"value"'''`
		result := replaceTripleQuotes(input)
		assert.Equal(t, `'"value"'`, result)
	})

	t.Run("real world example", func(t *testing.T) {
		input := `config: {count: '''4''', name: '''hello''', nested: {value: '''test'''}}`
		expected := `config: {count: '4', name: 'hello', nested: {value: 'test'}}`
		result := replaceTripleQuotes(input)
		assert.Equal(t, expected, result)
	})
}

func TestRenderKCL(t *testing.T) {
	// Create temporary directory for test KCL file
	tmpDir := t.TempDir()

	// Create a simple KCL file that accepts parameters
	kclContent := `
_params = option("params") or {}

result = {
	name = option("name") or "default"
	version = option("version") or "1.0"
	count = option("count") or "0"
}
`

	kclFile := tmpDir + "/test.k"
	err := os.WriteFile(kclFile, []byte(kclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test KCL file: %v", err)
	}

	tests := []struct {
		name    string
		answers map[string]interface{}
	}{
		{
			name: "with parameters",
			answers: map[string]interface{}{
				"name":    "myapp",
				"version": "2.0",
				"count":   "5",
			},
		},
		{
			name:    "empty parameters",
			answers: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderKCL(kclFile, tt.answers)
			assert.NotEmpty(t, result, "Expected non-empty YAML output")
			assert.Contains(t, result, "result:", "Expected 'result:' in output")
		})
	}
}

func TestConvertToOptionStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		wantLen  int
		wantKeys []string
	}{
		{
			name: "string values",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			wantLen:  2,
			wantKeys: []string{"key1=", "key2="},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			wantLen:  0,
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToOptionStrings(tt.input)
			assert.Len(t, result, tt.wantLen, "Expected correct number of options")

			for _, opt := range result {
				assert.Contains(t, opt, "=", "Expected 'key=value' format")
			}
		})
	}
}

func TestRenderKCLFromOCI(t *testing.T) {
	tests := []struct {
		name    string
		oci     string
		tag     string
		answers map[string]interface{}
	}{
		{
			name: "with tag and parameters",
			oci:  "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
			tag:  "0.1.1",
			answers: map[string]interface{}{
				"templateName":     "simple",
				"namespace":        "production",
				"storage":          "10Gi",
				"storageClassName": "fast",
			},
		},
		{
			name: "without tag",
			oci:  "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
			tag:  "",
			answers: map[string]interface{}{
				"namespace": "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will execute KCL against actual OCI source
			// Comment out if offline - it requires internet access
			result := RenderKCLFromOCI(tt.oci, tt.tag, tt.answers)
			assert.NotEmpty(t, result, "Expected non-empty YAML output from OCI source")
		})
	}
}

func TestRenderKCLToFile(t *testing.T) {
	// Create temporary directory for test KCL file and output
	tmpDir := t.TempDir()

	// Create a simple KCL file
	kclContent := `
_params = option("params") or {}

result = {
	name = option("name") or "default"
	version = option("version") or "1.0"
}
`

	kclFile := tmpDir + "/test.k"
	err := os.WriteFile(kclFile, []byte(kclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test KCL file: %v", err)
	}

	outputFile := tmpDir + "/output.yaml"
	answers := map[string]interface{}{
		"name":    "testapp",
		"version": "1.5",
	}

	// Call RenderKCLToFile
	result, err := RenderKCLToFile(kclFile, answers, outputFile)

	// Verify no error
	assert.NoError(t, err, "Expected no error writing to file")

	// Verify result is not empty
	assert.NotEmpty(t, result, "Expected non-empty YAML output")

	// Verify file was created and contains data
	fileContent, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Expected to read output file")
	assert.Equal(t, result, string(fileContent), "Expected file content to match returned YAML")
}

func TestRenderKCLFromOCIToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/oci_output.yaml"

	answers := map[string]interface{}{
		"namespace": "default",
	}

	// Call RenderKCLFromOCIToFile
	result, err := RenderKCLFromOCIToFile(
		"oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
		"0.1.1",
		answers,
		outputFile,
	)

	// Verify no error
	assert.NoError(t, err, "Expected no error writing to file")

	// Verify result is not empty
	assert.NotEmpty(t, result, "Expected non-empty YAML output from OCI source")

	// Verify file was created and contains data
	fileContent, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Expected to read output file")
	assert.Equal(t, result, string(fileContent), "Expected file content to match returned YAML")
}
