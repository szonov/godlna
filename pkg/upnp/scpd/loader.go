package scpd

import (
	"encoding/xml"
	"io"
	"os"
)

func (doc *Document) Load(xmlData []byte) (err error) {
	err = xml.Unmarshal(xmlData, &doc)
	return
}

func (doc *Document) LoadFile(file string) (err error) {
	var fp *os.File
	var xmlData []byte

	if fp, err = os.Open(file); err != nil {
		return
	}
	defer func() {
		_ = fp.Close()
	}()

	if xmlData, err = io.ReadAll(fp); err != nil {
		return
	}
	err = doc.Load(xmlData)
	return
}
