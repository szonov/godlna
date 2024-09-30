package scpd

import (
	"encoding/xml"
)

func (s *SCPD) Load(xmlData []byte) (err error) {
	err = xml.Unmarshal(xmlData, &s)
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
