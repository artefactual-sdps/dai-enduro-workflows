package activities_test

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
)

func TestCheckSIPInfo(t *testing.T) {
	t.Parallel()

	type test struct {
		setup   func(t *testing.T) string
		want    *activities.CheckSIPInfoResult
		wantErr string
	}

	tests := map[string]test{
		"Returns zero for an empty directory": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "empty").Path()
			},
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         0,
				NumberOfFiles:       0,
				NumberOfDirectories: 0,
				SizeHuman:           "0 B",
			},
		},
		"Sums nested file sizes and counts files and directories": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "sip",
					fs.WithFile("readme.txt", "hello"),
					fs.WithDir("data",
						fs.WithFile("payload.bin", "world!"),
						fs.WithFile("empty.dat", ""),
					),
				).Path()
			},
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         11,
				NumberOfFiles:       3,
				NumberOfDirectories: 1,
				SizeHuman:           "11 B",
			},
		},
		"Returns the size of a single file path": {
			setup: func(t *testing.T) string {
				t.Helper()
				dir := fs.NewDir(t, "file", fs.WithFile("only.txt", "abcd"))
				return dir.Join("only.txt")
			},
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         4,
				NumberOfFiles:       1,
				NumberOfDirectories: 0,
				SizeHuman:           "4 B",
			},
		},
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := activities.NewCheckSIPInfo().Execute(
				t.Context(),
				&activities.CheckSIPInfoParams{Path: tc.setup(t)},
			)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}

func TestCheckSIPInfoPermissionDenied(t *testing.T) {
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

	_, err := activities.NewCheckSIPInfo().Execute(
		t.Context(),
		&activities.CheckSIPInfoParams{Path: locked},
	)
	assert.ErrorContains(t, err, "permission denied")
}
