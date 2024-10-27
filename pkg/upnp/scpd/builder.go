package scpd

type VariableProperty struct {
	Name  string
	Value any
}

func IN(name, stateVariable string) Argument {
	return Argument{
		Name:      name,
		Direction: "in",
		Variable:  stateVariable,
	}
}

func OUT(name, stateVariable string) Argument {
	return Argument{
		Name:      name,
		Direction: "out",
		Variable:  stateVariable,
	}
}

func emptyProperty() VariableProperty {
	return VariableProperty{
		Name: "Empty",
	}
}

func Default(value string) VariableProperty {
	return VariableProperty{
		Name:  "Default",
		Value: value,
	}
}

func Range(min, max, step int) VariableProperty {
	if min == 0 && max == 0 && step == 0 {
		return emptyProperty()
	}
	return VariableProperty{
		Name:  "Range",
		Value: [3]int{min, max, step},
	}
}

func Only(values ...string) VariableProperty {
	if len(values) == 0 {
		return emptyProperty()
	}
	return VariableProperty{
		Name:  "Only",
		Value: &AllowedValueList{values},
	}
}

func Events() VariableProperty {
	return VariableProperty{
		Name: "Events",
	}
}

type DocumentBuilder struct {
	document *Document
}

func NewDocumentBuilder() *DocumentBuilder {
	return &DocumentBuilder{
		document: &Document{
			SpecVersion: Version,
		},
	}
}

func (c *DocumentBuilder) Version(major uint, minor uint) *DocumentBuilder {
	c.document.SpecVersion = SpecVersion{Major: major, Minor: minor}
	return c
}

func (c *DocumentBuilder) Action(name string, arguments ...Argument) *DocumentBuilder {
	c.document.Actions = append(c.document.Actions, Action{
		Name: name,
		Args: arguments,
	})
	return c
}

func (c *DocumentBuilder) Variable(name string, typ string, props ...VariableProperty) *DocumentBuilder {
	variable := StateVariable{
		Name:       name,
		SendEvents: "no",
		DataType:   typ,
	}
	for _, prop := range props {
		switch prop.Name {
		case "Default":
			variable.Default = prop.Value.(string)
		case "Range":
			val := prop.Value.([3]int)
			variable.AllowedValueRange = &AllowedValueRange{
				Min:  val[0],
				Max:  val[1],
				Step: val[2],
			}
		case "Only":
			variable.AllowedValues = prop.Value.(*AllowedValueList)
		case "Events":
			variable.SendEvents = "yes"
		}
	}
	c.document.StateVariables = append(c.document.StateVariables, variable)

	return c
}

func (c *DocumentBuilder) Document() *Document {
	return c.document
}
