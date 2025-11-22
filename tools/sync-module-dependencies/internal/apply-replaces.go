package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/grafana/replace-generator/internal/files"
	"github.com/grafana/replace-generator/internal/types"
)

func main() {
	fileHelper := files.GetFileHelper()

	projectReplaces, err := fileHelper.LoadProjectReplaces()
	if err != nil {
		log.Fatalf("Failed to load project replaces: %v", err)
	}

	for _, module := range projectReplaces.Modules {
		if err := applyReplacesToModule(fileHelper, module); err != nil {
			log.Fatalf("Failed to apply replaces to module %q: %v", module.Name, err)
		}
		log.Printf("Updated %s", module.Path)
	}
}

func applyReplacesToModule(dirs *files.FileHelper, module types.Module) error {
	targetPath := dirs.ModuleTargetPath(module.Path)
	replacesPath := dirs.ModuleOutputPath(module.OutputFile)

	replacesContent, err := os.ReadFile(replacesPath)
	if err != nil {
		return fmt.Errorf("read replaces file %s: %w", replacesPath, err)
	}

	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read target file %s: %w", targetPath, err)
	}

	startMarker, endMarker, err := getMarkers(module.FileType)
	if err != nil {
		return fmt.Errorf("get markers for file type %q: %w", module.FileType, err)
	}

	newContent := prepareContent(string(targetContent), string(replacesContent), startMarker, endMarker)

	if err := os.WriteFile(targetPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("write target file %s: %w", targetPath, err)
	}

	return nil
}

func prepareContent(targetContent, replacesContent, startMarker, endMarker string) string {
	markers := hasMarkers(targetContent, startMarker, endMarker)

	var newContent string
	if markers {
		newContent = removeBetweenMarkers(targetContent, startMarker, endMarker)
	} else {
		newContent = targetContent
	}

	return strings.TrimRight(newContent, "\n") + "\n" + replacesContent
}

func getMarkers(fileType string) (startMarker, endMarker string, err error) {
	switch fileType {
	case "mod":
		// Go mod files use // comments
		return "// BEGIN GENERATED REPLACES - DO NOT EDIT", "// END GENERATED REPLACES", nil
	default:
		return "", "", fmt.Errorf("unknown file_type %q", fileType)
	}
}

func hasMarkers(content, startMarker, endMarker string) bool {
	startMarkerTrimmed := strings.TrimSpace(startMarker)
	endMarkerTrimmed := strings.TrimSpace(endMarker)

	hasStart := false
	hasEnd := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		trimmedLine := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmedLine, startMarkerTrimmed) {
			hasStart = true
		}
		if trimmedLine == endMarkerTrimmed {
			hasEnd = true
		}
	}

	return hasStart && hasEnd
}

func removeBetweenMarkers(content, startMarker, endMarker string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	inMarkerBlock := false

	startMarkerTrimmed := strings.TrimSpace(startMarker)
	endMarkerTrimmed := strings.TrimSpace(endMarker)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if !inMarkerBlock && strings.HasPrefix(trimmedLine, startMarkerTrimmed) {
			inMarkerBlock = true
			continue // Skip the start marker line
		}

		if inMarkerBlock && trimmedLine == endMarkerTrimmed {
			inMarkerBlock = false
			continue // Skip the end marker line
		}

		if !inMarkerBlock {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}
