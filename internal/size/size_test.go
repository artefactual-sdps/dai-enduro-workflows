package size_test

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/size"
)

func TestDirInfo(t *testing.T) {
	t.Parallel()

	type test struct {
		name    string
		setup   func(t *testing.T) string
		want    size.Info
		wantErr string
	}

	for _, tc := range []test{
		{
			name: "Returns zero for an empty directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return fs.NewDir(t, "empty").Path()
			},
			want: size.Info{
				SizeInBytes:         0,
				NumberOfFiles:       0,
				NumberOfDirectories: 0,
			},
		},
		{
			name: "Sums nested file sizes and counts files and directories",
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
			want: size.Info{
				SizeInBytes:         11,
				NumberOfFiles:       3,
				NumberOfDirectories: 1,
			},
		},
		{
			name: "Returns the size of a single file path",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := fs.NewDir(t, "file", fs.WithFile("only.txt", "abcd"))
				return dir.Join("only.txt")
			},
			want: size.Info{
				SizeInBytes:         4,
				NumberOfFiles:       1,
				NumberOfDirectories: 0,
			},
		},
		{
			name: "Errors when the path does not exist",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing")
			},
			wantErr: "no such file or directory",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := size.DirInfo(tc.setup(t))
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
}

func TestDirInfoPermissionDenied(t *testing.T) {
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

	_, err := size.DirInfo(locked)
	assert.ErrorContains(t, err, "permission denied")
}

func TestFormateBytes(t *testing.T) {
	t.Parallel()

	type test struct {
		name string
		in   uint64
		want string
	}

	for _, tc := range []test{
		{name: "Zero bytes", in: 0, want: "0 B"},
		{name: "Bytes", in: 1, want: "1 B"},
		{name: "Kilobytes", in: size.KyloByte, want: "1.0 kB"},
		{name: "Megabytes", in: size.Megabyte, want: "1.0 MB"},
		{name: "Gigabytes", in: size.Gygabyte, want: "1.0 GB"},
		{name: "Terabytes", in: size.Terabyte, want: "1.0 TB"},
		{name: "Fractional kilobytes", in: 1500, want: "1.5 kB"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, size.FormateBytes(tc.in), tc.want)
		})
	}
}
