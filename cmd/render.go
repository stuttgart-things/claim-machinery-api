package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

const randomMarker = "🎲 Random"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	yamlStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1)
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Interactive template rendering",
	Long:  `Start an interactive CLI to select and render claim templates.`,
	RunE:  runRender,
}

func init() {
	rootCmd.AddCommand(renderCmd)
}

func runRender(cmd *cobra.Command, args []string) error {
	fmt.Println(logo)
	fmt.Printf("Version:    %s\n", version)
	fmt.Printf("Commit:     %s\n", commit)
	fmt.Printf("Build Date: %s\n\n", date)

	// Load templates directory (flag > env > default)
	if templatesDir == "" {
		templatesDir = os.Getenv("TEMPLATES_DIR")
	}
	if templatesDir == "" {
		templatesDir = filepath.Join(
			"internal",
			"claimtemplate",
			"testdata",
		)
	}

	// Load profile path
	if profilePath == "" {
		profilePath = os.Getenv("TEMPLATE_PROFILE_PATH")
	}
	if profilePath == "" {
		profilePath = "tests/profile.yaml"
	}

	var templates []*claimtemplate.ClaimTemplate
	var sources []string

	if profilePath == "" {
		// Load from directory only
		dirTemplates, err := app.LoadAllTemplates(templatesDir)
		if err != nil {
			fmt.Printf("❌ Failed to load templates from directory: %v\n", err)
			os.Exit(1)
		}
		templates = dirTemplates
		sources = []string{templatesDir}
	} else {
		// Combine directory templates with profile templates
		dirTemplates, err1 := app.LoadAllTemplates(templatesDir)
		if err1 != nil {
			fmt.Printf("❌ Failed to load templates from directory: %v\n", err1)
			os.Exit(1)
		}
		profileTemplates, profileSources, err2 := app.LoadTemplatesFromProfile(profilePath)
		if err2 != nil {
			fmt.Printf("❌ Failed to load templates from profile: %v\n", err2)
			os.Exit(1)
		}

		// Merge, de-duplicate by metadata.name (profile overrides directory on conflict)
		merged := make(map[string]*claimtemplate.ClaimTemplate)
		for _, t := range dirTemplates {
			merged[t.Metadata.Name] = t
		}
		for _, t := range profileTemplates {
			merged[t.Metadata.Name] = t
		}
		templates = make([]*claimtemplate.ClaimTemplate, 0, len(merged))
		for _, t := range merged {
			templates = append(templates, t)
		}

		sources = append([]string{templatesDir}, profileSources...)
	}

	fmt.Printf("📂 Loaded %d templates\n\n", len(templates))
	for _, s := range sources {
		fmt.Printf("   • %s\n", s)
	}
	fmt.Println()

	// Build template map and options for selection
	templateMap := make(map[string]*claimtemplate.ClaimTemplate)
	var templateOptions []huh.Option[string]

	for _, t := range templates {
		templateMap[t.Metadata.Name] = t
		label := fmt.Sprintf("%s - %s", t.Metadata.Name, t.Metadata.Title)
		templateOptions = append(templateOptions, huh.NewOption(label, t.Metadata.Name))
	}

	// Step 1: Select template
	var selectedTemplate string
	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a template").
				Description("Choose which claim template to render").
				Options(templateOptions...).
				Value(&selectedTemplate),
		),
	)

	if err := selectForm.Run(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	tmpl := templateMap[selectedTemplate]
	fmt.Printf("\n%s\n", titleStyle.Render("📋 "+tmpl.Metadata.Title))
	fmt.Printf("%s\n\n", tmpl.Metadata.Description)

	// Step 2: Build dynamic form based on template parameters
	params := make(map[string]interface{})
	paramValues := make(map[string]*string)

	// Create form fields for each parameter
	var formGroups []*huh.Group
	var currentFields []huh.Field
	var visibleCount int

	for _, p := range tmpl.Spec.Parameters {
		// Create a string pointer to hold the value (including hidden params)
		defaultVal := ""
		if p.Default != nil {
			defaultVal = fmt.Sprintf("%v", p.Default)
		}
		paramValues[p.Name] = &defaultVal

		// Skip hidden parameters - they use their default value
		if p.Hidden {
			continue
		}

		visibleCount++
		field := createField(p, paramValues[p.Name])
		if field != nil {
			currentFields = append(currentFields, field)
		}

		// Group fields (max 5 per group for better UX)
		if len(currentFields) >= 5 {
			formGroups = append(formGroups, huh.NewGroup(currentFields...))
			currentFields = nil
		}
	}

	// Add remaining fields as final group
	if len(currentFields) > 0 {
		formGroups = append(formGroups, huh.NewGroup(currentFields...))
	}

	// Run the parameter form
	if len(formGroups) > 0 {
		paramForm := huh.NewForm(formGroups...)
		if err := paramForm.Run(); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Resolve random selections and pass all values as strings
	rand.Seed(time.Now().UnixNano())
	for _, p := range tmpl.Spec.Parameters {
		strVal := *paramValues[p.Name]
		if strVal == "" {
			continue
		}

		// Resolve "Random" to actual random value from enum
		if strVal == randomMarker && len(p.Enum) > 0 {
			randomIdx := rand.Intn(len(p.Enum))
			strVal = p.Enum[randomIdx]
			fmt.Printf("🎲 Random selection for %s: %s\n", p.Name, strVal)
		}

		// Keep all values as strings - KCL schema expects string types
		params[p.Name] = strVal
	}

	// Step 3: Confirm and render (default: Yes)
	confirm := true
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Render the claim?").
				Description("This will generate the YAML using KCL").
				Affirmative("Yes, render it").
				Negative("Cancel").
				Value(&confirm),
		),
	)

	if err := confirmForm.Run(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	if !confirm {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	// Render using KCL
	fmt.Println("\n⏳ Rendering with KCL...")

	// Merge with defaults
	allParams := app.BuildParameterValues(tmpl)
	for k, v := range params {
		allParams[k] = v
	}

	// Convert all values to strings (KCL expects string types)
	stringParams := make(map[string]interface{})
	for k, v := range allParams {
		stringParams[k] = fmt.Sprintf("%v", v)
	}

	result := render.RenderKCLFromOCI(tmpl.Spec.Source, tmpl.Spec.Tag, stringParams)

	fmt.Println(successStyle.Render("\n✅ Rendered successfully!"))
	fmt.Println(yamlStyle.Render(result))

	// Generate default save path: /tmp/{templateName}-{resourceName}.yaml
	resourceName := "output"
	if name, ok := stringParams["name"]; ok {
		resourceName = fmt.Sprintf("%v", name)
	}
	defaultSavePath := fmt.Sprintf("/tmp/%s-%s.yaml", tmpl.Metadata.Name, resourceName)

	// Ask to save (with default path)
	savePath := defaultSavePath
	saveForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Save to file?").
				Description("Press Enter to use default, or clear to skip").
				Value(&savePath),
		),
	)

	if err := saveForm.Run(); err == nil && savePath != "" {
		if err := os.WriteFile(savePath, []byte(result), 0644); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
		} else {
			fmt.Printf("💾 Saved to %s\n", savePath)
		}
	}

	return nil
}

// createField creates the appropriate huh field based on parameter type
func createField(p claimtemplate.Parameter, value *string) huh.Field {
	title := p.Title
	if p.Required {
		title += " *"
	}

	description := p.Description
	if p.Pattern != "" {
		description += fmt.Sprintf(" (pattern: %s)", p.Pattern)
	}

	// If parameter has enum values, use Select
	if len(p.Enum) > 0 {
		var options []huh.Option[string]

		// Add Random option if allowed
		if p.AllowRandom {
			options = append(options, huh.NewOption(randomMarker, randomMarker))
		}

		for _, e := range p.Enum {
			enumStr := fmt.Sprintf("%v", e)
			options = append(options, huh.NewOption(enumStr, enumStr))
		}

		return huh.NewSelect[string]().
			Title(title).
			Description(description).
			Options(options...).
			Value(value)
	}

	// Handle different types
	switch p.Type {
	case "boolean":
		// Use select for boolean
		return huh.NewSelect[string]().
			Title(title).
			Description(description).
			Options(
				huh.NewOption("true", "true"),
				huh.NewOption("false", "false"),
			).
			Value(value)

	case "integer":
		return huh.NewInput().
			Title(title).
			Description(description).
			Placeholder(fmt.Sprintf("default: %v", p.Default)).
			Value(value).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				if _, err := strconv.Atoi(s); err != nil {
					return fmt.Errorf("must be a number")
				}
				return nil
			})

	default: // string
		input := huh.NewInput().
			Title(title).
			Description(description).
			Placeholder(fmt.Sprintf("default: %v", p.Default)).
			Value(value)

		// Add pattern validation if specified
		if p.Pattern != "" {
			// Note: For MVP, we skip regex validation to keep it simple
			// In production, add regexp validation here
		}

		return input
	}
}
