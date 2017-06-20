package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"go/format"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

var (
	buildVersion string
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("nats-rpc: ")
}

func main() {
	var (
		typeName      string
		fileName      string
		cliFileName   string
		serviceGroup  string
		subjectPrefix string
		printVersion  bool
	)

	flag.StringVar(&typeName, "type", "", "Type name.")
	flag.StringVar(&fileName, "client", "", "Output file name client interface.")
	flag.StringVar(&cliFileName, "cli", "", "Output file name for CLI.")
	flag.StringVar(&serviceGroup, "group", "", "Name of the NATS queue group.")
	flag.StringVar(&subjectPrefix, "prefix", "", "Prefix to all subjects.")
	flag.BoolVar(&printVersion, "version", false, "Print version.")

	flag.Parse()

	if printVersion {
		fmt.Println(buildVersion)
		return
	}

	if typeName == "" {
		log.Fatal("type required")
	}

	if fileName == "" {
		log.Fatal("file name required")
	}

	if cliFileName == "" {
		log.Fatal("cli file name required")
	}

	args := flag.Args()

	// Default to current directory.
	if len(args) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("could not get cwd: %s", err)
		}

		args = []string{dir}
	}

	gopath := filepath.Join(build.Default.GOPATH, "src")
	pkgPath, err := filepath.Rel(gopath, args[0])
	if err != nil {
		log.Fatalf("could not get relative path: %s", err)
	}

	pkg, err := ParsePackageDir(args[0])
	if err != nil {
		log.Fatal(err)
	}

	var (
		ok  bool
		obj types.Object
		inf *types.Interface
	)

	for _, obj = range pkg.defs {
		if obj == nil {
			continue
		}

		// Ignore objects that don't have the target name.
		if obj.Name() != typeName {
			continue
		}

		// Looking for an interface type..
		inf, ok = obj.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}

		break
	}

	meta := reflectInterface(inf)
	meta.Name = typeName
	meta.PkgPath = pkgPath
	meta.Pkg = pkg.typesPkg.Name()

	if serviceGroup == "" {
		serviceGroup = fmt.Sprintf("%#v", fmt.Sprintf("%s.svc", meta.Pkg))
	}

	for _, m := range meta.Methods {
		m.Pkg = meta.Pkg
		m.Topic = fmt.Sprintf("%#v", fmt.Sprintf("%s%s.%s", subjectPrefix, meta.Pkg, m.Name))
		m.ServiceGroup = serviceGroup
	}

	// Compile and generate files.
	var buf bytes.Buffer

	t := template.Must(template.New("client").Parse(fileTmpl))
	if err := t.Execute(&buf, meta); err != nil {
		log.Fatal(err)
	}

	// Format the output.
	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err = ioutil.WriteFile(fileName, src, 0644); err != nil {
		log.Fatalf("writing output: %s", err)
	}

	// Reuse buffer.
	buf.Reset()

	t = template.Must(template.New("cli").Parse(cliTmpl))
	if err := t.Execute(&buf, meta); err != nil {
		log.Fatal(err)
	}

	// Format the output.
	src, err = format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err = ioutil.WriteFile(cliFileName, src, 0644); err != nil {
		log.Fatalf("writing output: %s", err)
	}
}
