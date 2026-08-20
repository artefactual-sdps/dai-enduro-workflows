package activities

import (
	"context"
	"errors"

	"go.artefactual.dev/tools/temporal"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/size"
)

const CheckSIPInfoName = "check-sip-info"

type CheckSIPInfoParams struct {
	Path string
}

type CheckSIPInfoResult struct {
	size.Info
	SizeHuman string
}

type CheckSIPInfo struct{}

func NewCheckSIPInfo() *CheckSIPInfo {
	return &CheckSIPInfo{}
}

func (a *CheckSIPInfo) Execute(ctx context.Context, params *CheckSIPInfoParams) (*CheckSIPInfoResult, error) {
	if params.Path == "" {
		return nil, temporal.NewNonRetryableError(errors.New("path cannot be empty"))
	}

	dirInfo, err := size.DirInfo(params.Path)
	if err != nil {
		return nil, err
	}

	return &CheckSIPInfoResult{
		Info:      dirInfo,
		SizeHuman: size.FormateBytes(dirInfo.SizeInBytes),
	}, nil
}
