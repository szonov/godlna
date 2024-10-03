package scpd

import (
	"encoding/xml"
	"io"
	"os"
)

func (s *SCPD) Load(xmlData []byte) (err error) {
	err = xml.Unmarshal(xmlData, &s)
	return
}

func (s *SCPD) LoadFile(file string) (err error) {
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
	err = s.Load(xmlData)
	return
}

func (s *SCPD) GetVariable(stateVariableName string) *StateVariable {
	for _, stateVariable := range s.Variables {
		if stateVariableName == stateVariable.Name {
			return stateVariable
		}
	}
	return nil
}
