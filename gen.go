package natsrpc

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"go/format"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

func stringPtr(in string) *string {
	if in == "" {
		return nil
	}

	return &in
}

func outName(in *descriptor.FileDescriptorProto) string {
	if in.Name != nil {
		name := *in.Name
		ext := filepath.Ext(name)
		name = name[0 : len(name)-len(ext)]
		return name + ".pb.nats.go"
	}

	return "service.pb.nats.go"
}

// packageName determines the name of the package.
func packageName(in *descriptor.FileDescriptorProto) (string, error) {
	if in.Package != nil {
		return *in.Package, nil
	}

	if in.Name != nil {
		name := *in.Name
		ext := filepath.Ext(name)
		return name[0 : len(name)-len(ext)], nil
	}

	return "", errors.New("unable to determine package name")
}

func packagePath(in *descriptor.FileDescriptorProto) (string, error) {
	goPath := filepath.Join(build.Default.GOPATH, "src")

	fpath, err := filepath.Abs(*in.Name)
	if err != nil {
		return "", err
	}

	return filepath.Rel(goPath, filepath.Dir(fpath))
}

// base and lower are template helper functions.
func newTemplate(content string) (*template.Template, error) {
	fn := map[string]interface{}{
		"base": func(in string) string {
			idx := strings.LastIndex(in, ".")
			if idx == -1 {
				return in
			}
			return in[idx+1:]
		},
		"lower": strings.ToLower,
	}

	return template.New("page").Funcs(fn).Parse(content)
}

type service struct {
	Pkg     string
	PkgPath string
	Name    string
	Methods []*method
}

type method struct {
	Name       string
	Topic      string
	InputType  string
	OutputType string
}

func ParseFile(in *descriptor.FileDescriptorProto, out, tmpl string) (*plugin_go.CodeGeneratorResponse_File, error) {
	if len(in.Service) != 1 {
		return nil, errors.New("exactly one sevice must be defined")
	}

	if out == "" {
		out = outName(in)
	}

	pkg, err := packageName(in)
	if err != nil {
		return nil, err
	}

	pkgPath, err := packagePath(in)
	if err != nil {
		return nil, err
	}

	sp := in.Service[0]

	sd := &service{
		Pkg:     pkg,
		PkgPath: pkgPath,
		Name:    sp.GetName(),
	}

	for _, m := range sp.Method {
		sd.Methods = append(sd.Methods, &method{
			Name:       m.GetName(),
			Topic:      fmt.Sprintf("%s.%s", pkg, m.GetName()), // TODO: support alternate prefix
			InputType:  m.GetInputType(),
			OutputType: m.GetOutputType(),
		})
	}

	buf := bytes.NewBuffer(nil)
	t, err := newTemplate(tmpl)
	if err != nil {
		return nil, err
	}

	t.Execute(buf, sd)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &plugin_go.CodeGeneratorResponse_File{
		Name:    stringPtr(out),
		Content: stringPtr(string(src)),
	}, nil
}
