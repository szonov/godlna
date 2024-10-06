package scpd

import (
	"encoding/xml"
	"io"
	"os"
)

// reference https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf
// page 30 and below: "2.3 Description: ServiceType description"

// SpecVersion The same as upnp.SpecVersion
type SpecVersion struct {
	Major uint `xml:"major"`
	Minor uint `xml:"minor"`
}

type Argument struct {
	// Name Required. Name of formal parameter. Should be name of a state variable that
	// models an effect the action causes. Must not contain a hyphen character (-, 2D 	33	Hex in UTF-8).
	// First character must be a USASCII letter (A-Z, a-z), USASCII digit	(0-9), an underscore ("_"),
	// or a non-experimental Unicode letter or digit greater than U+007F.
	// Succeeding characters must be a USASCII letter (A-Z, a-z),	USASCII digit (0-9), an underscore ("_"), a period ("."),
	// a Unicode combiningchar,	an extender, or a non-experimental Unicode letter or digit greater than U+007F.
	// The first three letters must not be "XML" in any combination of case.String.
	// Case-sensitive. Should be < 32 characters.
	Name string `xml:"name"`

	// Direction Required. Whether argument is an input or output parameter.
	// Must be in xor out. Any in arguments must be listed before any out arguments.
	Direction string `xml:"direction"`

	// Retval Optional. Identifies at most one out argument as the return value.
	// If included, must be the first out argument. (Element only; no value.)
	// I did not find any use of this property in the examples considered -> removed
	// Retval string `xml:"retval"`

	// Required. Must be the name of a state variable. Case Sensitive.
	// Defines the type of the argument
	Variable string `xml:"relatedStateVariable"`
}

type Action struct {
	// Name Required. Name of action. Must not contain a hyphen character (-, 2D Hex in UTF-8)
	// nor a hash character (#, 23 Hex in UTF-8). Case-sensitive. First character must be a USASCII
	// letter (A-Z, a-z), USASCII digit (0-9), an underscore ("_"), or a non-experimental Unicode
	// letter or digit greater than U+007F. Succeeding characters must be a USASCII letter
	// (A-Z,a-z), USASCII digit (0-9), an underscore ("_"), a period ("."), a Unicode combiningchar,
	// an extender, or a non-experimental Unicode letter or digit greater than U+007F.
	// The first three letters must not be "XML" in any combination of case.
	// • For standard actions defined by a UPnP Forum working committee, must not begin with X_ nor A_.
	// • For non-standard actions specified by a UPnP vendor and added to a standard service,
	//   must begin with X_.
	Name string `xml:"name"`

	// Arguments Required if and only if parameters are defined for action.
	Arguments []*Argument `xml:"argumentList>argument"`
}

func (a *Action) GetArgument(name string) *Argument {
	for _, arg := range a.Arguments {
		if arg != nil && name == arg.Name {
			return arg
		}
	}
	return nil
}

type AllowedRange struct {
	// Min Required. Inclusive lower bound. Single numeric value.
	Min int `xml:"minimum"`

	// Max Required. Inclusive upper bound. Single numeric value.
	Max int `xml:"maximum"`

	// Step Recommended. Size of an increment operation, i.e., value of s in the operation
	// v = v + s. Single numeric value.
	Step int `xml:"step,omitempty"`
}

type StateVariable struct {
	// Name Required, same rules as for Action.Name
	Name string `xml:"name"`

	// Events attribute defines whether event messages will be generated when
	// the value of this state variable changes; non-evented state variables have sendEvents="no";
	// default is sendEvents="yes".
	Events string `xml:"sendEvents,attr,omitempty"`

	// DataType Required. Same as data types defined by XML Schema, Part 2: Datatypes.
	DataType string `xml:"dataType"`

	// Default Recommended. Expected, initial value. Defined by a UPnP Forum working committee or
	// delegated to UPnP vendor. Must match data type. Must satisfy allowedValueList or allowedValueRange constraints.
	Default string `xml:"defaultValue,omitempty"`

	// AllowedValues Recommended. Enumerates legal string values. Prohibited for data types other than
	// string. At most one of allowedValueRange and allowedValueList may be specified. Subelements are ordered
	// Every subelement is string and must be < 32 characters.
	AllowedValues *[]string `xml:"allowedValueList>allowedValue,omitempty"`

	// AllowedRange Recommended. Defines bounds for legal numeric values; defines resolution for numeric
	// values. Defined only for numeric data types. At most one of allowedValueRange and allowedValueList may be specified.
	AllowedRange *AllowedRange `xml:"allowedValueRange,omitempty"`
}

type SCPD struct {
	XMLName     xml.Name         `xml:"urn:schemas-upnp-org:service-1-0 scpd"`
	SpecVersion SpecVersion      `xml:"specVersion"`
	Actions     []Action         `xml:"actionList>action"`
	Variables   []*StateVariable `xml:"serviceStateTable>stateVariable"`
}

func (s *SCPD) GetVariable(name string) *StateVariable {
	for _, stateVariable := range s.Variables {
		if name == stateVariable.Name {
			return stateVariable
		}
	}
	return nil
}

func (s *SCPD) GetAction(name string) *Action {
	for _, action := range s.Actions {
		if name == action.Name {
			return &action
		}
	}
	return nil
}

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
