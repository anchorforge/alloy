package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grafana/replace-generator/internal/files"
	"github.com/grafana/replace-generator/internal/types"
)

func main() {
	fileHelper := files.GetFileHelper()

	projectReplaces, err := fileHelper.LoadProjectReplaces()
	if err != nil {
		log.Fatalf("Failed to load project replaces: %v", err)
	}

	normalizeComments(projectReplaces.Replaces)

	for _, module := range projectReplaces.Modules {
		if err := generateModuleReplaces(fileHelper, projectReplaces, module); err != nil {
			log.Fatalf("Failed to generate replaces for module %q: %v", module.Name, err)
		}
	}
}

// Removes unnecessary newlines and whitespaces from comment metadata
func normalizeComments(entries []types.ReplaceEntry) {
	for i := range entries {
		entries[i].Comment = strings.ReplaceAll(entries[i].Comment, "\n", " ")
		entries[i].Comment = strings.TrimSpace(entries[i].Comment)
	}
}

// Generates the .txt files that contain the replace directives formatted into their corresponding template
// For types.Module we will output a temp replaces-mod.txt file generated from the replaces-mod.tpl
func generateModuleReplaces(dirs *files.FileHelper, projectReplaces *types.ProjectReplaces, module types.Module) error {
	templatePath := dirs.TemplatePath(module.FileType)
	outputPath := dirs.ModuleOutputPath(module.OutputFile)

	err := generateFromTemplate(templatePath, projectReplaces.Replaces, outputPath)
	if err != nil {
		return fmt.Errorf("could not execute template generation: %w", err)
	}

	return nil
}

func generateFromTemplate(templatePath string, entries []types.ReplaceEntry, outputPath string) error {
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("could not read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(tmplContent))

	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, entries); err != nil {
		return fmt.Errorf("could not generate template: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("could not write output file: %w", err)
	}

	return nil
}
