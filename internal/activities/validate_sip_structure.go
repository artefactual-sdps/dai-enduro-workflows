package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unicode/utf8"

	"go.artefactual.dev/tools/temporal"
)

const ValidateSIPStructureName = "validate-sip-structure"

type ValidateSIPStructureParams struct {
	Path string
}

type ValidateSIPStructureResult struct {
	ValidationErrors []string
}

type ValidateSIPStructure struct{}

func NewValidateSIPStructure() *ValidateSIPStructure {
	return &ValidateSIPStructure{}
}

func (a *ValidateSIPStructure) Execute(
	ctx context.Context,
	params *ValidateSIPStructureParams,
) (*ValidateSIPStructureResult, error) {
	if params == nil || params.Path == "" {
		return nil, temporal.NewNonRetryableError(errors.New("path cannot be empty"))
	}

	logger := temporal.GetLogger(ctx)
	result := &ValidateSIPStructureResult{}

	// Using root prevents: G122 — TOCTOU / symlink race (CWE-367).
	root, err := os.OpenRoot(params.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	fsys := root.FS()

	hasReadme := false
	hasMetadataDirectory := false
	dirs := []string{}
	nonEmptyDirs := map[string]struct{}{}
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		nonEmptyDirs[filepath.Dir(path)] = struct{}{}

		if d.IsDir() {
			dirs = append(dirs, path)

			// If the path equals to "metadata" it means it's located at the root of the Folder.
			if path == "metadata" {
				hasMetadataDirectory = true
			}
		} else {
			raw, err := root.ReadFile(path)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Failed to validate UTF-8 for %q", path))
				return err
			}
			// All files MUST be Unicode Transformation Format - 8 bits (UTF-8) encoded.
			if !utf8.Valid(raw) {
				msg := fmt.Sprintf("Files MUST be UTF-8 encoded, %q is not", path)
				result.ValidationErrors = append(result.ValidationErrors, msg)
			}

			if path == "metadata/README.md" {
				hasReadme = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if !hasMetadataDirectory {
		result.ValidationErrors = append(result.ValidationErrors, "SIP Must include a top-level metadata directory")
	} else if !hasReadme {
		result.ValidationErrors = append(result.ValidationErrors, "Metadata directory must include a README.md file")
	}
	for _, dir := range dirs {
		if dir == "." {
			continue // SIP root is not a "folder in the SIP"
		}
		if _, found := nonEmptyDirs[dir]; !found {
			result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("folder %q is empty", dir))
		}
	}

	return result, nil
}
