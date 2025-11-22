package types

// ReplaceEntry represents a single replace directive for a Go module dependency.
type ReplaceEntry struct {
	// Comment is an optional comment describing why this replace is needed.
	Comment string `yaml:"comment"`

	// Dependency is the module path being replaced (e.g., "github.com/example/package").
	Dependency string `yaml:"dependency"`

	// Replacement is the replacement path or version (e.g., "./local/path" or "github.com/fork/package v1.0.0").
	Replacement string `yaml:"replacement"`
}

// Module represents a Go module that needs replace directives applied.
type Module struct {
	// Name is the identifier for this module (e.g., "alloy", "collector").
	Name string `yaml:"name"`

	// Path is the path to the module file relative to the project root (e.g., "go.mod", "collector/go.mod").
	Path string `yaml:"path"`

	// FileType is the type of file being processed (e.g., "mod" for go.mod files).
	FileType string `yaml:"file_type"`

	// OutputFile is the name of the generated replace file (e.g., "alloy-mod-replaces.txt").
	OutputFile string `yaml:"output_file"`
}

// ProjectReplaces is the root structure of the dependency-replacements.yaml file.
type ProjectReplaces struct {
	// Modules is the list of modules that need replace directives applied.
	Modules []Module `yaml:"modules"`

	// Replaces is the list of replace entries that can be applied to modules.
	Replaces []ReplaceEntry `yaml:"replaces"`
}
