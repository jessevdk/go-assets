package assets

import (
	"os"
)

func ExampleGenerator() {
	g := Generator{
		PackageName:  "main",
		VariableName: "MyAssets",
		Compressed:   true,
		StripPrefix:  ".",
	}

	if err := g.AddDir("."); err != nil {
		panic(err)
	}

	// This will write a go file to standard out. The generated go file
	// will reside in the g.PackageName package and will contain a
	// single variable g.VariableName of type assets.FileSystem containing
	// the whole file system.
	g.Write(os.Stdout)
}
