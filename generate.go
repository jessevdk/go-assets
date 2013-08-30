package assets

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// An asset generator. The generator can be used to generate an asset go file
// with all the assets that were added to the generator embedded into it.
// The generated assets are made available by the specified go variable
// VariableName which is of type assets.FileSystem.
type Generator struct {
	// The package name to generate assets in,
	PackageName string

	// The variable name containing the asset filesystem (defaults to Assets),
	VariableName string

	// Whether the assets will be compressed using gzip (defaults to false),
	Compressed bool

	// Strip the specified prefix from all paths,
	StripPrefix string

	fsDirsMap  map[string][]string
	fsFilesMap map[string]os.FileInfo
}

func (x *Generator) addPath(parent string, info os.FileInfo) error {
	p := path.Join(parent, info.Name())

	if x.fsFilesMap == nil {
		x.fsFilesMap = make(map[string]os.FileInfo)
	}

	if x.fsDirsMap == nil {
		x.fsDirsMap = make(map[string][]string)
	}

	x.fsFilesMap[p] = info

	if info.IsDir() {
		f, err := os.Open(p)
		fi, err := f.Readdir(-1)

		if err != nil {
			return err
		}

		x.fsDirsMap[p] = make([]string, 0, len(fi))

		for _, f := range fi {
			if err := x.addPath(p, f); err != nil {
				return err
			}
		}
	} else {
		x.fsDirsMap[parent] = append(x.fsDirsMap[parent], info.Name())
	}

	return nil
}

// Add a file or directory asset to the generator. Added directories will be
// recursed automatically.
func (x *Generator) Add(p string) error {
	p = path.Clean(p)

	info, err := os.Stat(p)

	if err != nil {
		return err
	}

	return x.addPath(path.Dir(p), info)
}

// Write the asset tree specified in the generator to the given writer. The
// written asset tree is a valid, standalone go file with the assets
// embedded into it.
func (x *Generator) Write(wr io.Writer) error {
	p := x.PackageName

	if len(p) == 0 {
		p = "main"
	}

	variableName := x.VariableName

	if len(variableName) == 0 {
		variableName = "Assets"
	}

	writer := &bytes.Buffer{}

	// Write package and import
	fmt.Fprintf(writer, "package %s\n\n", p)
	fmt.Fprintln(writer, "import (")
	fmt.Fprintln(writer, "\t\"github.com/jessevdk/go-assets\"")
	fmt.Fprintln(writer, "\t\"time\"")
	fmt.Fprintln(writer, ")")
	fmt.Fprintln(writer)

	vnames := make(map[string]string)

	// Write file contents as const strings
	if x.fsFilesMap != nil {
		// Create mapping from full file path to asset variable name.
		// This also reads the file and writes the contents as a const
		// string
		for k, v := range x.fsFilesMap {
			if v.IsDir() {
				continue
			}

			f, err := os.Open(k)

			if err != nil {
				return err
			}

			defer f.Close()

			var data []byte

			if x.Compressed {
				buf := &bytes.Buffer{}
				gw := gzip.NewWriter(buf)

				if _, err := io.Copy(gw, f); err != nil {
					gw.Close()
					return err
				}

				gw.Close()
				data = buf.Bytes()
			} else {
				data, err = ioutil.ReadAll(f)

				if err != nil {
					return err
				}
			}

			s := sha1.New()
			io.WriteString(s, k)

			vname := fmt.Sprintf("__%s%x", variableName, s.Sum(nil))
			vnames[k] = vname

			fmt.Fprintf(writer, "var %s = []byte(%#v)\n", vname, string(data))
		}

		fmt.Fprintln(writer)
	}

	fmt.Fprintf(writer, "var %s assets.FileSystem\n\n", variableName)

	fmt.Fprintln(writer, "func init() {")
	fmt.Fprintf(writer, "\t%s = assets.FileSystem{\n", variableName)

	if x.fsDirsMap == nil {
		x.fsDirsMap = make(map[string][]string)
	}

	if x.fsFilesMap == nil {
		x.fsFilesMap = make(map[string]os.FileInfo)
	}

	dirmap := make(map[string][]string)

	for k, v := range x.fsDirsMap {
		vv := make([]string, len(v))

		for i, vi := range v {
			vv[i] = strings.TrimPrefix(vi, x.StripPrefix)

			if len(vv[i]) == 0 {
				vv[i] = "/"
			}
		}

		kk := strings.TrimPrefix(k, x.StripPrefix)

		if len(kk) == 0 {
			kk = "/"
		}

		dirmap[kk] = vv
	}

	fmt.Fprintf(writer, "\t\tDirs: %#v,\n", dirmap)
	fmt.Fprintln(writer, "\t\tFiles: map[string]*assets.File{")

	// Write files
	for k, v := range x.fsFilesMap {
		kk := strings.TrimPrefix(k, x.StripPrefix)

		if len(kk) == 0 {
			kk = "/"
		}

		fmt.Fprintf(writer, "\t\t\t%#v: &assets.File{\n", kk)
		fmt.Fprintf(writer, "\t\t\t\tPath:     %#v,\n", kk)
		fmt.Fprintf(writer, "\t\t\t\tFileMode: %#v,\n", v.Mode())

		mt := v.ModTime()

		fmt.Fprintf(writer, "\t\t\t\tMTime:    time.Unix(%#v, %#v),\n", mt.Unix(), mt.UnixNano())

		if !v.IsDir() {
			fmt.Fprintf(writer, "\t\t\t\tData:     %s,\n", vnames[k])
		}

		fmt.Fprintln(writer, "\t\t\t},")
	}

	fmt.Fprintln(writer, "\t\t},")
	fmt.Fprintf(writer, "\t\tCompressed: %#v,\n", x.Compressed)
	fmt.Fprintf(writer, "\t}\n")
	fmt.Fprintln(writer, "}")

	ret, err := format.Source(writer.Bytes())

	if err != nil {
		return err
	}

	wr.Write(ret)
	return nil
}
