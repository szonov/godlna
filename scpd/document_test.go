package scpd

import (
	"encoding/xml"
	"fmt"
	"testing"
)

type Context struct {
	Action *Action
}

func TestDocument(t *testing.T) {

	doc := new(Document)
	if err := doc.LoadFile("./scpd_template.xml"); err != nil {
		t.Fatal(err)
	}

	if b, err := xml.MarshalIndent(doc, "", "  "); err == nil {
		fmt.Println(string(b))
	}
}
