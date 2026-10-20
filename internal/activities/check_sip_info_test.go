package activities_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		"Errors when a relative path exceeds the character limit": {
			setup: func(t *testing.T) string {
				t.Helper()
				longName := strings.Repeat("a", activities.MaxFilePathLength+1)
				return fs.NewDir(t, "sip",
					fs.WithFile(longName, "x"),
				).Path()
			},
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         1,
				NumberOfFiles:       1,
				NumberOfDirectories: 0,
				SizeHuman:           "1 B",
				ValidationErrors: []string{
					fmt.Sprintf(
						"the path %q exceeds the maximum of %d allowed characters",
						strings.Repeat("a", activities.MaxFilePathLength+1),
						activities.MaxFilePathLength,
					),
				},
			},
		},
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
		"Accepts a valid SIP tree": {
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "SIP_2025-10-20_A1B2_ABT",
					fs.WithFile("readme.txt", "hello"),
					fs.WithDir("data",
						fs.WithFile("payload.bin", "world"),
						fs.WithDir("nested",
							fs.WithFile("ok.txt", "ok"),
						),
					),
				).Path()
			},
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         12,
				NumberOfFiles:       3,
				NumberOfDirectories: 2,
				SizeHuman:           "12 B",
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         2,
				NumberOfFiles:       1,
				NumberOfDirectories: 1,
				SizeHuman:           "2 B",
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         1,
				NumberOfFiles:       1,
				NumberOfDirectories: 1,
				SizeHuman:           "1 B",
				ValidationErrors: []string{
					`"bad folder" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
				},
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         1,
				NumberOfFiles:       1,
				NumberOfDirectories: 1,
				SizeHuman:           "1 B",
				ValidationErrors: []string{
					`"data.dir" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
				},
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         2,
				NumberOfFiles:       2,
				NumberOfDirectories: 3,
				SizeHuman:           "2 B",
				ValidationErrors: []string{
					`folder "other/data" has a duplicate name "data" in the SIP`,
				},
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         1,
				NumberOfFiles:       1,
				NumberOfDirectories: 7,
				SizeHuman:           "1 B",
				ValidationErrors: []string{
					`"a/b/c/d/e/f/g" exceeds the allowed nested folder limit of 5`,
					`"a/b/c/d/e/f/g/deep.txt" exceeds the allowed nested folder limit of 5`,
				},
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
			want: &activities.CheckSIPInfoResult{
				SizeInBytes:         3,
				NumberOfFiles:       3,
				NumberOfDirectories: 4,
				SizeHuman:           "3 B",
				ValidationErrors: []string{
					`"bad name" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
					`folder "other/data" has a duplicate name "data" in the SIP`,
				},
			},
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
