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

type subjectParams struct {
	Pkg     string
	Service string
	Method  string
}

type Options struct {
	Subject string
	OutFile string
}

func ParseOptions(p string) (*Options, error) {
	var opts Options

	kvs := strings.Split(p, ",")
	for _, v := range kvs {
		kv := strings.SplitN(v, "=", 2)
		switch strings.ToLower(kv[0]) {
		case "subject":
			opts.Subject = kv[1]
		case "outfile":
			opts.OutFile = kv[1]
		default:
			return nil, fmt.Errorf("unknown param: %s", kv[0])
		}
	}

	return &opts, nil
}

func ParseFile(in *descriptor.FileDescriptorProto, tmpl string, opts Options) (*plugin_go.CodeGeneratorResponse_File, error) {
	if len(in.Service) != 1 {
		return nil, errors.New("exactly one sevice must be defined")
	}

	// Parse subject template.
	if opts.Subject == "" {
		opts.Subject = "{{.Pkg}}.{{.Method}}"
	}

	subjectTmpl, err := template.New("subject").Parse(opts.Subject)
	if err != nil {
		return nil, err
	}

	if opts.OutFile == "" {
		opts.OutFile = outName(in)
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

	buf := bytes.NewBuffer(nil)
	for _, m := range sp.Method {
		buf.Reset()
		err := subjectTmpl.Execute(buf, &subjectParams{
			Pkg:     pkg,
			Service: sd.Name,
			Method:  m.GetName(),
		})
		if err != nil {
			return nil, err
		}

		sd.Methods = append(sd.Methods, &method{
			Name:       m.GetName(),
			Topic:      buf.String(),
			InputType:  m.GetInputType(),
			OutputType: m.GetOutputType(),
		})
	}

	buf.Reset()
	t, err := newTemplate(tmpl)
	if err != nil {
		return nil, err
	}

	if err := t.Execute(buf, sd); err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &plugin_go.CodeGeneratorResponse_File{
		Name:    stringPtr(opts.OutFile),
		Content: stringPtr(string(src)),
	}, nil
}
