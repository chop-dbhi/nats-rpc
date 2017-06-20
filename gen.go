package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

func defaultImporter() types.Importer {
	return importer.Default()
}

// prefixDirectory places the directory name on the beginning of each name in the list.
func prefixDirectory(directory string, names []string) []string {
	if directory == "." {
		return names
	}

	ret := make([]string, len(names))
	for i, name := range names {
		ret[i] = filepath.Join(directory, name)
	}

	return ret
}

// File holds a single parsed file and associated data.
type File struct {
	pkg *Package
	// Parsed AST.
	file *ast.File
}

type Package struct {
	dir   string
	name  string
	files []*File
	// objects defined in the AST.
	defs     map[*ast.Ident]types.Object
	typesPkg *types.Package
}

// check type-checks the package. The package must be OK to proceed.
func (p *Package) check(fs *token.FileSet, astFiles []*ast.File) error {
	p.defs = make(map[*ast.Ident]types.Object)

	config := types.Config{Importer: defaultImporter(), FakeImportC: true}
	info := &types.Info{
		Defs: p.defs,
	}

	typesPkg, err := config.Check(p.dir, fs, astFiles, info)
	if err != nil {
		return err
	}

	p.typesPkg = typesPkg
	return nil
}

// ParsePackageDir parses the package residing in the directory.
func ParsePackageDir(d string) (*Package, error) {
	pkg, err := build.Default.ImportDir(d, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot process directory %s: %s", d, err)
	}

	var names []string

	names = append(names, pkg.GoFiles...)
	names = prefixDirectory(d, names)

	return parsePackage(d, names, nil)
}

// parsePackage analyzes the single package constructed from the named files.
// If text is non-nil, it is a string to be used instead of the content of the file,
// to be used for testing. parsePackage exits if there is an error.
func parsePackage(directory string, names []string, text interface{}) (*Package, error) {
	var (
		pkg      Package
		astFiles []*ast.File
	)

	fs := token.NewFileSet()

	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		parsedFile, err := parser.ParseFile(fs, name, text, 0)
		if err != nil {
			return nil, err
		}

		astFiles = append(astFiles, parsedFile)
		pkg.files = append(pkg.files, &File{
			file: parsedFile,
			pkg:  &pkg,
		})
	}

	if len(astFiles) == 0 {
		return nil, fmt.Errorf("%s: no buildable Go files", directory)
	}

	pkg.name = astFiles[0].Name.Name
	pkg.dir = directory

	// Type check the package.
	err := pkg.check(fs, astFiles)
	if err != nil {
		return nil, err
	}

	return &pkg, nil
}
