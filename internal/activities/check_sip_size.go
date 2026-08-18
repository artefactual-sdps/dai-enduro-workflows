package activities

import (
	"context"
	"errors"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/size"
)

const CheckSIPSizeName = "check-sip-size"

type CheckSIPSizeParams struct {
	Path string
}

type CheckSIPSizeResult struct {
	SizeInBytes uint64
	SizeHuman   string
}

type CheckSIPSize struct{}

func NewCheckSIPSize() *CheckSIPSize {
	return &CheckSIPSize{}
}

func (a *CheckSIPSize) Execute(ctx context.Context, params *CheckSIPSizeParams) (*CheckSIPSizeResult, error) {
	if params.Path == "" {
		return nil, errors.New("path cannot be empty")
	}

	res, err := size.GetDirSize(params.Path)
	if err != nil {
		return nil, err
	}

	return &CheckSIPSizeResult{
		SizeInBytes: res,
		SizeHuman:   size.FormateBytes(res),
	}, nil
}
