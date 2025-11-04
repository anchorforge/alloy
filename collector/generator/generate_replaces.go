package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Replace represents a single "replace" directive from go.mod
type Replace struct {
	From string
	To   string
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <source-go-mod> <target-file>...", os.Args[0])
	}

	sourceGoModPath := os.Args[1]
	targetFiles := os.Args[2:]

	// Handle source path: if it's a directory, append "go.mod"
	if info, err := os.Stat(sourceGoModPath); err == nil && info.IsDir() {
		sourceGoModPath = filepath.Join(sourceGoModPath, "go.mod")
	}

	// Parse all "replace" directives from source go.mod
	replaces, err := parseGoModReplaces(sourceGoModPath)
	if err != nil {
		log.Fatalf("Failed to read replaces from %s: %v", sourceGoModPath, err)
	}

	// Filter out local or unwanted replaces
	filteredReplaces := filterReplaces(replaces)

	// Update each target file
	for _, targetPath := range targetFiles {
		if err := updateTargetFile(targetPath, filteredReplaces); err != nil {
			log.Fatalf("Failed to update %s: %v", targetPath, err)
		}
		log.Printf("✅ Updated %s with %d replace directives", targetPath, len(filteredReplaces))
	}
}

// updateTargetFile determines the file type and updates it accordingly
func updateTargetFile(targetPath string, replaces []Replace) error {
	if isGoModFile(targetPath) {
		return updateGoMod(targetPath, replaces)
	}
	return updateYAML(targetPath, replaces)
}

// isGoModFile checks if the target file is a go.mod file
func isGoModFile(path string) bool {
	return strings.HasSuffix(path, ".mod") || strings.HasSuffix(path, "go.mod")
}

// parseGoModReplaces reads the go.mod file and extracts all "replace" directives.
func parseGoModReplaces(goModPath string) ([]Replace, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open go.mod: %w", err)
	}
	defer file.Close()

	var replaces []Replace
	scanner := bufio.NewScanner(file)

	// Match both:
	//   replace module => replacement
	//   replace module v1.2.3 => replacement
	singleLinePattern := regexp.MustCompile(`^replace\s+(\S+(?:\s+\S+)?)\s+=>\s+(.+?)(?:\s+//.*)?$`)

	inBlock := false
	var blockLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "replace ("):
			inBlock = true
			blockLines = nil

		case inBlock && line == ")":
			replaces = append(replaces, parseReplaceBlock(blockLines)...)
			inBlock = false

		case inBlock:
			blockLines = append(blockLines, line)

		default:
			if match := singleLinePattern.FindStringSubmatch(line); match != nil {
				from := cleanComment(match[1])
				to := cleanComment(match[2])
				replaces = append(replaces, Replace{From: from, To: to})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading go.mod: %w", err)
	}

	return replaces, nil
}

// parseReplaceBlock parses multiple replace lines inside a "replace (...)" block.
func parseReplaceBlock(lines []string) []Replace {
	var replaces []Replace
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.Split(line, "=>")
		if len(parts) != 2 {
			continue
		}

		from := cleanComment(parts[0])
		to := cleanComment(parts[1])
		replaces = append(replaces, Replace{From: from, To: to})
	}
	return replaces
}

// cleanComment removes any trailing comments and trims spaces.
func cleanComment(s string) string {
	if idx := strings.Index(s, "//"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// filterReplaces removes unwanted replace entries.
func filterReplaces(replaces []Replace) []Replace {
	var filtered []Replace
	for _, r := range replaces {
		// Skip local paths (./ or ../)
		if strings.HasPrefix(r.To, "./") || strings.HasPrefix(r.To, "../") {
			continue
		}
		// Skip specific unwanted module
		if r.From == "github.com/grafana/alloy/syntax" {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// updateYAML injects the replace directives into a YAML file.
func updateYAML(yamlPath string, replaces []Replace) error {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("cannot read YAML file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	const marker = "# Replace directives copied from main go.mod of Alloy"

	startIdx := findLineIndex(lines, marker)
	if startIdx == -1 {
		return fmt.Errorf("marker '%s' not found in %s", marker, yamlPath)
	}

	endIdx := findYAMLReplaceSectionEnd(lines, startIdx)

	// Rebuild YAML with updated replaces
	var newContent []string
	newContent = append(newContent, lines[:startIdx+1]...)

	for _, r := range replaces {
		newContent = append(newContent, fmt.Sprintf("  - %s => %s", r.From, r.To))
	}

	newContent = append(newContent, lines[endIdx:]...)

	output := strings.Join(newContent, "\n")
	if err := os.WriteFile(yamlPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}

// updateGoMod injects the replace directives into a go.mod file.
func updateGoMod(goModPath string, replaces []Replace) error {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("cannot read go.mod file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	const marker = "// Replace directives copied from main go.mod of Alloy"

	startIdx := findLineIndex(lines, marker)
	if startIdx == -1 {
		return fmt.Errorf("marker '%s' not found in %s", marker, goModPath)
	}

	endIdx := findGoModReplaceSectionEnd(lines, startIdx)

	// Rebuild go.mod with updated replaces
	var newContent []string
	newContent = append(newContent, lines[:startIdx+1]...)
	newContent = append(newContent, "")

	for _, r := range replaces {
		newContent = append(newContent, fmt.Sprintf("replace %s => %s", r.From, r.To))
	}

	newContent = append(newContent, lines[endIdx:]...)

	output := strings.Join(newContent, "\n")
	if err := os.WriteFile(goModPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write go.mod file: %w", err)
	}

	return nil
}

// findLineIndex finds the index of the first line containing the given substring.
func findLineIndex(lines []string, substr string) int {
	for i, line := range lines {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

// findYAMLReplaceSectionEnd determines where the "replace" section ends in a YAML file.
func findYAMLReplaceSectionEnd(lines []string, startIdx int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		// The section ends when we find a top-level key (not indented)
		if line != "" && !strings.HasPrefix(line, "  ") {
			return i
		}
	}
	return len(lines)
}

// findGoModReplaceSectionEnd determines where the replace section ends in a go.mod file.
func findGoModReplaceSectionEnd(lines []string, startIdx int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		// The section ends when we find a new top-level directive (module, require, exclude, etc.)
		if line == "" {
			// Check if next non-empty line is a top-level directive
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				if isTopLevelDirective(nextLine) {
					return i
				}
				break
			}
		} else if isTopLevelDirective(line) {
			return i
		}
	}
	return len(lines)
}

// isTopLevelDirective checks if a line is a top-level go.mod directive
func isTopLevelDirective(line string) bool {
	return strings.HasPrefix(line, "module ") ||
		strings.HasPrefix(line, "require ") ||
		strings.HasPrefix(line, "exclude ") ||
		strings.HasPrefix(line, "go ")
}
