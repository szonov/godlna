package service

import (
	scpd "github.com/szonov/go-upnp-lib/scpd"
)

type (
	Config struct {
		serviceType       string
		actions           map[string]ConfigAction
		eventfulVariables []string
	}
	ConfigAction struct {
		in  []string
		out []string
	}
)

func NewConfig(serviceType string) *Config {
	return &Config{
		serviceType:       serviceType,
		actions:           make(map[string]ConfigAction),
		eventfulVariables: make([]string, 0),
	}
}

func (c *Config) AddAction(name string, in []string, out []string) {
	c.actions[name] = ConfigAction{in, out}
}

func (c *Config) AddEventfulVariables(variables ...string) {
	c.eventfulVariables = append(c.eventfulVariables, variables...)
}

func (c *Config) EventfulVariables() []string {
	return c.eventfulVariables
}

func (c *Config) FromDocument(doc *scpd.Document) *Config {
	c.parseActions(doc)
	c.parseEventfulVariables(doc)
	return c
}

func (c *Config) parseEventfulVariables(doc *scpd.Document) {
	for _, variable := range doc.StateVariables {
		if variable.SendEvents == "yes" {
			c.AddEventfulVariables(variable.Name)
		}
	}
}

func (c *Config) parseActions(doc *scpd.Document) {
	for _, action := range doc.Actions {
		in := make([]string, 0)
		out := make([]string, 0)
		for _, arg := range action.Args {
			if arg.Direction == "in" {
				in = append(in, arg.Name)
			} else {
				out = append(out, arg.Name)
			}
		}
		c.AddAction(action.Name, in, out)
	}
}

func (c *Config) NewAction(actionName string) *Action {
	if ac, ok := c.actions[actionName]; ok {
		action := &Action{
			Name:        actionName,
			ServiceType: c.serviceType,
			ArgIn:       make(ActionArgs),
			ArgOut:      make(ActionArgs),
		}
		for _, a := range ac.in {
			action.ArgIn[a] = ""
		}
		for _, a := range ac.out {
			action.ArgOut[a] = ""
		}
		return action
	}
	return nil
}
