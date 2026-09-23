package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"pyntra/internal/config"
)

const slackMaxLen = 3900

type slackEnvelope struct {
	Type       string          `json:"type"`
	EnvelopeID string          `json:"envelope_id,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type slackEventPayload struct {
	Event struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Channel string `json:"channel"`
		User    string `json:"user"`
		BotID   string `json:"bot_id"`
		SubType string `json:"subtype"`
	} `json:"event"`
}

// runSlack connects to Slack via Socket Mode and replies through the agent.
func runSlack(ctx context.Context, cfg config.SlackBotConfig, proc Processor, logger *zap.Logger) {
	client := &http.Client{Timeout: 30 * time.Second}
	sessions := newSessionStore()
	logger.Info("slack bot started")

	for {
		if ctx.Err() != nil {
			logger.Info("slack bot stopped")
			return
		}
		wssURL, err := slackOpenConnection(ctx, client, cfg.AppToken)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("slack connection open failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}
		if err := slackReadLoop(ctx, wssURL, cfg, proc, sessions, client, logger); err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("slack socket closed, reconnecting", zap.Error(err))
			time.Sleep(2 * time.Second)
		}
	}
}

func slackOpenConnection(ctx context.Context, client *http.Client, appToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/apps.connections.open", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+appToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !out.OK {
		return "", fmt.Errorf("apps.connections.open failed: %s", out.Error)
	}
	return out.URL, nil
}

func slackReadLoop(ctx context.Context, wssURL string, cfg config.SlackBotConfig, proc Processor, sessions *sessionStore, client *http.Client, logger *zap.Logger) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wssURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Close the socket when the context is cancelled so ReadMessage unblocks.
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var env slackEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		switch env.Type {
		case "hello":
			continue
		case "disconnect":
			return fmt.Errorf("slack requested disconnect")
		case "events_api":
			// Acknowledge immediately (required within 3s).
			if env.EnvelopeID != "" {
				ack, _ := json.Marshal(map[string]string{"envelope_id": env.EnvelopeID})
				_ = conn.WriteMessage(websocket.TextMessage, ack)
			}
			var p slackEventPayload
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				continue
			}
			ev := p.Event
			if ev.Type != "message" && ev.Type != "app_mention" {
				continue
			}
			// Ignore messages from bots (including our own) and edits/deletes.
			if ev.BotID != "" || ev.SubType != "" {
				continue
			}
			text := strings.TrimSpace(ev.Text)
			if text == "" || ev.Channel == "" {
				continue
			}
			key := "slack:" + ev.Channel + ":" + ev.User
			switch parseCommand(text) {
			case cmdNew:
				sessions.reset(key)
				slackPost(client, cfg.BotToken, ev.Channel, "🧹 Started a new conversation.", logger)
				continue
			case cmdHelp:
				slackPost(client, cfg.BotToken, ev.Channel, botGreeting, logger)
				continue
			}
			go func(channel, key, text string) {
				conv := sessions.get(key)
				reply, newConv, perr := proc.ProcessMessage(context.Background(), conv, text, cfg.Role)
				if perr != nil {
					logger.Warn("slack process failed", zap.Error(perr))
					slackPost(client, cfg.BotToken, channel, "⚠️ Error: "+perr.Error(), logger)
					return
				}
				sessions.set(key, newConv)
				slackPost(client, cfg.BotToken, channel, reply, logger)
			}(ev.Channel, key, text)
		}
	}
}

func slackPost(client *http.Client, botToken, channel, text string, logger *zap.Logger) {
	if strings.TrimSpace(text) == "" {
		text = "(empty response)"
	}
	for _, part := range chunk(text, slackMaxLen) {
		body, _ := json.Marshal(map[string]interface{}{"channel": channel, "text": part})
		req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Authorization", "Bearer "+botToken)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(req)
		if err != nil {
			logger.Warn("slack chat.postMessage failed", zap.Error(err))
			return
		}
		resp.Body.Close()
	}
}
