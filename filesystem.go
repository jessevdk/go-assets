package assets

import (
	"bytes"
	"net/http"
	"os"
)

// An in-memory asset file system. The file system implements the
// http.FileSystem interface.
type FileSystem struct {
	// A map of directory paths to the files in those directories.
	Dirs map[string][]string

	// A map of file/directory paths to assets.File types.
	Files map[string]*File

	// Whether or not the file data in the file system is stored in gzip
	// compressed form.
	Compressed bool
}

func (f *FileSystem) NewFile(path string, filemode os.FileMode, mtime time.Time, data []byte) *File {
	return &File{
		Path:     path,
		FileMode: filemode,
		Mtime:    mtime,
		Data:     data,

		fs: f,
	}
}

// Implementation of http.FileSystem
func (f *FileSystem) Open(path string) (http.File, error) {
	if fi, ok := f.Files[path]; ok {
		if !fi.IsDir() {
			// Make a copy for reading
			ret := fi
			ret.buf = bytes.NewReader(ret.Data)

			return ret, nil
		}

		return fi, nil
	}

	return nil, os.ErrNotExist
}

func (f *FileSystem) readDir(p string, index int, count int) ([]os.FileInfo, error) {
	if d, ok := f.Dirs[p]; ok {
		maxl := index + count

		if maxl > len(d) {
			maxl = len(d)
		}

		ret := make([]os.FileInfo, 0, maxl-index)

		for i := index; i < maxl; i++ {
			ret = append(ret, f.Files[path.Join(p, d[i])])
		}

		return ret, nil
	}

	return nil, os.ErrNotExist
}
