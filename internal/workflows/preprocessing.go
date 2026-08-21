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

const (
	MAX_FILES       = 999_999
	MAX_DIRECTORIES = 999_999
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
	validateSIPNameTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP name")
	if validationErrors := sip.ValidateName(sipName); len(validationErrors) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateSIPNameTask,
			fmt.Sprintf("Invalid SIP name '%s':\n%s", sipName, ul(validationErrors)),
		)
		return result, nil
	}
	validateSIPNameTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP name is valid: %s", sipName)

	taskValidateSize := result.NewTask(temporalsdk_workflow.Now(ctx), "SIP validate Size")
	var validatSIPSizeResult activities.CheckSIPInfoResult
	err := temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		activities.CheckSIPInfoName,
		&activities.CheckSIPInfoParams{Path: sourcePath},
	).Get(ctx, &validatSIPSizeResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), taskValidateSize, "SIP validation has failed")
		return result, nil
	}

	if validatSIPSizeResult.SizeInBytes > size.Terabyte {
		result.ValidationError(temporalsdk_workflow.Now(ctx), taskValidateSize, "SIP is bigger than 1 Terabyte")
	} else {
		taskValidateSize.Succeed(temporalsdk_workflow.Now(ctx), "SIP size checked: %s", validatSIPSizeResult.SizeHuman)
	}

	{
		// Payload size validation.
		validationErrors := []string{}
		taskValidatePayloadSize := result.NewTask(temporalsdk_workflow.Now(ctx), "SIP validate payload")
		if validatSIPSizeResult.NumberOfFiles > MAX_FILES {
			msg := fmt.Sprintf("SIP payload has more than %d files", MAX_FILES)
			validationErrors = append(validationErrors, msg)
		}
		if validatSIPSizeResult.NumberOfDirectories > MAX_DIRECTORIES {
			msg := fmt.Sprintf("SIP payload has more than %d directories", MAX_DIRECTORIES)
			validationErrors = append(validationErrors, msg)
		}
		if len(validationErrors) > 0 {
			result.ValidationError(temporalsdk_workflow.Now(ctx), taskValidatePayloadSize, validationErrors...)
		} else {
			taskValidatePayloadSize.Succeed(
				temporalsdk_workflow.Now(ctx),
				"SIP payload size checked. Files: %d - Directories: %d",
				validatSIPSizeResult.NumberOfFiles,
				validatSIPSizeResult.NumberOfDirectories,
			)
		}
		if result.Outcome != childwf.OutcomeSuccess {
			return result, nil
		}
	}

	// Bag the SIP for Enduro processing.
	taskBagSip := result.NewTask(temporalsdk_workflow.Now(ctx), "Bag SIP")
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
		result.SystemError(temporalsdk_workflow.Now(ctx), taskBagSip, "bagging has failed")
		return result, nil
	}
	taskBagSip.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

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
