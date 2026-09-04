package csvs_test

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/csvs"
)

// TestCSVValidatorCmd is a bit silly in the sense that we are not running the actual CLI
// validator anywhere. But these tesst are meant to document and showcase the output that
// the CSV validator will provided given certain inputs, and how our code is meant to parse them.
// The actual validator lives here: https://github.com/digital-preservation/csv-validator
func TestCSVValidatorCmd(t *testing.T) {
	t.Parallel()

	schemaPath := fs.NewFile(t, "schema.csvs", fs.WithContent("version 1.0\nfilename: notEmpty\n")).Path()
	csvPath := fs.NewFile(
		t,
		"metadata.csv",
		fs.WithContent("filename,identifier,identifier.ianus\na.pdf,id,ianus\n"),
	).Path()

	tests := map[string]struct {
		setup   func(t *testing.T) *csvs.CSVValidatorCmd
		want    []string
		wantErr string
	}{
		"Accepts a CSV that the validator passes": {
			setup: func(t *testing.T) *csvs.CSVValidatorCmd {
				t.Helper()
				script := `#!/bin/sh
echo 'processing 1 of 2'
echo 'processing 2 of 2'
echo PASS
exit 0
`
				return &csvs.CSVValidatorCmd{Command: fakeCSVValidator(t, script)}
			},
		},
		"Collects validator error lines": {
			setup: func(t *testing.T) *csvs.CSVValidatorCmd {
				t.Helper()
				script := `#!/bin/sh
echo 'processing 1 of 2'
echo 'Error:   notEmpty fails for line: 1, column: filename, value: ""'
echo 'processing 2 of 2'
echo 'Error:   notEmpty fails for line: 1, column: identifier, value: ""'
echo 'Error:   notEmpty fails for line: 1, column: identifier.ianus, value: ""'
echo FAIL
exit 1
`
				return &csvs.CSVValidatorCmd{Command: fakeCSVValidator(t, script)}
			},
			want: []string{
				`notEmpty fails for line: 1, column: filename, value: ""`,
				`notEmpty fails for line: 1, column: identifier, value: ""`,
				`notEmpty fails for line: 1, column: identifier.ianus, value: ""`,
			},
		},
		"Collects missing column header errors": {
			setup: func(t *testing.T) *csvs.CSVValidatorCmd {
				t.Helper()
				script := `#!/bin/sh
echo 'processing 1 of 4'
echo 'Error:   Metadata header, cannot find the column headers - identifier.identifiertype, title, publicationyear - .'
echo FAIL
exit 1
`
				return &csvs.CSVValidatorCmd{Command: fakeCSVValidator(t, script)}
			},
			want: []string{
				"Metadata header, cannot find the column headers - identifier.identifiertype, title, publicationyear - .",
			},
		},
		"Errors when csv-validator-cmd is not found": {
			setup: func(t *testing.T) *csvs.CSVValidatorCmd {
				t.Helper()
				return &csvs.CSVValidatorCmd{Command: filepath.Join(t.TempDir(), "missing-csv-validator-cmd")}
			},
			wantErr: "not found",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.setup(t).Validate(t.Context(), csvPath, schemaPath)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}

func fakeCSVValidator(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "csv-validator-cmd")
	assert.NilError(t, os.WriteFile(path, []byte(script), 0o600))
	assert.NilError(t, os.Chmod(path, 0o700))
	return path
}
