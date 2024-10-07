package events

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	callbackRegexp     = regexp.MustCompile(`<([^<>]+)>`)
	DefaultTimeout int = 300
)

func ParseCallbackHeader(callback string) (ret []*url.URL, err error) {
	if callback == "" {
		err = fmt.Errorf("empty callback")
		return
	}
	list := callbackRegexp.FindAllStringSubmatch(callback, -1)
	for _, m := range list {
		var u *url.URL
		if u, err = url.Parse(m[1]); err != nil {
			return
		}
		ret = append(ret, u)
	}
	return
}

// ParseTimeoutHeader skips all errors,
// returns
// - parsed value in format Second-[INT],
// - default value in all other cases (even if header is "Second-infinite")
func ParseTimeoutHeader(timeout string) int {
	prefix := "Second-"
	if !strings.HasPrefix(timeout, prefix) {
		return DefaultTimeout
	}
	timeout = timeout[len(prefix):]
	if val, err := strconv.Atoi(timeout); err == nil && val > 0 {
		return val
	}
	return DefaultTimeout
}

type Subscribers struct {
	all map[string]*Subscriber
}

// Subscribe new subscriber and returns it
func (s Subscribers) Subscribe(sid string, callback []*url.URL, timeout int) *Subscriber {
	// todo: check exists and not expired
	//if subscriber, exists := s.all[sid]; exists {
	//	// already exists
	//	return nil
	//}
	subscriber := &Subscriber{
		SID:     sid,
		Url:     callback,
		Timeout: timeout,
	}
	s.all[sid] = subscriber
	return subscriber
}

func (s Subscribers) Renew(sid string, timeout int) *Subscriber {
	return new(Subscriber)
}

func (s Subscribers) Unsubscribe(sid string) int {
	return http.StatusOK
}

func (s Subscribers) NewSubscribe(callback []*url.URL, timeout int) (*Subscriber, int) {
	//5xx. If a publisher is not able to acc
	subscriber := &Subscriber{
		SID:     "uuid:122221212",
		Url:     callback,
		Timeout: timeout,
	}
	return subscriber, http.StatusOK
}
func (s Subscribers) RenewSubscribe(sid string, timeout int) (*Subscriber, int) {
	//5xx. If a publisher is not able to acc
	return new(Subscriber), http.StatusOK
}

type Subscriber struct {
	SID     string
	Url     []*url.URL
	Timeout int
}

type Property struct {
	Name    string
	Value   string
	updated bool
}

type PropertySet struct {
	Properties []*Property
}

func (p *PropertySet) AddProperty(name string) {
	// lock for writing does not require,
	// properties added only when controller born and should not be added dynamically later
	p.Properties = append(p.Properties, &Property{Name: name})
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
