package workflows

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/config"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/sip"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/size"
)

type PreprocessingWorkflow struct {
	cfg config.PreprocessingConfig
}

func NewPreprocessingWorkflow(cfg config.PreprocessingConfig) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{cfg: cfg}
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *childwf.PreprocessingParams,
) (*childwf.PreprocessingResult, error) {
	result := &childwf.PreprocessingResult{}
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
	}
	result.RelativePath = params.RelativePath
	sourcePath := filepath.Join(w.cfg.SharedPath, params.RelativePath)

	sipName := filepath.Base(sourcePath)
	validateSIPNameTask := result.NewTask(temporalsdk_workflow.Now(ctx), "SIP validate Name")
	if validationErrors := sip.ValidateName(sipName); len(validationErrors) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateSIPNameTask,
			fmt.Sprintf("Invalid SIP name '%s':\n%s", sipName, ul(validationErrors)),
		)
		return result, nil
	}
	validateSIPNameTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP name is valid: %s", sipName)

	task0 := result.NewTask(temporalsdk_workflow.Now(ctx), "SIP validate Size")
	var validatSIPSizeResult activities.CheckSIPSizeResult
	err := temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		activities.CheckSIPSizeName,
		&activities.CheckSIPSizeParams{Path: sourcePath},
	).Get(ctx, &validatSIPSizeResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), task0, "SIP validation has failed")
		return result, nil
	}

	if validatSIPSizeResult.SizeInBytes > size.Terabyte {
		result.ValidationError(temporalsdk_workflow.Now(ctx), task0, "SIP is bigger than 1 Terabyte")
		return result, nil
	} else {
		task0.Succeed(temporalsdk_workflow.Now(ctx), "SIP size checked: %s", validatSIPSizeResult.SizeHuman)
	}

	// Bag the SIP for Enduro processing.
	task := result.NewTask(temporalsdk_workflow.Now(ctx), "Bag SIP")
	var createBag bagcreate.Result
	err = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{
			SourcePath: sourcePath,
		},
	).Get(ctx, &createBag)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), task, "bagging has failed")
		return result, nil
	}
	task.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

	return result, nil
}

func withFilesystemActivityOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 2,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

// ul formats a list of strings as an unordered, Markdown-style list.
func ul(items []string) string {
	if len(items) == 0 {
		return ""
	}

	var s strings.Builder
	for _, i := range items {
		fmt.Fprintf(&s, "- %s\n", i)
	}

	return strings.TrimSuffix(s.String(), "\n")
}
