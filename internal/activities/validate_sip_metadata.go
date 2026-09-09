package activities

import (
	"context"
	"errors"
	"os"

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
	MetadataPath string
	SchemaPath   string
}

type ValidateSIPMetadataResult struct {
	ValidationErrors []string
}

type ValidateSIPMetadata struct {
	validator MetadataValidator
}

func NewValidateSIPMetadata(validator MetadataValidator) *ValidateSIPMetadata {
	return &ValidateSIPMetadata{validator: validator}
}

func (a *ValidateSIPMetadata) Execute(
	ctx context.Context,
	params *ValidateSIPMetadataParams,
) (*ValidateSIPMetadataResult, error) {
	if params == nil || params.MetadataPath == "" {
		return nil, temporal.NewNonRetryableError(errors.New("CSV metadata path cannot be empty"))
	}
	if params.SchemaPath == "" {
		return nil, temporal.NewNonRetryableError(errors.New("CSV schema path cannot be empty"))
	}

	result := &ValidateSIPMetadataResult{}
	var err error
	if err := errors.Join(err,
		fileExists(params.MetadataPath),
		fileExists(params.SchemaPath),
	); err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	validationErrors, err := a.validator.Validate(ctx, params.MetadataPath, params.SchemaPath)
	if err != nil {
		return nil, err
	}
	result.ValidationErrors = validationErrors
	return result, nil
}

func fileExists(path string) error {
	_, err := os.Stat(path)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		err = errors.New(path + " is missing")
	}
	return err
}
