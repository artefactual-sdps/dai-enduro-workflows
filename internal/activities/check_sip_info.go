package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

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
	SizeInBytes         uint64
	NumberOfFiles       uint
	NumberOfDirectories uint
	SizeHuman           string
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
