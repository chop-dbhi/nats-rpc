package natsrpc

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"go/format"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	camelRegexp = regexp.MustCompile("[A-Z][^A-Z]*")

	templateFuncs = map[string]interface{}{
		"lower": strings.ToLower,
		"base": func(in string) string {
			idx := strings.LastIndex(in, ".")
			if idx == -1 {
				return in
			}
			return in[idx+1:]
		},
		"hyphenize": func(s string) string {
			return strings.ToLower(strings.Join(camelRegexp.FindAllString(s, -1), "-"))
		},
	}
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
	return template.New("page").Funcs(templateFuncs).Parse(content)
}

type service struct {
	Pkg     string
	PkgPath string
	Name    string
	Subject string
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
}

type Options struct {
	Subject string
	OutFile string
}

func ParseOptions(p string) (*Options, error) {
	if p == "" {
		return &Options{}, nil
	}

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
		opts.Subject = "{{.Pkg}}"
	}

	sp := in.Service[0]

	pkg, err := packageName(in)
	if err != nil {
		return nil, err
	}

	pkgPath, err := packagePath(in)
	if err != nil {
		return nil, err
	}

	subjectTmpl, err := template.New("subject").Parse(opts.Subject)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	err = subjectTmpl.Execute(buf, &subjectParams{
		Pkg:     pkg,
		Service: sp.GetName(),
	})
	if err != nil {
		return nil, err
	}
	subject := buf.String()

	if opts.OutFile == "" {
		opts.OutFile = outName(in)
	}

	sd := &service{
		Pkg:     pkg,
		PkgPath: pkgPath,
		Subject: subject,
		Name:    sp.GetName(),
	}

	for _, m := range sp.Method {
		sd.Methods = append(sd.Methods, &method{
			Name:       m.GetName(),
			Topic:      fmt.Sprintf("%s.%s", subject, m.GetName()),
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
