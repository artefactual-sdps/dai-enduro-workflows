package size

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/dustin/go-humanize"
)

const (
	KyloByte = 1000
	Megabyte = 1000 * KyloByte
	Gygabyte = 1000 * Megabyte
	Terabyte = 1000 * Gygabyte // 1_000_000_000_000
)

type Info struct {
	SizeInBytes         uint64
	NumberOfFiles       uint
	NumberOfDirectories uint
}

// Returns information about the path provided.
// - Size in Bytes
// - Number of files
// - Number of directories
func DirInfo(path string) (Info, error) {
	var result Info
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Ignore the root for the directory count.
			if p == path {
				return nil
			}

			result.NumberOfDirectories++
		} else {
			info, err := d.Info()
			if err != nil {
				return err
			}

			s := info.Size()
			if s < 0 {
				return fmt.Errorf("negative file size for %s: %d", p, s)
			}
			result.SizeInBytes += uint64(s)
			result.NumberOfFiles++
		}

		return nil
	})

	return result, err
}

func FormateBytes(b uint64) string {
	return humanize.Bytes(b)
}
