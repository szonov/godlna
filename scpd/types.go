package scpd

import "encoding/xml"

// reference https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf
// page 30 and below: "2.3 Description: Service description"

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
	// Case sensitive. Should be < 32 characters.
	Name string `xml:"name"`

	// Direction Required. Whether argument is an input or output parameter.
	// Must be in xor out. Any in arguments must be listed before any out arguments.
	Direction string `xml:"direction"`

	// Retval Optional. Identifies at most one out argument as the return value.
	// If included, must be the first out argument. (Element only; no value.)
	Retval string `xml:"retval"`

	// Required. Must be the name of a state variable. Case Sensitive.
	// Defines the type of the argument
	RelatedStateVar string `xml:"relatedStateVariable"`
}

type Action struct {
	// Name Required. Name of action. Must not contain a hyphen character (-, 2D Hex in UTF-8)
	// nor a hash character (#, 23 Hex in UTF-8). Case sensitive. First character must be a USASCII
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

type AllowedValueRange struct {
	// Minimum Required. Inclusive lower bound. Single numeric value.
	Minimum int `xml:"minimum"`

	// Maximum Required. Inclusive upper bound. Single numeric value.
	Maximum int `xml:"maximum"`

	// Step Recommended. Size of an increment operation, i.e., value of s in the operation
	// v = v + s. Single numeric value.
	Step int `xml:"step,omitempty"`
}

type StateVariable struct {
	// Name Required, same rules as for Action.Name
	Name string `xml:"name"`

	// SendEvents attribute defines whether event messages will be generated when
	// the value of this state variable changes; non-evented state variables have sendEvents="no";
	// default is sendEvents="yes".
	SendEvents string `xml:"sendEvents,attr,omitempty"`

	// Required. Same as data types defined by XML Schema, Part 2: Datatypes.
	DataType string `xml:"dataType"`

	// DefaultValue Recommended. Expected, initial value. Defined by a UPnP Forum working committee or
	// delegated to UPnP vendor. Must match data type. Must satisfy allowedValueList or allowedValueRange constraints.
	DefaultValue string `xml:"defaultValue,omitempty"`

	// AllowedValueList Recommended. Enumerates legal string values. Prohibited for data types other than
	// string. At most one of allowedValueRange and allowedValueList may be specified. Subelements are ordered
	// Every subelement is string and must be < 32 characters.
	AllowedValueList *[]string `xml:"allowedValueList>allowedValue,omitempty"`

	// AllowedValueRange Recommended. Defines bounds for legal numeric values; defines resolution for numeric
	// values. Defined only for numeric data types. At most one of allowedValueRange and allowedValueList may be specified.
	AllowedValueRange *AllowedValueRange `xml:"allowedValueRange,omitempty"`
}

type SCPD struct {
	XMLName           xml.Name         `xml:"urn:schemas-upnp-org:service-1-0 scpd"`
	SpecVersion       SpecVersion      `xml:"specVersion"`
	ActionList        []Action         `xml:"actionList>action"`
	ServiceStateTable []*StateVariable `xml:"serviceStateTable>stateVariable"`
}