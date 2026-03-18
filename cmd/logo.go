package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

const logoRaw = `
███╗   ███╗ █████╗  ██████╗██╗  ██╗██╗███╗   ██╗███████╗██████╗ ██╗   ██╗
████╗ ████║██╔══██╗██╔════╝██║  ██║██║████╗  ██║██╔════╝██╔══██╗╚██╗ ██╔╝
██╔████╔██║███████║██║     ███████║██║██╔██╗ ██║█████╗  ██████╔╝ ╚████╔╝
██║╚██╔╝██║██╔══██║██║     ██╔══██║██║██║╚██╗██║██╔══╝  ██╔══██╗  ╚██╔╝
██║ ╚═╝ ██║██║  ██║╚██████╗██║  ██║██║██║ ╚████║███████╗██║  ██║   ██║
╚═╝     ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝   ╚═╝
`

var (
	gradientStart = "#ff00ff" // neon magenta
	gradientEnd   = "#00ffff" // electric cyan
)

func renderLogo() string {
	lines := strings.Split(strings.TrimPrefix(logoRaw, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}

	// Find the max width for gradient calculation
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	startColor, _ := colorful.Hex(gradientStart)
	endColor, _ := colorful.Hex(gradientEnd)

	var result strings.Builder
	result.WriteString("\n") // Add spacing before logo
	for _, line := range lines {
		for i, char := range line {
			if char == ' ' {
				result.WriteRune(char)
				continue
			}
			// Calculate gradient position based on horizontal position
			t := float64(i) / float64(maxWidth)
			c := startColor.BlendLuv(endColor, t)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex()))
			result.WriteString(style.Render(string(char)))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// Logo returns the rendered logo with gradient colors
var logo = renderLogo()

// --- Styled output helpers ---------------------------------------------------

var (
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffff")).
			PaddingTop(1)

	methodStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff00ff"))

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffff"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

func renderVersionBadge(version, commit, date string) string {
	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00ffff")).
		Bold(true).
		Padding(0, 1).
		Render("VERSION")
	value := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Render(version)

	commitLabel := dimStyle.Render(" commit:") + " " + commit
	dateLabel := dimStyle.Render(" built:") + " " + date

	return "  " + label + value + commitLabel + dateLabel
}

func printSection(title string) {
	fmt.Println(sectionStyle.Render("  " + title))
}

func printKV(key, value string) {
	// Pad the key before styling so ANSI codes don't break alignment
	padded := fmt.Sprintf("%-24s", key)
	fmt.Printf("  │ %s %s\n", dimStyle.Render(padded), value)
}

func printEnvTable(vars []struct{ name, desc, fallback string }) {
	// Compute column widths
	nameW, descW := len("VARIABLE"), len("DESCRIPTION")
	for _, v := range vars {
		if len(v.name) > nameW {
			nameW = len(v.name)
		}
		if len(v.desc) > descW {
			descW = len(v.desc)
		}
	}

	hdr := fmt.Sprintf("  %-*s  %-*s  %s", nameW, "VARIABLE", descW, "DESCRIPTION", "VALUE")
	fmt.Println(dimStyle.Render(hdr))
	fmt.Println(dimStyle.Render("  " + strings.Repeat("─", nameW) + "  " + strings.Repeat("─", descW) + "  " + strings.Repeat("─", 20)))

	setStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))

	for _, v := range vars {
		val := os.Getenv(v.name)
		display := dimStyle.Render(v.fallback)
		if val != "" {
			// Mask sensitive values
			if strings.Contains(strings.ToUpper(v.name), "TOKEN") || strings.Contains(strings.ToUpper(v.name), "SECRET") {
				display = setStyle.Render("****")
			} else {
				display = setStyle.Render(val)
			}
		}
		nameCol := fmt.Sprintf("%-*s", nameW, v.name)
		descCol := fmt.Sprintf("%-*s", descW, v.desc)
		fmt.Printf("  %s  %s  %s\n",
			pathStyle.Render(nameCol),
			dimStyle.Render(descCol),
			display,
		)
	}
}

func printTemplateTable(templates [][3]string) {
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	labelImport := dimStyle.Render("import")
	labelOCI := dimStyle.Render("oci")

	for i, t := range templates {
		fmt.Printf("  %s\n", nameStyle.Render(t[0]))
		fmt.Printf("    %s  %s\n", labelImport, dimStyle.Render(t[2]))
		fmt.Printf("    %s     %s\n", labelOCI, pathStyle.Render(t[1]))
		if i < len(templates)-1 {
			fmt.Println()
		}
	}
}

func printEndpoint(method, path, desc string) {
	fmt.Printf("  │ %s  %-42s %s\n",
		methodStyle.Render(method),
		pathStyle.Render(path),
		dimStyle.Render(desc),
	)
}
