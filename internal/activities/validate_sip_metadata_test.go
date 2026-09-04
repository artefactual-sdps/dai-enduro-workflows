package activities_test

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
)

type testMetadataValidator struct {
	validationErrs []string
	systemErr      error
}

func (s *testMetadataValidator) Validate(_ context.Context, _, _ string) ([]string, error) {
	return s.validationErrs, s.systemErr
}

func TestValidateSIPMetadata(t *testing.T) {
	t.Parallel()

	schemaPath := fs.NewFile(t, "schema.csvs", fs.WithContent("version 1.0\nfilename: notEmpty\n")).Path()

	tests := map[string]struct {
		setup              func(t *testing.T) string
		schema             string
		validator          *testMetadataValidator
		wantValidationErrs []string
		wantErr            string
	}{
		"Errors when the SIP source path is empty": {
			setup: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			schema:  schemaPath,
			wantErr: "SIP source path cannot be empty",
		},
		"Errors when the CSV schema path is empty": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip").Path()
			},
			wantErr: "CSV schema path cannot be empty",
		},
		"Reports a missing metadata.csv as a validation error": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("README.md", "# SIP\n"),
					),
				).Path()
			},
			schema:             schemaPath,
			wantValidationErrs: []string{"metadata/metadata.csv is missing"},
		},
		"Accepts a CSV that the validator accepts": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("metadata.csv", "filename,identifier,identifier.ianus\na.pdf,id,ianus\n"),
					),
				).Path()
			},
			schema:    schemaPath,
			validator: &testMetadataValidator{},
		},
		"Returns validator system errors": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("metadata.csv", "filename\na.pdf\n"),
					),
				).Path()
			},
			schema:    schemaPath,
			validator: &testMetadataValidator{systemErr: errors.New("csv-validator-cmd not found")},
			wantErr:   "csv-validator-cmd not found",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sipPath := tc.setup(t)
			got, err := activities.NewValidateSIPMetadata(tc.validator).Execute(
				t.Context(),
				&activities.ValidateSIPMetadataParams{
					SIPSourcePath: sipPath,
					CSVSchemaPath: tc.schema,
				},
			)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got.ValidationErrors, tc.wantValidationErrs)
		})
	}
}
