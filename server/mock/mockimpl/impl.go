// impl generates method stubs for implementing an interface.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

const usage = `impl [o output.go] <recv> <iface>

impl generates method stubs for recv to implement iface.

Examples:

impl 'f *File' io.Reader
impl Murmur hash.Hash

Don't forget the single quotes around the receiver type
to prevent shell globbing.
`

// findInterface returns the import path and identifier of an interface.
// For example, given "http.ResponseWriter", findInterface returns
// "net/http", "ResponseWriter".
// If a fully qualified interface is given, such as "net/http.ResponseWriter",
// it simply parses the input.
func findInterface(iface string) (path string, id string, err error) {
	if len(strings.Fields(iface)) != 1 {
		return "", "", fmt.Errorf("couldn't parse interface: %s", iface)
	}

	if slash := strings.LastIndex(iface, "/"); slash > -1 {
		// package path provided
		dot := strings.LastIndex(iface, ".")
		// make sure iface does not end with "/" (e.g. reject net/http/)
		if slash+1 == len(iface) {
			return "", "", fmt.Errorf("interface name cannot end with a '/' character: %s", iface)
		}
		// make sure iface does not end with "." (e.g. reject net/http.)
		if dot+1 == len(iface) {
			return "", "", fmt.Errorf("interface name cannot end with a '.' character: %s", iface)
		}
		// make sure iface has exactly one "." after "/" (e.g. reject net/http/httputil)
		if strings.Count(iface[slash:], ".") != 1 {
			return "", "", fmt.Errorf("invalid interface name: %s", iface)
		}
		return iface[:dot], iface[dot+1:], nil
	}

	src := []byte("package hack\n" + "var i " + iface)
	// If we couldn't determine the import path, goimports will
	// auto fix the import path.
	imp, err := imports.Process(".", src, nil)
	if err != nil {
		return "", "", fmt.Errorf("couldn't parse interface: %s", iface)
	}

	// imp should now contain an appropriate import.
	// Parse out the import and the identifier.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", imp, 0)
	if err != nil {
		panic(err)
	}
	if len(f.Imports) == 0 {
		return "", "", fmt.Errorf("unrecognized interface: %s", iface)
	}
	raw := f.Imports[0].Path.Value   // "io"
	path, err = strconv.Unquote(raw) // io
	if err != nil {
		panic(err)
	}
	decl := f.Decls[1].(*ast.GenDecl)      // var i io.Reader
	spec := decl.Specs[0].(*ast.ValueSpec) // i io.Reader
	sel := spec.Type.(*ast.SelectorExpr)   // io.Reader
	id = sel.Sel.Name                      // Reader
	return path, id, nil
}

// Pkg is a parsed build.Package.
type Pkg struct {
	*build.Package
	*token.FileSet
}

// typeSpec locates the *ast.TypeSpec for type id in the import path.
func typeSpec(path string, id string) (Pkg, *ast.TypeSpec, error) {
	pkg, err := build.Import(path, "", 0)
	if err != nil {
		return Pkg{}, nil, fmt.Errorf("couldn't find package %s: %v", path, err)
	}

	fset := token.NewFileSet() // share one fset across the whole package
	for _, file := range pkg.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, file), nil, 0)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			decl, ok := decl.(*ast.GenDecl)
			if !ok || decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.TypeSpec)
				if spec.Name.Name != id {
					continue
				}
				return Pkg{Package: pkg, FileSet: fset}, spec, nil
			}
		}
	}
	return Pkg{}, nil, fmt.Errorf("type %s not found in %s", id, path)
}

// gofmt pretty-prints e.
func (p Pkg) gofmt(e ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.FileSet, e)
	return buf.String()
}

// fullType returns the fully qualified type of e.
// Examples, assuming package net/http:
//
//	fullType(int) => "int"
//	fullType(Handler) => "http.Handler"
//	fullType(io.Reader) => "io.Reader"
//	fullType(*Request) => "*http.Request"
func (p Pkg) fullType(e ast.Expr) string {
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			// Using typeSpec instead of IsExported here would be
			// more accurate, but it'd be crazy expensive, and if
			// the type isn't exported, there's no point trying
			// to implement it anyway.
			if n.IsExported() {
				n.Name = p.Package.Name + "." + n.Name
			}
		case *ast.SelectorExpr:
			return false
		}
		return true
	})
	return p.gofmt(e)
}

func (p Pkg) params(field *ast.Field, defaultName string) []Param {
	var params []Param
	typ := p.fullType(field.Type)
	for _, name := range field.Names {
		params = append(params, Param{Name: name.Name, Type: typ})
	}
	// Handle anonymous params
	if len(params) == 0 {
		params = []Param{{Type: typ, Name: defaultName}}
	}
	return params
}

// Method represents a method signature.
type Method struct {
	RecvShort string
	Recv      string
	Func
}

type Struc struct {
	IName string
	Func
}

// Func represents a function signature.
type Func struct {
	Name   string
	Params []Param
	Res    []Param
}

// Param represents a parameter in a function or method signature.
type Param struct {
	Name string
	Type string
}

// CalledArgument will correctly generate call to a function with a
// variadic parameter
func (p *Param) CalledArgument() string {
	variadic, _ := regexp.MatchString("^[.]{3}", p.Type)
	if variadic {
		return p.Name + "..."
	}
	return p.Name
}

func (p Pkg) funcsig(f *ast.Field) Func {
	fn := Func{Name: f.Names[0].Name}
	typ := f.Type.(*ast.FuncType)
	if typ.Params != nil {
		for pos, field := range typ.Params.List {
			defaultName := fmt.Sprintf("p%d", pos)
			fn.Params = append(fn.Params, p.params(field, defaultName)...)
		}
	}
	if typ.Results != nil {
		for _, field := range typ.Results.List {
			fn.Res = append(fn.Res, p.params(field, "")...)
		}
	}
	return fn
}

// funcs returns the set of methods required to implement iface.
// It is called funcs rather than methods because the
// function descriptions are functions; there is no receiver.
func funcs(iface string) ([]Func, error) {
	// Locate the interface.
	path, id, err := findInterface(iface)
	if err != nil {
		return nil, err
	}

	// Parse the package and find the interface declaration.
	p, spec, err := typeSpec(path, id)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %s", iface, err)
	}
	idecl, ok := spec.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("not an interface: %s", iface)
	}

	if idecl.Methods == nil {
		return nil, fmt.Errorf("empty interface: %s", iface)
	}

	var fns []Func
	for _, fndecl := range idecl.Methods.List {
		if len(fndecl.Names) == 0 {
			// Embedded interface: recurse
			embedded, err := funcs(p.fullType(fndecl.Type))
			if err != nil {
				return nil, err
			}
			fns = append(fns, embedded...)
			continue
		}

		fn := p.funcsig(fndecl)
		fns = append(fns, fn)
	}
	return fns, nil
}

const stub = "func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n" + "{{.RecvShort}}.mu.Lock()" + "\n" +
	"{{.RecvShort}}.{{.Name}}FuncInvoked = true" + "\n" +
	"{{.RecvShort}}.mu.Unlock()" + "\n" +
	"return {{.RecvShort}}.{{.Name}}Func({{range .Params}}{{.CalledArgument}}, {{end}})" +
	"\n" + "}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

// genStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
// If recv is not a valid receiver expression,
// genStubs will panic.
func genStubs(recv string, fns []Func) []byte {
	var buf bytes.Buffer
	for _, fn := range fns {
		meth := Method{Recv: recv, RecvShort: shortRecv(recv), Func: fn}
		tmpl.Execute(&buf, meth) //nolint:errcheck
	}

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return pretty
}

func shortRecv(recv string) string {
	s := strings.SplitN(recv, "*", 2)[0]
	return s
}

const packageStr = "// Automatically generated by mockimpl. DO NOT EDIT!" +
	"\n\n" + "package mock" + "\n\n"

const str = "{{.Name}}Func  {{.Name}}Func" +
	"\n" + "{{.Name}}FuncInvoked bool" +
	"\n\n"

const funcTypeStr = "type {{.Name}}Func  func" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"\n\n"

var (
	tmplStr         = template.Must(template.New("testtwo").Parse(str))
	funcTypetmplStr = template.Must(template.New("funcTypetmpl").Parse(funcTypeStr))
)

func genStr(name string, fns []Func) []byte {
	var buf bytes.Buffer
	for _, fn := range fns {
		meth := Struc{IName: name, Func: fn}
		funcTypetmplStr.Execute(&buf, meth) //nolint:errcheck
	}
	buf.WriteString("type ")
	buf.WriteString(name)
	buf.WriteString(" struct {\n")
	for _, fn := range fns {
		meth := Struc{IName: name, Func: fn}
		tmplStr.Execute(&buf, meth) //nolint:errcheck
	}
	buf.WriteString("\n")
	buf.WriteString("mu sync.Mutex")
	buf.WriteString("\n")
	buf.WriteString("}")

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return pretty
	// return buf.Bytes()
}

// validReceiver reports whether recv is a valid receiver expression.
func validReceiver(recv string) bool {
	if recv == "" {
		// The parse will parse empty receivers, but we don't want to accept them,
		// since it won't generate a usable code snippet.
		return false
	}
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", "package hack\nfunc ("+recv+") Foo()", 0)
	return err == nil
}

func main() {
	flOut := flag.String("o", "", "output file")
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	recv, iface := args[0], args[1]
	if !validReceiver(recv) {
		fatal(fmt.Sprintf("invalid receiver: %q", recv))
	}

	fns, err := funcs(iface)
	if err != nil {
		fatal(err)
	}

	src := genStubs(recv, fns)
	recName := strings.SplitN(recv, " ", 2)
	name := strings.TrimPrefix(recName[1], "*")
	src2 := genStr(name, fns)

	path, ifaceID, err := findInterface(iface)
	if err != nil {
		fatal(err)
	}

	var buf bytes.Buffer
	fmt.Fprint(&buf, packageStr)
	fmt.Fprintf(&buf, "import \"%s\"\n\n", path)
	fmt.Fprintf(&buf, "var _ %s.%s = (*%s)(nil)\n\n", filepath.Base(path), ifaceID, name)
	fmt.Fprint(&buf, string(src2))
	buf.WriteString("\n")
	fmt.Fprint(&buf, string(src))
	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	imp, err := imports.Process("", pretty, nil)
	if err != nil {
		panic(err)
	}
	switch *flOut {
	case "":
		fmt.Println(string(imp))
	default:
		f, err := os.Create(*flOut)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		_, err = f.Write(imp)
		if err != nil {
			log.Fatal(err) //nolint:gocritic // ignore exitAfterDefer
		}
	}
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
