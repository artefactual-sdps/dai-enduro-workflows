package workflows

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/bagextract"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/config"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/sip"
)

const (
	MAX_FILES       = 999_999
	MAX_DIRECTORIES = 999_999
	MAX_BYTES       = 1_000_000_000_000 // 1 Terabyte.
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

	extractBagTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Extract the SIP bag")
	var res bagextract.Result
	err := temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		bagextract.Name,
		bagextract.Params{
			Path: sourcePath,
			Keep: []string{"metadata"},
		},
	).Get(ctx, &res)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), extractBagTask, "Failed to extract the SIP bag")
		return result, nil
	}
	extractBagTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP bag was extracted.")
	sourcePath = res.Path

	sipName := filepath.Base(sourcePath)
	validateSIPNameTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP name")
	if validationErrors := sip.ValidateName(sipName); len(validationErrors) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateSIPNameTask,
			fmt.Sprintf("Invalid SIP name '%s':\n%s", sipName, ul(validationErrors)),
		)
	} else {
		validateSIPNameTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP name is valid: %s", sipName)
	}

	taskValidateSize := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP size")
	var validatSIPSizeResult activities.CheckSIPInfoResult
	err = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		activities.CheckSIPInfoName,
		&activities.CheckSIPInfoParams{Path: sourcePath},
	).Get(ctx, &validatSIPSizeResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), taskValidateSize, "SIP validation has failed")
		return result, nil
	}

	if validatSIPSizeResult.SizeInBytes > MAX_BYTES {
		result.ValidationError(temporalsdk_workflow.Now(ctx), taskValidateSize, "SIP is bigger than 1 Terabyte")
	} else {
		taskValidateSize.Succeed(temporalsdk_workflow.Now(ctx), "SIP size checked: %s", validatSIPSizeResult.SizeHuman)
	}

	{
		// Payload size validation.
		validationErrors := []string{}
		taskValidatePayloadSize := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP payload")
		if validatSIPSizeResult.NumberOfFiles > MAX_FILES {
			msg := fmt.Sprintf("SIP payload has more than %d files", MAX_FILES)
			validationErrors = append(validationErrors, msg)
		}
		if validatSIPSizeResult.NumberOfDirectories > MAX_DIRECTORIES {
			msg := fmt.Sprintf("SIP payload has more than %d directories", MAX_DIRECTORIES)
			validationErrors = append(validationErrors, msg)
		}
		if len(validationErrors) > 0 {
			msg := strings.Join(validationErrors, " - ")
			result.ValidationError(temporalsdk_workflow.Now(ctx), taskValidatePayloadSize, msg)
		} else {
			taskValidatePayloadSize.Succeed(
				temporalsdk_workflow.Now(ctx),
				"SIP payload size checked. Files: %d - Directories: %d",
				validatSIPSizeResult.NumberOfFiles,
				validatSIPSizeResult.NumberOfDirectories,
			)
		}
		// Stop before walking the SIP when it exceeds size or payload limits.
		if validatSIPSizeResult.SizeInBytes > MAX_BYTES ||
			validatSIPSizeResult.NumberOfFiles > MAX_FILES ||
			validatSIPSizeResult.NumberOfDirectories > MAX_DIRECTORIES {
			return result, nil
		}
	}

	validateFileAndFolderNamesTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate file and folder names")
	var validateFileAndFolderResult activities.ValidateFileAndFolderResult
	err = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		activities.ValidateFileAndFolderName,
		&activities.ValidateFileAndFolderParams{Path: sourcePath},
	).Get(ctx, &validateFileAndFolderResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(
			temporalsdk_workflow.Now(ctx),
			validateFileAndFolderNamesTask,
			"file and folder name validation has failed",
		)
		return result, nil
	}
	if len(validateFileAndFolderResult.ValidationErrors) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateFileAndFolderNamesTask,
			fmt.Sprintf("Invalid file and folder names:\n%s", ul(validateFileAndFolderResult.ValidationErrors)),
		)
	} else {
		validateFileAndFolderNamesTask.Succeed(temporalsdk_workflow.Now(ctx), "File and folder names are valid")
	}

	validateSIPStructureTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP structure")
	var validateSIPStructureResult activities.ValidateSIPStructureResult
	err = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		activities.ValidateSIPStructureName,
		&activities.ValidateSIPStructureParams{Path: sourcePath},
	).Get(ctx, &validateSIPStructureResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(
			temporalsdk_workflow.Now(ctx),
			validateSIPStructureTask,
			"SIP structure validation has failed",
		)
		return result, nil
	}
	if len(validateSIPStructureResult.ValidationErrors) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateSIPStructureTask,
			fmt.Sprintf("Invalid SIP structure:\n%s", ul(validateSIPStructureResult.ValidationErrors)),
		)
	} else {
		validateSIPStructureTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP structure is valid")
	}

	validateFileFormatsTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate file formats")
	var ffvalidateResult ffvalidate.Result
	err = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		ffvalidate.Name,
		&ffvalidate.Params{Path: sourcePath},
	).Get(ctx, &ffvalidateResult)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(
			temporalsdk_workflow.Now(ctx),
			validateFileFormatsTask,
			"file format validation has failed",
		)
		return result, nil
	}
	if len(ffvalidateResult.Failures) > 0 {
		result.ValidationError(
			temporalsdk_workflow.Now(ctx),
			validateFileFormatsTask,
			fmt.Sprintf("Invalid file formats:\n%s", ul(ffvalidateResult.Failures)),
		)
	} else {
		validateFileFormatsTask.Succeed(temporalsdk_workflow.Now(ctx), "File formats are valid")
	}

	// Validate SIP Metadata
	if validateSIPStructureResult.HasMetadataDirectory {
		validateSIPMetadataTask := result.NewTask(temporalsdk_workflow.Now(ctx), "Validate the SIP metadata")
		var validateSIPMetadataResult activities.ValidateSIPMetadataResult
		err = temporalsdk_workflow.ExecuteActivity(
			withFilesystemActivityOpts(ctx),
			activities.ValidateSIPMetadataName,
			&activities.ValidateSIPMetadataParams{
				SIPSourcePath: sourcePath,
				CSVSchemaPath: w.cfg.CSVSchemaPath,
			},
		).Get(ctx, &validateSIPMetadataResult)
		if err != nil {
			logger.Error("System error", "message", err.Error())
			result.SystemError(
				temporalsdk_workflow.Now(ctx),
				validateSIPMetadataTask,
				"SIP metadata validation has failed",
			)
			return result, nil
		}
		if len(validateSIPMetadataResult.ValidationErrors) > 0 {
			result.ValidationError(
				temporalsdk_workflow.Now(ctx),
				validateSIPMetadataTask,
				fmt.Sprintf("Invalid SIP metadata:\n%s", ul(validateSIPMetadataResult.ValidationErrors)),
			)
		} else {
			validateSIPMetadataTask.Succeed(temporalsdk_workflow.Now(ctx), "The SIP metadata is valid")
		}
	}

	if result.Outcome != childwf.OutcomeSuccess {
		return result, nil
	}

	// Bag the SIP for Enduro processing.
	taskBagSIP := result.NewTask(temporalsdk_workflow.Now(ctx), "Bag SIP")
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
		result.SystemError(temporalsdk_workflow.Now(ctx), taskBagSIP, "bagging has failed")
		return result, nil
	}
	taskBagSIP.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

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
