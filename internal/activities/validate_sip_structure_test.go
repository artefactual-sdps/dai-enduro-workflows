package activities_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
)

func TestValidateSIPStructure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup   func(t *testing.T) string
		want    func(path string) []string
		wantErr string
	}{
		"Errors when the path is empty": {
			setup: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			wantErr: "path cannot be empty",
		},
		"Errors when the path does not exist": {
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing")
			},
			wantErr: "no such file or directory",
		},
		"Accepts a valid SIP tree": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("README.md", "# SIP\n"),
					),
					fs.WithDir("payload",
						fs.WithFile("hello.txt", "hello"),
					),
				).Path()
			},
			want: func(string) []string { return nil },
		},
		"Errors when the SIP has no top-level metadata directory": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithFile("hello.txt", "hello"),
				).Path()
			},
			want: func(string) []string {
				return []string{"SIP Must include a top-level metadata directory"}
			},
		},
		"Errors when metadata is missing a README.md file": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("other.md", "# other\n"),
					),
				).Path()
			},
			want: func(string) []string {
				return []string{"Metadata directory must include a README.md file"}
			},
		},
		"Errors when metadata exists but is empty": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata"),
					fs.WithFile("hello.txt", "hello"),
				).Path()
			},
			want: func(string) []string {
				return []string{
					"Metadata directory must include a README.md file",
					`folder "metadata" is empty`,
				}
			},
		},
		"Errors when the SIP contains an empty directory": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("README.md", "# SIP\n"),
					),
					fs.WithDir("empty"),
				).Path()
			},
			want: func(string) []string {
				return []string{`folder "empty" is empty`}
			},
		},
		"Errors when a file is not UTF-8 encoded": {
			setup: func(t *testing.T) string {
				t.Helper()
				dir := fs.NewDir(t, "sip",
					fs.WithDir("metadata",
						fs.WithFile("README.md", "# SIP\n"),
					),
					fs.WithFile("bad.bin", ""),
				)
				assert.NilError(t, os.WriteFile(dir.Join("bad.bin"), []byte{0xff, 0xfe, 0x00}, 0o600))
				return dir.Path()
			},
			want: func(string) []string {
				return []string{
					fmt.Sprintf("Files MUST be UTF-8 encoded, %q is not", "bad.bin"),
				}
			},
		},
		"Collects multiple validation errors": {
			setup: func(t *testing.T) string {
				t.Helper()
				dir := fs.NewDir(t, "sip",
					fs.WithDir("not_empty", fs.WithDir("empty_dir")),
					fs.WithFile("bad.bin", ""),
				)
				assert.NilError(t, os.WriteFile(dir.Join("bad.bin"), []byte{0xff, 0xfe, 0x00}, 0o600))
				return dir.Path()
			},
			want: func(string) []string {
				return []string{
					"SIP Must include a top-level metadata directory",
					fmt.Sprintf("Files MUST be UTF-8 encoded, %q is not", "bad.bin"),
					`folder "not_empty/empty_dir" is empty`,
				}
			},
		},
		"Errors when metadata is a file rather than a directory": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithFile("metadata", "not a directory"),
				).Path()
			},
			wantErr: "not a directory",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := tc.setup(t)
			got, err := activities.NewValidateSIPStructure().Execute(
				t.Context(),
				&activities.ValidateSIPStructureParams{Path: path},
			)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, got.ValidationErrors, tc.want(path))
		})
	}
}

func TestValidateSIPStructureNilParams(t *testing.T) {
	t.Parallel()

	_, err := activities.NewValidateSIPStructure().Execute(t.Context(), nil)
	assert.ErrorContains(t, err, "path cannot be empty")
}

func TestValidateSIPStructurePermissionDenied(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root can walk unreadable directories")
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	assert.NilError(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() {
		_ = os.Chmod(locked, 0o700)
	})

	_, err := activities.NewValidateSIPStructure().Execute(
		t.Context(),
		&activities.ValidateSIPStructureParams{Path: locked},
	)
	assert.ErrorContains(t, err, "permission denied")
}
