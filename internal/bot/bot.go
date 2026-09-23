// Package bot provides chat-bot integrations (Telegram, Slack, Discord) that let
// users talk to Pyntra from a messaging app. Each bot maintains a per-chat
// conversation so multi-turn context is preserved, and forwards messages to the
// agent through the Processor interface.
package bot

import (
	"context"
	"strings"
	"sync"

	"go.uber.org/zap"

	"pyntra/internal/config"
)

// Processor runs a user message through the agent and returns the assistant's
// reply together with the (possibly newly created) conversation ID.
type Processor interface {
	ProcessMessage(ctx context.Context, conversationID, message, role string) (string, string, error)
}

// Manager starts and stops the enabled chat bots.
type Manager struct {
	proc    Processor
	logger  *zap.Logger
	mu      sync.Mutex
	cancels []context.CancelFunc
}

// NewManager creates a bot manager bound to the given message processor.
func NewManager(proc Processor, logger *zap.Logger) *Manager {
	return &Manager{proc: proc, logger: logger}
}

// Start launches every enabled and fully configured bot in its own goroutine.
func (m *Manager) Start(cfg config.BotsConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Telegram.Enabled && strings.TrimSpace(cfg.Telegram.Token) != "" {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancels = append(m.cancels, cancel)
		go runTelegram(ctx, cfg.Telegram, m.proc, m.logger)
	}
	if cfg.Slack.Enabled && strings.TrimSpace(cfg.Slack.AppToken) != "" && strings.TrimSpace(cfg.Slack.BotToken) != "" {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancels = append(m.cancels, cancel)
		go runSlack(ctx, cfg.Slack, m.proc, m.logger)
	}
	if cfg.Discord.Enabled && strings.TrimSpace(cfg.Discord.Token) != "" {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancels = append(m.cancels, cancel)
		go runDiscord(ctx, cfg.Discord, m.proc, m.logger)
	}
}

// Stop cancels all running bots.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cancels {
		c()
	}
	m.cancels = nil
}

// sessionStore maps a platform chat key -> conversation ID so each chat keeps
// its own multi-turn context.
type sessionStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newSessionStore() *sessionStore { return &sessionStore{m: make(map[string]string)} }

func (s *sessionStore) get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key]
}

func (s *sessionStore) set(key, conv string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = conv
}

func (s *sessionStore) reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

const botGreeting = "👋 Pyntra security assistant. Send a message to start. Commands: /new resets the conversation, /help shows this."

type botCmd int

const (
	cmdNone botCmd = iota
	cmdNew
	cmdHelp
)

// parseCommand recognizes simple control commands at the start of a message.
func parseCommand(text string) botCmd {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 0 {
		return cmdNone
	}
	switch fields[0] {
	case "/new", "/reset":
		return cmdNew
	case "/help", "/start":
		return cmdHelp
	}
	return cmdNone
}

// chunk splits a reply into pieces no larger than max runes, respecting the
// platform message-size limits (Telegram 4096, Discord 2000, Slack ~3900).
func chunk(s string, max int) []string {
	if max <= 0 {
		return []string{s}
	}
	r := []rune(s)
	if len(r) <= max {
		if len(r) == 0 {
			return []string{""}
		}
		return []string{s}
	}
	var out []string
	for len(r) > 0 {
		n := max
		if n > len(r) {
			n = len(r)
		}
		out = append(out, string(r[:n]))
		r = r[n:]
	}
	return out
}
