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
	MaxFilePathLength    = 230
	maxNestedDirectories = 5
)

var directoryNameAllowedChars = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type CheckSIPInfoParams struct {
	Path string
}

type CheckSIPInfoResult struct {
	SizeInBytes         uint64
	NumberOfFiles       uint
	NumberOfDirectories uint
	SizeHuman           string
	ValidationErrors    []string
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
	uniqueNames := map[string]struct{}{}
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ignore the root for the directory count and validation.
		if p == path && d.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}

		if utf8.RuneCountInString(relativePath) > MaxFilePathLength {
			msg := fmt.Sprintf(
				"the path %q exceeds the maximum of %d allowed characters",
				relativePath,
				MaxFilePathLength,
			)
			result.ValidationErrors = append(result.ValidationErrors, msg)
		}

		if strings.Count(relativePath, string(filepath.Separator)) > maxNestedDirectories {
			msg := fmt.Sprintf("%q exceeds the allowed nested folder limit of %d", relativePath, maxNestedDirectories)
			result.ValidationErrors = append(result.ValidationErrors, msg)
		}

		base := filepath.Base(p)
		if d.IsDir() {
			if !directoryNameAllowedChars.MatchString(base) {
				msg := fmt.Sprintf(
					"%q has disallowed characters, allowed: letters (a-z, A-Z), numbers (0-9), dash (-), and underscore (_)",
					base,
				)
				result.ValidationErrors = append(result.ValidationErrors, msg)
			}

			// Folder names must be unique.
			if _, found := uniqueNames[base]; found {
				msg := fmt.Sprintf("folder %q has a duplicate name %q in the SIP", relativePath, base)
				result.ValidationErrors = append(result.ValidationErrors, msg)
			} else {
				uniqueNames[base] = struct{}{}
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
