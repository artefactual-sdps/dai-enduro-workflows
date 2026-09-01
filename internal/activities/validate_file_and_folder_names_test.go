package activities_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
)

func TestValidateFileAndFolder(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup   func(t *testing.T) string
		want    []string
		wantErr string
	}{
		"Accepts a valid SIP tree": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "SIP_2025-10-20_IANUS1234_ABT",
					fs.WithFile("readme.txt", "hello"),
					fs.WithDir("data",
						fs.WithFile("payload.bin", "world"),
						fs.WithDir("nested",
							fs.WithFile("ok.txt", "ok"),
						),
					),
				).Path()
			},
		},
		"Ignores the walk on root": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "SIP with spaces",
					fs.WithDir("data",
						fs.WithFile("ok.txt", "ok"),
					),
				).Path()
			},
		},
		"Errors when a folder name has disallowed characters": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("bad folder",
						fs.WithFile("file.txt", "x"),
					),
				).Path()
			},
			want: []string{
				`"bad folder" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
			},
		},
		"Errors when a folder name contains a dot": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("data.dir",
						fs.WithFile("file.txt", "x"),
					),
				).Path()
			},
			want: []string{
				`"data.dir" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
			},
		},
		"Errors when two folders share the same name": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("data",
						fs.WithFile("a.txt", "a"),
					),
					fs.WithDir("other",
						fs.WithDir("data",
							fs.WithFile("b.txt", "b"),
						),
					),
				).Path()
			},
			want: []string{
				`folder name "other/data" has a duplicate name data in the SIP`,
			},
		},
		"Errors when a relative path exceeds the character limit": {
			setup: func(t *testing.T) string {
				t.Helper()
				longName := strings.Repeat("a", activities.MAX_FILE_PATH_LENGTH+1)
				return fs.NewDir(t, "sip",
					fs.WithFile(longName, "x"),
				).Path()
			},
			want: []string{
				fmt.Sprintf(
					"%q has more than %d characters",
					strings.Repeat("a", activities.MAX_FILE_PATH_LENGTH+1),
					activities.MAX_FILE_PATH_LENGTH,
				),
			},
		},
		"Errors when nested folders exceed the limit": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("a",
						fs.WithDir("b",
							fs.WithDir("c",
								fs.WithDir("d",
									fs.WithDir("e",
										fs.WithDir("f",
											fs.WithDir("g",
												fs.WithFile("deep.txt", "x"),
											),
										),
									),
								),
							),
						),
					),
				).Path()
			},
			want: []string{
				`"a/b/c/d/e/f/g" exceeds the allowed nested folder limit of 5`,
				`"a/b/c/d/e/f/g/deep.txt" exceeds the allowed nested folder limit of 5`,
			},
		},
		"Collects multiple validation errors": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("bad name",
						fs.WithFile("x.txt", "x"),
					),
					fs.WithDir("data",
						fs.WithFile("a.txt", "a"),
					),
					fs.WithDir("other",
						fs.WithDir("data",
							fs.WithFile("b.txt", "b"),
						),
					),
				).Path()
			},
			want: []string{
				`"bad name" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
				`folder name "other/data" has a duplicate name data in the SIP`,
			},
		},
		"Errors when the path does not exist": {
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing")
			},
			wantErr: "no such file or directory",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			act := activities.NewValidateFileAndFolder()
			got, err := act.Execute(t.Context(), &activities.ValidateFileAndFolderParams{
				Path: tc.setup(t),
			})
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, got.ValidationErrors, tc.want)
		})
	}
}
