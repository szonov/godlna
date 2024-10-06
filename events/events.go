package events

type Subscribers struct {
	all map[string]*Subscriber
}

type Subscriber struct {
	SID string
}

type Property struct {
	Name    string
	Value   string
	updated bool
}

type PropertySet struct {
	Properties []*Property
}

func (p *PropertySet) AddProperty(name, value string) {
	// lock for writing does not require,
	// properties added only when controller born and should not be added dynamically later
	p.Properties = append(p.Properties, &Property{Name: name, Value: value})
}

func (p *PropertySet) Set(name, value string, initialState ...bool) bool {
	// todo: lock for writing
	for _, prop := range p.Properties {
		if prop != nil && prop.Name == name {
			if prop.Value != value {
				prop.Value = value
				if len(initialState) == 0 || !initialState[0] {
					prop.updated = true
				}
			}
			return true
		}
	}
	return false
}

func (p *PropertySet) Get(name string) string {
	// todo: lock for reading
	for _, prop := range p.Properties {
		if prop != nil && prop.Name == name {
			return prop.Value
		}
	}
	return ""
}
