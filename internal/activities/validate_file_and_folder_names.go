package activities

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const ValidateFileAndFolderName = "validate-file-and-folder-names"

const (
	MAX_FILE_PATH_LENGTH = 230
	MAX_NESTED_FOLDERS   = 5
)

var FolderNameAllowedCharacters = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type ValidateFileAndFolderParams struct {
	Path string
}

type ValidateFileAndFolderResult struct {
	ValidationErrors []string
}

type ValidateFileAndFolder struct{}

func NewValidateFileAndFolder() *ValidateFileAndFolder {
	return &ValidateFileAndFolder{}
}

func (a *ValidateFileAndFolder) Execute(
	ctx context.Context,
	params *ValidateFileAndFolderParams,
) (*ValidateFileAndFolderResult, error) {
	result := &ValidateFileAndFolderResult{}
	uniqueNameMap := map[string]struct{}{}
	failures := []string{}

	err := filepath.WalkDir(params.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(params.Path, path)
		if err != nil {
			return err
		}
		// Ignore root
		if relativePath == "." {
			return nil
		}

		if utf8.RuneCountInString(relativePath) > MAX_FILE_PATH_LENGTH {
			msg := fmt.Sprintf("%s has more than: %d characters", relativePath, MAX_FILE_PATH_LENGTH)
			failures = append(failures, msg)
		}
		if strings.Count(relativePath, "/") > MAX_NESTED_FOLDERS {
			msg := fmt.Sprintf("%s exceeds the allowed nested folder limit of '%d'", relativePath, MAX_NESTED_FOLDERS)
			failures = append(failures, msg)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		base := filepath.Base(path)
		if info.IsDir() {
			if !FolderNameAllowedCharacters.MatchString(base) {
				msg := fmt.Sprintf(
					"%s has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)",
					base,
				)
				failures = append(failures, msg)
			}

			// Folder names must be unique.
			if _, found := uniqueNameMap[base]; found {
				msg := fmt.Sprintf("%s has a duplicate name: %s in SIP", relativePath, base)
				failures = append(failures, msg)
			} else {
				uniqueNameMap[base] = struct{}{}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result.ValidationErrors = failures
	return result, nil
}
