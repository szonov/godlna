package scpd

func NewDocument(majorMinor ...uint) *Document {
	doc := &Document{
		SpecVersion: Version,
	}
	if len(majorMinor) > 0 {
		doc.SpecVersion.Major = majorMinor[0]
	}
	if len(majorMinor) > 1 {
		doc.SpecVersion.Minor = majorMinor[1]
	}
	return doc
}

func (doc *Document) Action(name string, arguments ...Argument) *Document {
	doc.Actions = append(doc.Actions, Action{
		Name: name,
		Args: arguments,
	})
	return doc
}

func (doc *Document) Variable(name string, typ string, props ...VariableProperty) *Document {
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
	doc.StateVariables = append(doc.StateVariables, variable)

	return doc
}

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
