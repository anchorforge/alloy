package files

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/grafana/replace-generator/internal/types"
	"gopkg.in/yaml.v3"
)

type FileHelper struct {
	// ScriptDir is the directory where the sync-module-dependencies tools are located.
	// This is where templates and output files are stored.
	ScriptDir string

	// ProjectRoot is the root directory of the Alloy project.
	ProjectRoot string

	// ProjectReplacesPath is the absolute path to dependency-replacements.yaml.
	ProjectReplacesPath string
}

var fileHelper *FileHelper

func GetFileHelper() *FileHelper {
	if fileHelper == nil {
		fileHelper = newFileHelper()
	}
	return fileHelper
}

func newFileHelper() *FileHelper {
	scriptDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to resolve working directory: %v", err)
	}

	scriptDir, err = filepath.Abs(scriptDir)
	if err != nil {
		log.Fatalf("Failed to resolve script directory: %v", err)
	}

	projectRoot, err := filepath.Abs(filepath.Join(scriptDir, "..", ".."))
	if err != nil {
		log.Fatalf("Failed to resolve project root: %v", err)
	}

	projectReplacesPath := filepath.Join(projectRoot, "dependency-replacements.yaml")

	absReplacesPath, err := filepath.Abs(projectReplacesPath)
	if err != nil {
		log.Fatalf("Failed to resolve dependency-replacements.yaml: %v", err)
	}

	return &FileHelper{
		ScriptDir:           scriptDir,
		ProjectRoot:         projectRoot,
		ProjectReplacesPath: absReplacesPath,
	}
}

func (d *FileHelper) TemplatePath(fileType string) string {
	var templateName string
	switch fileType {
	case "mod":
		templateName = "replaces-mod.tpl"
	default:
		log.Fatalf("Unknown file_type %q (expected 'mod')", fileType)
	}
	return filepath.Join(d.ScriptDir, templateName)
}

func (d *FileHelper) ModuleOutputPath(outputFile string) string {
	return filepath.Join(d.ScriptDir, outputFile)
}

func (d *FileHelper) ModuleTargetPath(modulePath string) string {
	return filepath.Join(d.ProjectRoot, modulePath)
}

func (d *FileHelper) ModuleDir(modulePath string) string {
	moduleDir := filepath.Join(d.ProjectRoot, filepath.Dir(modulePath))
	abs, err := filepath.Abs(moduleDir)
	if err != nil {
		log.Fatalf("Failed to resolve module directory %s: %v", moduleDir, err)
	}
	return abs
}

func (d *FileHelper) LoadProjectReplaces() (*types.ProjectReplaces, error) {
	data, err := os.ReadFile(d.ProjectReplacesPath)
	if err != nil {
		return nil, fmt.Errorf("read dependency-replacements.yaml: %w", err)
	}

	var projectReplaces types.ProjectReplaces
	if err := yaml.Unmarshal(data, &projectReplaces); err != nil {
		return nil, fmt.Errorf("parse dependency-replacements.yaml: %w", err)
	}

	return &projectReplaces, nil
}
