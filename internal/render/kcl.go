package render

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	kcl "kcl-lang.io/kcl-go"
)

func RenderKCL(
	kclFile string,
	allAnswers map[string]interface{}) (string, error) {

	// READ MAIN KCL FILE
	content, err := os.ReadFile(kclFile)
	if err != nil {
		return "", fmt.Errorf("error reading KCL file: %w", err)
	}

	// OUTPUT ALL ANSWERS + MODIFY
	for key, value := range allAnswers {
		fmt.Printf("%s=%v\n", key, value)
	}

	values := convertToOptionStrings(allAnswers)

	// Prepare KCL options with explicit key-value pairs
	opts := []kcl.Option{
		kcl.WithCode(string(content)),
		kcl.WithOptions(values...),
	}

	// Execute KCL
	result, err := kcl.Run(kclFile, opts...)
	if err != nil {
		return "", fmt.Errorf("KCL execution failed: %w", err)
	}

	// Output generated YAML
	return replaceTripleQuotes(result.GetRawYamlResult()), nil
}

// RenderKCLToFile renders KCL and writes output to both stdout and file
func RenderKCLToFile(
	kclFile string,
	allAnswers map[string]interface{},
	destination string) (string, error) {

	// Get rendered YAML
	yaml, err := RenderKCL(kclFile, allAnswers)
	if err != nil {
		return "", fmt.Errorf("failed to render KCL: %w", err)
	}

	// Write to file
	err = os.WriteFile(destination, []byte(yaml), 0644)
	if err != nil {
		return yaml, fmt.Errorf("failed to write YAML to file %s: %w", destination, err)
	}

	// Also write to stdout
	fmt.Printf("\n--- Rendered YAML output ---\n%s\n--- Written to: %s ---\n", yaml, destination)

	return yaml, nil
}

// RenderKCLFromOCI renders KCL from an OCI source (e.g., oci://ghcr.io/...)
func RenderKCLFromOCI(
	ociSource string,
	tag string,
	allAnswers map[string]interface{}) (string, error) {

	// Build command: kcl run <oci-source> -D key=value ...
	args := []string{"run", "--quiet"}

	// Add OCI source and tag
	if tag != "" {
		args = append(args, ociSource, "--tag", tag)
	} else {
		args = append(args, ociSource)
	}

	// Add parameters as -D flags
	for key, value := range allAnswers {
		fmt.Printf("%s=%v\n", key, value)
		args = append(args, "-D", formatKCLFlag(key, value))
	}

	// Execute kcl CLI command
	cmd := exec.Command("kcl", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("KCL execution from OCI source failed: %w\nStderr: %s", err, stderr.String())
	}

	// Output generated YAML
	return replaceTripleQuotes(stdout.String()), nil
}

// RenderKCLFromOCIToFile renders KCL from OCI source and writes output to both stdout and file
func RenderKCLFromOCIToFile(
	ociSource string,
	tag string,
	allAnswers map[string]interface{},
	destination string) (string, error) {

	// Get rendered YAML
	yaml, err := RenderKCLFromOCI(ociSource, tag, allAnswers)
	if err != nil {
		return "", fmt.Errorf("failed to render KCL from OCI: %w", err)
	}

	// Write to file
	err = os.WriteFile(destination, []byte(yaml), 0644)
	if err != nil {
		return yaml, fmt.Errorf("failed to write YAML to file %s: %w", destination, err)
	}

	// Also write to stdout
	fmt.Printf("\n--- Rendered YAML output ---\n%s\n--- Written to: %s ---\n", yaml, destination)

	return yaml, nil
}

func convertToOptionStrings(answers map[string]interface{}) []string {
	var options []string

	for key, value := range answers {
		// Convert the value to string
		var strValue string
		switch v := value.(type) {
		case []string:
			strValue = formatKCLList(v)
		case []interface{}:
			items := make([]string, 0, len(v))
			for _, item := range v {
				items = append(items, fmt.Sprintf("%v", item))
			}
			strValue = formatKCLList(items)
		case string:
			strValue = "'" + v + "'"
		}

		// Create the "key=value" string and add to slice
		options = append(options, fmt.Sprintf("%s=%s", key, strValue))
	}

	return options
}

// formatKCLFlag formats a key-value pair as a KCL -D flag string.
func formatKCLFlag(key string, value interface{}) string {
	switch v := value.(type) {
	case []string:
		return fmt.Sprintf("%s=%s", key, formatKCLList(v))
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return fmt.Sprintf("%s=%s", key, formatKCLList(items))
	default:
		return fmt.Sprintf("%s=%v", key, value)
	}
}

// formatKCLList formats a string slice as a KCL list literal, e.g. ["a","b"].
func formatKCLList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, `"`+item+`"`)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

// replaceTripleQuotes replaces ”'value”' with 'value' in a string
func replaceTripleQuotes(input string) string {
	// Updated regex to handle empty values
	re := regexp.MustCompile(`'''([^']*)'''`)
	return re.ReplaceAllString(input, `'$1'`)
}

// fixQuotesInMap processes a map replacing ”'value”' with 'value' in all values
func fixQuotesInMap(data map[string]string) map[string]string {
	re := regexp.MustCompile(`'''([^']*)'''`)
	result := make(map[string]string, len(data))

	for k, v := range data {
		result[k] = re.ReplaceAllString(v, `'$1'`)
	}
	return result
}
