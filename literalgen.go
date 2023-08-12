package literalgen

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
)

type Generator struct {
	packagename    string
	packageimports map[string]string
	emitters       []Emitter
}

func New(packagename string) *Generator {
	return &Generator{
		packagename: packagename,
	}
}

type Emitter interface {
	Emit(io.Writer)
}

type literal struct {
	name      string
	generator *Generator
	t         reflect.Type
	value     reflect.Value
}

func (g *Generator) importPackage(path string) string {
	if path == "" {
		return ""
	}
	if g.packageimports == nil {
		g.packageimports = make(map[string]string)
	}
	short, ok := g.packageimports[path]
	if !ok {
		short = fmt.Sprintf("p%d", len(g.packageimports))
		g.packageimports[path] = short
	}
	return short
}

func (g *Generator) AddLiteral(name string, value any) Emitter {
	return g.AddTypedLiteral(name, reflect.TypeOf(value), reflect.ValueOf(value))
}

func (g *Generator) AddTypedLiteral(name string, t reflect.Type, value reflect.Value) Emitter {
	g.importType(t)
	e := &literal{
		name:      name,
		generator: g,
		t:         t,
		value:     value,
	}
	g.emitters = append(g.emitters, e)
	return e
}

func (g *Generator) importType(t reflect.Type) {
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		g.importType(t.Elem())
	}
	if t.Kind() == reflect.Struct {
		g.importPackage(t.PkgPath())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			g.importType(field.Type)
		}
	}
}

func (g *Generator) Emit(w io.Writer) {
	io.WriteString(w, "package ")
	io.WriteString(w, g.packagename)
	io.WriteString(w, "\n")
	if g.packageimports != nil {
		io.WriteString(w, "import (\n")
		for path, short := range g.packageimports {
			io.WriteString(w, "\t ")
			io.WriteString(w, short)
			io.WriteString(w, " \"")
			io.WriteString(w, path)
			io.WriteString(w, "\"\n")
		}
		io.WriteString(w, ")\n")
	}
	for _, emitter := range g.emitters {
		emitter.Emit(w)
	}
}

func (l *literal) Emit(w io.Writer) {
	io.WriteString(w, "var ")
	io.WriteString(w, l.name)
	io.WriteString(w, " ")
	l.generator.emitType(w, l.t)
	io.WriteString(w, " = ")
	l.generator.generateLiteral(w, l.t, l.value)
	io.WriteString(w, "\n")
}

func (g *Generator) emitType(w io.Writer, t reflect.Type) {
	pkgp := t.PkgPath()
	if pkgp != "" {
		io.WriteString(w, g.importPackage(pkgp))
		io.WriteString(w, ".")
	}
	io.WriteString(w, t.Name())
}

func (g *Generator) generateLiteral(w io.Writer, t reflect.Type, value reflect.Value) {
	if t.Kind() == reflect.Pointer {
		io.WriteString(w, "&")
		g.generateLiteral(w, t.Elem(), value.Elem())
		return
	}
	if t.Kind() == reflect.Struct {
		g.emitType(w, t)
		io.WriteString(w, "{\n")
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			io.WriteString(w, field.Name)
			io.WriteString(w, ":")
			g.generateLiteral(w, field.Type, value.FieldByName(field.Name))
			io.WriteString(w, ",\n")
		}
		io.WriteString(w, "}")
		return
	}
	if t.Kind() == reflect.String {
		io.WriteString(w, strconv.Quote(value.String()))
		return
	}
	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		io.WriteString(w, "[]")
		io.WriteString(w, t.Elem().Name())
		io.WriteString(w, "{\n")
		for i := 0; i < value.Len(); i++ {
			g.generateLiteral(w, t.Elem(), value.Index(i))
			io.WriteString(w, ",\n")
		}
		io.WriteString(w, "}")
		return
	}

	if t.Kind() == reflect.Interface {
		if value.IsNil() {
			io.WriteString(w, "nil")
		} else {
			v := value.Elem()
			g.generateLiteral(w, v.Type(), v)
		}
		return
	}
	if t.Kind() == reflect.Bool {
		io.WriteString(w, fmt.Sprint(value.Bool()))
		return
	}
	if t.Kind() == reflect.Int || t.Kind() == reflect.Int32 || t.Kind() == reflect.Int64 || t.Kind() == reflect.Int16 || t.Kind() == reflect.Int8 {
		io.WriteString(w, fmt.Sprint(value.Int()))
		return
	}
	if t.Kind() == reflect.Uint || t.Kind() == reflect.Uint32 || t.Kind() == reflect.Uint64 || t.Kind() == reflect.Uint16 || t.Kind() == reflect.Uint8 {
		io.WriteString(w, fmt.Sprint(value.Uint()))
		return
	}
	if t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64 {
		io.WriteString(w, fmt.Sprint(value.Float()))
		return
	}

	// Attempt to process according to generic type
	if value.Type() != t {
		g.generateLiteral(w, value.Type(), value)
		return
	}

	// default behavior
	io.WriteString(w, value.String())
}
