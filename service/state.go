package service

import (
	"github.com/szonov/go-upnp-lib/events"
	"sync"
)

type (
	State struct {
		events    *events.Manager
		variables sync.Map
	}
)

func NewState(eventfulVariables []string) *State {
	state := &State{
		events: events.NewManager(),
	}
	for _, variableName := range eventfulVariables {
		state.variables.Store(variableName, "")
	}
	return state
}

func (s *State) GetVariable(name string) string {
	if v, ok := s.variables.Load(name); ok {
		return v.(string)
	}
	return ""
}

func (s *State) SetVariable(name string, value string) *State {
	if _, ok := s.variables.Load(name); ok {
		s.variables.Store(name, value)
	}
	return s
}

func (s *State) AllVariables() map[string]string {
	ret := make(map[string]string)
	s.variables.Range(func(key, value interface{}) bool {
		ret[key.(string)] = value.(string)
		return true
	})
	return ret
}

func (s *State) Subscribe(sid string, nt string, callback string, timeout string) events.SubscribeResult {
	res := s.events.Subscribe(sid, nt, callback, timeout)
	if res.Success && res.IsNewSubscription {
		s.events.SendInitialState(res.SID, s.AllVariables())
	}
	return res
}

func (s *State) Unsubscribe(sid string, nt string, callback string) int {
	return s.events.Unsubscribe(sid, nt, callback)
}

func (s *State) NotifyChanges(changedVariableNames ...string) {
	changedVariables := make(map[string]string)

	for _, name := range changedVariableNames {
		if v, ok := s.variables.Load(name); ok {
			changedVariables[name] = v.(string)
		}
	}
	s.events.NotifyChanges(changedVariables)
}
