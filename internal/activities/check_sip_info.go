package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"go.artefactual.dev/tools/temporal"
)

const (
	CheckSIPInfoName = "check-sip-info"
)

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
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		// Ignore root.
		// if relativePath == "." {
		// 	return nil
		// }

		if utf8.RuneCountInString(relativePath) > MAX_FILE_PATH_LENGTH {
			msg := fmt.Sprintf("%q has more than %d characters", relativePath, MAX_FILE_PATH_LENGTH)
			result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
		}
		if strings.Count(relativePath, string(filepath.Separator)) > MAX_NESTED_FOLDERS {
			msg := fmt.Sprintf("%q exceeds the allowed nested folder limit of %d", relativePath, MAX_NESTED_FOLDERS)
			result.FileAndFolderValidationErrors = append(result.FileAndFolderValidationErrors, msg)
		}

		if d.IsDir() {
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
