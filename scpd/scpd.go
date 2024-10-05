package scpd

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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

// String generate scpd tag for StateVariable in format
// "{state_variable},{data_type},[events,][min={min},][max={int},][step={int},][default={default}][ {allowed}]"
// - {state_variable} - required, name of state variable, example: SystemUpdateID
// - {data_type} - required, data type of state variable, example: ui4
// - events - optional, appears only if sendEvents="yes"
// - min={min} - optional, Minimum Range value, skipped if empty or 0, example min=4
// - max={max} - optional, Maximum Range value, skipped if empty or 0, example max=10
// - step={step} - optional, Step Range value, skipped if empty or 0, example step=1
// - default={default} - optional, Default value, skipped if empty example default=NORMAL
// - {allowed} - optional, comma separated list of allowed values, IMPORTANT: have space before
// Example:
//
//	OnLine string `scpd:"A_ARG_TYPE_OnLine,string,events,default=line busy,line,broken"`
func (sv *StateVariable) String() string {
	ret := sv.Name + "," + sv.DataType
	if sv.Events == "yes" {
		ret += ",events"
	}
	if sv.AllowedRange != nil {
		if sv.AllowedRange.Min != 0 {
			ret += ",min=" + strconv.Itoa(sv.AllowedRange.Min)
		}
		if sv.AllowedRange.Max != 0 {
			ret += ",max=" + strconv.Itoa(sv.AllowedRange.Max)
		}
		if sv.AllowedRange.Step != 0 {
			ret += ",step=" + strconv.Itoa(sv.AllowedRange.Step)
		}
	}
	if sv.Default != "" {
		ret += ",default=" + sv.Default
	}
	if sv.AllowedValues != nil && len(*sv.AllowedValues) > 0 {
		av := make([]string, 0)
		for _, avv := range *sv.AllowedValues {
			if avv != "" {
				av = append(av, avv)
			}
		}
		if len(av) > 0 {
			ret += " " + strings.Join(av, ",")
		}
	}
	return ret
}

// LoadString restores *StateVariable from scpd tag
func (sv *StateVariable) LoadString(s string) error {
	var err error
	if s == "" {
		return fmt.Errorf("scpd: empty tag")
	}
	parts := strings.SplitN(s, " ", 2)
	main := strings.Split(parts[0], ",")
	if len(main) < 2 {
		return fmt.Errorf("scpd: not enough parts ('%s')", s)
	}
	rng := &AllowedRange{}
	for i, val := range main {
		if val == "" {
			return fmt.Errorf("scpd: empty part ('%s')", s)
		}
		switch i {
		case 0:
			sv.Name = val
		case 1:
			sv.DataType = val
		default:
			if val == "events" {
				sv.Events = "yes"
			} else {
				p := strings.SplitN(val, "=", 2)
				if len(p) != 2 {
					return fmt.Errorf("scpd: invalid part: %s ('%s')", val, s)
				}
				switch p[0] {
				case "min":
					if rng.Min, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid min: %s ('%s')", val, s)
					}
				case "max":
					if rng.Max, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid max: %s ('%s')", val, s)
					}
				case "step":
					if rng.Step, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid step: %s ('%s')", val, s)
					}
				case "default":
					sv.Default = p[1]
				}
			}
		}
	}
	if sv.Events == "" {
		sv.Events = "no"
	}
	if rng.Min != 0 || rng.Max != 0 || rng.Step != 0 {
		sv.AllowedRange = rng
	}
	if len(parts) == 2 {
		val := strings.Trim(parts[1], " ")
		if val != "" {
			values := strings.Split(val, ",")
			sv.AllowedValues = &values
		}
	}
	return nil
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
