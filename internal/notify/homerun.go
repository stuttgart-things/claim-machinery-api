package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// HomerunMessage represents a message for the homerun2 omni-pitcher /pitch endpoint.
type HomerunMessage struct {
	Title           string `json:"title"`
	Message         string `json:"message"`
	Severity        string `json:"severity,omitempty"`
	Author          string `json:"author,omitempty"`
	Timestamp       string `json:"timestamp,omitempty"`
	System          string `json:"system,omitempty"`
	Tags            string `json:"tags,omitempty"`
	AssigneeAddress string `json:"assigneeaddress,omitempty"`
	AssigneeName    string `json:"assigneename,omitempty"`
	Artifacts       string `json:"artifacts,omitempty"`
	Url             string `json:"url,omitempty"`
}

// HomerunConfig holds the configuration for homerun2 notifications.
type HomerunConfig struct {
	Enabled   bool
	URL       string
	AuthToken string
}

// LoadHomerunConfig reads homerun configuration from environment variables.
func LoadHomerunConfig() HomerunConfig {
	enabled := isTruthy(os.Getenv("ENABLE_HOMERUN"))
	url := os.Getenv("HOMERUN_URL")
	token := os.Getenv("HOMERUN_AUTH_TOKEN")

	return HomerunConfig{
		Enabled:   enabled && url != "",
		URL:       url,
		AuthToken: token,
	}
}

// SendHomerunMessage sends a message to the homerun2 omni-pitcher.
// It is safe to call even when homerun is disabled (no-op).
// On failure it logs a warning but never returns an error.
func SendHomerunMessage(cfg HomerunConfig, msg HomerunMessage) {
	if !cfg.Enabled {
		return
	}

	if msg.Timestamp == "" {
		msg.Timestamp = time.Now().Format(time.RFC3339)
	}
	if msg.System == "" {
		msg.System = "claim-machinery-api"
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("homerun: failed to marshal message: %v", err)
		return
	}

	pitchURL := strings.TrimRight(cfg.URL, "/") + "/pitch"

	req, err := http.NewRequest(http.MethodPost, pitchURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("homerun: failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("homerun: failed to send message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("homerun: pitcher returned status %s", resp.Status)
		return
	}

	log.Printf("homerun: notification sent (%s)", msg.Title)
}

// BuildOrderMessage constructs a HomerunMessage for a successful template order.
func BuildOrderMessage(
	templateName string,
	templateTitle string,
	templateType string,
	templateTags []string,
	templateSource string,
	templateTag string,
	params map[string]interface{},
	author string,
) HomerunMessage {
	if author == "" {
		author = "claim-machinery-api"
	}

	// Build parameter summary
	var paramLines []string
	for k, v := range params {
		paramLines = append(paramLines, fmt.Sprintf("  %s: %v", k, v))
	}

	title := templateTitle
	if title == "" {
		title = templateName
	}

	msgBody := fmt.Sprintf("Template '%s' ordered successfully.", title)
	if len(paramLines) > 0 {
		msgBody += "\n\nParameters:\n" + strings.Join(paramLines, "\n")
	}

	// Build tags: claim-order + type + template tags
	tagParts := []string{"claim-order"}
	if templateType != "" {
		tagParts = append(tagParts, templateType)
	}
	tagParts = append(tagParts, templateTags...)

	// Build artifacts from OCI source
	artifacts := templateSource
	if templateTag != "" {
		artifacts += ":" + templateTag
	}

	return HomerunMessage{
		Title:    "Claim Order: " + templateName,
		Message:  msgBody,
		Severity: "success",
		Author:   author,
		System:   "claim-machinery-api",
		Tags:     strings.Join(tagParts, ","),
		Artifacts: artifacts,
	}
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}
