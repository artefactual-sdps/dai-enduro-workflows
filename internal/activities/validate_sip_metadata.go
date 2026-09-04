package activities

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"go.artefactual.dev/tools/temporal"
)

const (
	ValidateSIPMetadataName = "validate-sip-metadata"
	sipMetadataCSV          = "metadata/metadata.csv"
)

// MetadataValidator validates a metadata.csv file against a CSV schema.
// If validationErrs nil or empty the validation is considered successful.
type MetadataValidator interface {
	Validate(ctx context.Context, csvPath, schemaPath string) (validatioErrs []string, systemErr error)
}

type ValidateSIPMetadataParams struct {
	SIPSourcePath string
	CSVSchemaPath string
}

type ValidateSIPMetadataResult struct {
	ValidationErrors []string
}

type ValidateSIPMetadata struct {
	csvs MetadataValidator
}

func NewValidateSIPMetadata(validator MetadataValidator) *ValidateSIPMetadata {
	return &ValidateSIPMetadata{csvs: validator}
}

func (a *ValidateSIPMetadata) Execute(
	ctx context.Context,
	params *ValidateSIPMetadataParams,
) (*ValidateSIPMetadataResult, error) {
	if params == nil || params.SIPSourcePath == "" {
		return nil, temporal.NewNonRetryableError(errors.New("SIP source path cannot be empty"))
	}
	if params.CSVSchemaPath == "" {
		return nil, temporal.NewNonRetryableError(errors.New("CSV schema path cannot be empty"))
	}

	result := &ValidateSIPMetadataResult{}
	csvMetadataPath := filepath.Join(params.SIPSourcePath, sipMetadataCSV)
	if _, err := os.Stat(csvMetadataPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.ValidationErrors = append(result.ValidationErrors, "metadata/metadata.csv is missing")
			return result, nil
		}
		return nil, err
	}

	validationErrors, err := a.csvs.Validate(ctx, csvMetadataPath, params.CSVSchemaPath)
	if err != nil {
		return nil, err
	}
	result.ValidationErrors = validationErrors
	return result, nil
}
