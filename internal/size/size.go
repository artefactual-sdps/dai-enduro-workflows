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

// Returns the size of the folder in bytes.
func GetDirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}

			s := info.Size()
			if s < 0 {
				return fmt.Errorf("negative file size for %s: %d", p, s)
			}
			size += uint64(s)
		}

		return nil
	})

	return size, err
}

func FormateBytes(b uint64) string {
	return humanize.Bytes(b)
}
