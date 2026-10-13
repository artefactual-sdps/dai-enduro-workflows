package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"go.artefactual.dev/tools/temporal"
)

const (
	CheckSIPInfoName     = "check-sip-info"
	MAX_FILE_PATH_LENGTH = 230
	MAX_NESTED_FOLDERS   = 5
)

var FolderNameAllowedCharacters = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type CheckSIPInfoParams struct {
	Path string
}

type CheckSIPInfoResult struct {
	SizeInBytes                   uint64
	NumberOfFiles                 uint
	NumberOfDirectories           uint
	SizeHuman                     string
	FileAndFolderValidationErrors []string
}

type CheckSIPInfo struct{}

func NewCheckSIPInfo() *CheckSIPInfo {
	return &CheckSIPInfo{}
}

func (a *CheckSIPInfo) Execute(ctx context.Context, params *CheckSIPInfoParams) (*CheckSIPInfoResult, error) {
	if params.Path == "" {
		return nil, temporal.NewNonRetryableError(errors.New("path cannot be empty"))
	}

	result, err := collectSIPInfo(params.Path)
	if err != nil {
		return nil, err
	}

	result.SizeHuman = humanize.Bytes(result.SizeInBytes)
	return result, nil
}

func collectSIPInfo(path string) (*CheckSIPInfoResult, error) {
	var result CheckSIPInfoResult
	uniqueNameMap := map[string]struct{}{}
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ignore the root for the directory count.
		if p == path && d.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}

		if utf8.RuneCountInString(relativePath) > MAX_FILE_PATH_LENGTH {
			msg := fmt.Sprintf("%q has more than %d characters", relativePath, MAX_FILE_PATH_LENGTH)
			result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
		}

		if strings.Count(relativePath, string(filepath.Separator)) > MAX_NESTED_FOLDERS {
			msg := fmt.Sprintf("%q exceeds the allowed nested folder limit of %d", relativePath, MAX_NESTED_FOLDERS)
			result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
		}

		base := filepath.Base(p)
		if d.IsDir() {
			if !FolderNameAllowedCharacters.MatchString(base) {
				msg := fmt.Sprintf(
					"%q has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)",
					base,
				)
				result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
			}

			// Folder names must be unique.
			if _, found := uniqueNameMap[base]; found {
				msg := fmt.Sprintf("folder %q has a duplicate name %q in the SIP", relativePath, base)
				result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
			} else {
				uniqueNameMap[base] = struct{}{}
			}

			// Ignore the root for the directory count.
			if p == path {
				return nil
			}

			result.NumberOfDirectories++
		} else {
			info, err := d.Info()
			if err != nil {
				return err
			}

			s := info.Size()
			if s < 0 {
				return fmt.Errorf("negative file size for %s: %d", p, s)
			}
			result.SizeInBytes += uint64(s)
			result.NumberOfFiles++
		}

		return nil
	})

	return &result, err
}
