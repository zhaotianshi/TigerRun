package capture

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const maxStoredSessions = 2000

type Store struct {
	mu       sync.RWMutex
	nextID   atomic.Uint64
	sessions []*CapturedSession
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(session *CapturedSession) string {
	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now()
	}
	if session.FinishedAt.IsZero() {
		session.FinishedAt = time.Now()
	}
	session.ID = fmt.Sprintf("%06d", s.nextID.Add(1))
	session.Timing = Timing{
		StartedAt:  session.StartedAt.Format(time.RFC3339Nano),
		FinishedAt: session.FinishedAt.Format(time.RFC3339Nano),
		DurationMs: maxInt64(0, session.FinishedAt.Sub(session.StartedAt).Milliseconds()),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append(s.sessions, cloneSession(session))
	if len(s.sessions) > maxStoredSessions {
		s.sessions = append([]*CapturedSession(nil), s.sessions[len(s.sessions)-maxStoredSessions:]...)
	}
	return session.ID
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = nil
}

func (s *Store) Summaries() []SessionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SessionSummary, 0, len(s.sessions))
	for i := len(s.sessions) - 1; i >= 0; i-- {
		out = append(out, toSummary(s.sessions[i]))
	}
	return out
}

func (s *Store) AllDetails() []SessionDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SessionDetail, 0, len(s.sessions))
	for _, session := range s.sessions {
		out = append(out, toDetail(session))
	}
	return out
}

func (s *Store) Detail(id string) (SessionDetail, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		if session.ID == id {
			return toDetail(session), true
		}
	}
	return SessionDetail{}, false
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

func toSummary(session *CapturedSession) SessionSummary {
	return SessionSummary{
		ID:              session.ID,
		StartedAt:       session.StartedAt.Format(time.RFC3339Nano),
		DurationMs:      session.Timing.DurationMs,
		Source:          session.Source,
		Method:          session.Method,
		Scheme:          session.Scheme,
		Host:            session.Host,
		Path:            session.Path,
		URL:             session.URL,
		StatusCode:      session.StatusCode,
		Status:          session.Status,
		Protocol:        session.Protocol,
		ContentType:     session.ContentType,
		ResponseSize:    session.ResponseSize,
		RequestSize:     session.RequestSize,
		InterceptedTLS:  session.InterceptedTLS,
		TunnelOnly:      session.TunnelOnly,
		Error:           session.Error,
		Rule:            session.Rule,
		ResponsePreview: preview(session.ResponseBody.Text),
	}
}

func toDetail(session *CapturedSession) SessionDetail {
	return SessionDetail{
		SessionSummary:  toSummary(session),
		RequestHeaders:  cloneHeaders(session.RequestHeaders),
		ResponseHeaders: cloneHeaders(session.ResponseHeaders),
		RequestBody:     session.RequestBody,
		ResponseBody:    session.ResponseBody,
		Timing:          session.Timing,
		Certificate:     session.Certificate,
	}
}

func cloneSession(session *CapturedSession) *CapturedSession {
	cp := *session
	cp.RequestHeaders = cloneHeaders(session.RequestHeaders)
	cp.ResponseHeaders = cloneHeaders(session.ResponseHeaders)
	return &cp
}

func cloneHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func preview(text string) string {
	const limit = 180
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
