package events

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

var (
	CleanAfterVisits uint32 = 500
)

type (
	Manager struct {
		subscribers sync.Map
		visits      uint32
	}

	Subscriber struct {
		SID     string
		URLs    []url.URL
		Timeout time.Time
		Seq     uint32
	}

	SubscribeResult struct {
		// Success indicates success operation
		Success bool
		// StatusCode  is exact http status code which should be sent to the subscriber
		StatusCode int
		// SID is subscriber id, new or existing
		SID string
		// Timeout is the actual timeout string a ready to add as TIMEOUT header value
		TimeoutHeaderString string
		// IsNewSubscription is flag shown new subscription, required to send initial status (if true)
		IsNewSubscription bool
	}
)

func (s Subscriber) IsExpired() bool {
	return s.Timeout.Before(time.Now())
}

func NewManager() *Manager {
	return &Manager{
		visits: 0,
	}
}

func (m *Manager) Clean() {
	slog.Debug("clean up expired subscribers")
	m.subscribers.Range(func(sid, v interface{}) bool {
		if v.(Subscriber).IsExpired() {
			m.subscribers.Delete(sid)
		}
		return true
	})
}

func (m *Manager) checkForCleanup() {
	if visits := atomic.AddUint32(&m.visits, 1); visits == CleanAfterVisits {
		atomic.StoreUint32(&m.visits, 0)
		m.Clean()
	}
}

func (m *Manager) Subscribe(sid string, nt string, callback string, timeout string) SubscribeResult {

	tm := ParseTimeoutHeader(timeout)
	res := SubscribeResult{
		SID: sid,
	}
	if sid == "" {
		res.IsNewSubscription = true
	}
	if res.IsNewSubscription {
		if nt != "upnp:event" {
			res.StatusCode = http.StatusPreconditionFailed
			return res
		}
		urls, err := ParseCallbackHeader(callback)
		if err != nil || len(urls) == 0 {
			res.StatusCode = http.StatusPreconditionFailed
			return res
		}
		// make new sid
		res.SID = NewSID(callback)
		res.StatusCode = m.newSubscribe(res.SID, urls, tm)
	} else {
		if nt != "" || callback != "" {
			res.StatusCode = http.StatusBadRequest
			return res
		}
		res.StatusCode = m.renew(res.SID, tm)
	}

	if res.StatusCode == http.StatusOK {
		res.Success = true
		res.TimeoutHeaderString = fmt.Sprintf("Second-%d", int(time.Until(tm).Seconds()))
	}

	m.checkForCleanup()
	return res
}

func (m *Manager) Unsubscribe(sid string, nt string, callback string) int {
	if nt != "" || callback != "" {
		return http.StatusBadRequest
	}
	if sid == "" {
		return http.StatusPreconditionFailed
	}
	if _, ok := m.subscribers.LoadAndDelete(sid); !ok {
		return http.StatusPreconditionFailed
	}
	return http.StatusOK
}

func (m *Manager) newSubscribe(sid string, callback []url.URL, timeout time.Time) int {
	if _, exists := m.subscribers.Load(sid); exists {
		// md5 collision??
		return http.StatusInternalServerError
	}
	m.subscribers.Store(sid, Subscriber{
		SID:     sid,
		URLs:    callback,
		Timeout: timeout,
		Seq:     0,
	})
	return http.StatusOK
}

func (m *Manager) renew(sid string, timeout time.Time) int {
	s, exists := m.subscribers.Load(sid)
	if !exists {
		return http.StatusPreconditionFailed
	}
	subscriber := s.(Subscriber)
	subscriber.Timeout = timeout
	m.subscribers.Store(sid, subscriber)
	return http.StatusOK
}

func (m *Manager) NotifyAll(stateVariables map[string]string) {
	if len(stateVariables) == 0 {
		return
	}
	body, err := BuildNotificationBody(stateVariables)
	if err != nil {
		slog.Debug("Failed to build notification body", "err", err.Error())
		return
	}
	m.subscribers.Range(func(sid, v interface{}) bool {
		subscriber := v.(Subscriber)
		subscriber.Seq++
		// To prevent overflow, must be wrapped from 4294967295 to 1.
		// since Seq is unit32 it is become "0" after overlap max 32bit number, so increment only zero value
		if subscriber.Seq == 0 {
			subscriber.Seq++
		}
		m.subscribers.Store(sid, subscriber)
		m.notifySubscriber(subscriber, body)
		return true
	})
}

func (m *Manager) SendInitialState(sid string, stateVariables map[string]string) {
	if len(stateVariables) == 0 {
		return
	}
	v, exists := m.subscribers.Load(sid)
	if !exists {
		return
	}
	subscriber := v.(Subscriber)
	subscriber.Seq = 0 // it is already should be "0", but...
	body, err := BuildNotificationBody(stateVariables)
	if err != nil {
		slog.Debug("Failed to build notification body", "err", err.Error())
		return
	}
	go func() {
		// function calls when new subscribe happens,
		// give a time to close connection and deliver "SID" value to subscriber
		// doc: page 65. > The response must be sent within 30 seconds, including expected transmission time.
		time.Sleep(2 * time.Second)
		m.notifySubscriber(subscriber, body)
	}()
}

func (m *Manager) notifySubscriber(s Subscriber, body []byte) {
	if s.IsExpired() {
		// skip sending to expired subscribers
		m.subscribers.Delete(s.SID)
		return
	}
	for _, u := range s.URLs {
		if err := SendNotification(s.SID, s.Seq, u, body); err != nil {
			slog.Debug("Failed to send notification",
				slog.String("err", err.Error()),
				slog.String("to", u.String()),
				slog.String("sid", s.SID),
			)
		}
	}
}

func (m *Manager) Dump() {
	m.subscribers.Range(func(k, v interface{}) bool {
		ss := v.(Subscriber)
		slog.Info("SUBSCRIBER", "sid", ss.SID, "seq", ss.Seq, "urls", ss.URLs, "expired", ss.IsExpired())
		return true
	})
}
