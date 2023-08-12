package literalgen_test

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"github.com/balazsgrill/literalgen"
)

type s struct {
	A any
}

func TestAnyString(t *testing.T) {
	g := literalgen.New("a")

	buf := bytes.NewBuffer(make([]byte, 0))
	v := s{
		A: "string",
	}
	e := g.AddTypedLiteral("v", reflect.TypeOf(s{}).Field(0).Type, reflect.ValueOf(v).Field(0))

	e.Emit(buf)

	sv := buf.String()
	if sv != "\"string\"" {
		log.Printf("%s != %s", sv, "\"string\"")
		t.Fail()
	}
}
