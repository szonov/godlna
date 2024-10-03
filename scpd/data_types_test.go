package scpd

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"
)

type Args struct {
	Kino UI1
	Link URI
}

func dump(key string, v any) {
	fmt.Printf("DUMP[%s]: %v %T\n", key, v, v)
}

func TestName(t *testing.T) {
	x := `<Action><Kino>10</Kino><Link>https://www.example.com/?a=56</Link></Action>`

	args := &Args{}
	if err := xml.Unmarshal([]byte(x), args); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("V: %d\n", args.Kino)

	xxx := uint8(args.Kino * 120)

	dump("xxx", xxx)
	x2 := reflect.TypeOf(UI1(0))
	dump("x2", x2)
	dump("STR", args.Kino.String())

	args.Kino = 255

	dump("LINK", args.Link.RawQuery)

	dump("REF", reflect.TypeOf(UI1(0)))
}
