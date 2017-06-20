package main

import (
	"go/types"
	"log"
)

type Interface struct {
	PkgPath string
	Pkg     string
	Name    string
	Methods []*Method
}

type Method struct {
	Pkg          string
	Name         string
	Topic        string
	Request      *Var
	Response     *Var
	ServiceGroup string

	ins  []*Var
	outs []*Var
}

type Var struct {
	Pkg  string
	Type string
	Ptr  bool
}

func reflectInterface(iface *types.Interface) *Interface {
	var x Interface

	// Method count.
	nm := iface.NumMethods()

	x.Methods = make([]*Method, nm)

	for i := 0; i < nm; i++ {
		m := iface.Method(i)
		x.Methods[i] = reflectMethod(m)
	}

	return &x
}

func reflectMethod(m *types.Func) *Method {
	sig := m.Type().(*types.Signature)
	params := sig.Params()
	results := sig.Results()

	if params.Len() != 2 {
		log.Fatalf("expected 2 params, got %d", params.Len())
	}

	if results.Len() != 2 {
		log.Fatalf("expected 2 results, got %d", results.Len())
	}

	x := Method{
		Name: m.Name(),
	}

	x.Request = reflectVar(params.At(1))
	x.Response = reflectVar(results.At(0))

	return &x
}

func reflectVar(v *types.Var) *Var {
	var x Var

	t := v.Type()

	switch u := t.(type) {
	case *types.Named:
		o := u.Obj()
		x.Type = o.Name()
		p := o.Pkg()
		if p != nil {
			x.Pkg = p.Name()
		}
	case *types.Pointer:
		x.Ptr = true
		o := u.Elem().(*types.Named).Obj()
		x.Type = o.Name()
		p := o.Pkg()
		if p != nil {
			x.Pkg = p.Name()
		}
	}

	return &x
}
