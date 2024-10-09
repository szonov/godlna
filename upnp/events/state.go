package events

import (
	"net/http"
	"strconv"
	"sync"
)

type (
	State struct {
		events    *Manager
		variables sync.Map
	}
)

func NewState(eventfulVariables []string) *State {
	state := &State{
		events: NewManager(),
	}
	for _, variableName := range eventfulVariables {
		state.variables.Store(variableName, "")
	}
	return state
}

func (s *State) Get(name string) string {
	if v, ok := s.variables.Load(name); ok {
		return v.(string)
	}
	return ""
}

func (s *State) GetUint32(name string) uint32 {
	if u, err := strconv.ParseUint(s.Get(name), 10, 32); err == nil {
		return uint32(u)
	}
	return 0
}

func (s *State) Set(name string, value string) *State {
	if _, ok := s.variables.Load(name); ok {
		s.variables.Store(name, value)
	}
	return s
}

func (s *State) SetUint32(name string, value uint32) *State {
	s.Set(name, strconv.FormatUint(uint64(value), 10))
	return s
}

func (s *State) All() map[string]string {
	ret := make(map[string]string)
	s.variables.Range(func(key, value interface{}) bool {
		ret[key.(string)] = value.(string)
		return true
	})
	return ret
}

func (s *State) Subscribe(sid string, nt string, callback string, timeout string) SubscribeResult {
	res := s.events.Subscribe(sid, nt, callback, timeout)
	if res.Success && res.IsNewSubscription {
		s.events.SendInitialState(res.SID, s.All())
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

func (s *State) NetHttpEventSubURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "SUBSCRIBE" {
		res := s.Subscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
			r.Header.Get("TIMEOUT"),
		)
		if res.Success {
			w.Header()["SID"] = []string{res.SID}
			w.Header()["TIMEOUT"] = []string{res.TimeoutHeaderString}
		}
		w.WriteHeader(res.StatusCode)
	} else if r.Method == "UNSUBSCRIBE" {
		statusCode := s.Unsubscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
		)
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
