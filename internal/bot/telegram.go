package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"pyntra/internal/config"
)

const telegramMaxLen = 4096

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Text string `json:"text"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type tgResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// runTelegram polls the Telegram Bot API for updates and replies via the agent.
func runTelegram(ctx context.Context, cfg config.TelegramBotConfig, proc Processor, logger *zap.Logger) {
	base := "https://api.telegram.org/bot" + cfg.Token
	client := &http.Client{Timeout: 70 * time.Second}
	sessions := newSessionStore()
	var offset int64

	logger.Info("telegram bot started")
	for {
		if ctx.Err() != nil {
			logger.Info("telegram bot stopped")
			return
		}
		updates, err := tgGetUpdates(ctx, client, base, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("telegram getUpdates failed", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message == nil || strings.TrimSpace(u.Message.Text) == "" {
				continue
			}
			chatID := u.Message.Chat.ID
			text := strings.TrimSpace(u.Message.Text)
			key := "tg:" + strconv.FormatInt(chatID, 10)

			switch parseCommand(text) {
			case cmdNew:
				sessions.reset(key)
				tgSend(client, base, chatID, "🧹 Started a new conversation.", logger)
				continue
			case cmdHelp:
				tgSend(client, base, chatID, botGreeting, logger)
				continue
			}

			go func(chatID int64, key, text string) {
				conv := sessions.get(key)
				reply, newConv, perr := proc.ProcessMessage(context.Background(), conv, text, cfg.Role)
				if perr != nil {
					logger.Warn("telegram process failed", zap.Error(perr))
					tgSend(client, base, chatID, "⚠️ Error: "+perr.Error(), logger)
					return
				}
				sessions.set(key, newConv)
				tgSend(client, base, chatID, reply, logger)
			}(chatID, key, text)
		}
	}
}

func tgGetUpdates(ctx context.Context, client *http.Client, base string, offset int64) ([]tgUpdate, error) {
	url := fmt.Sprintf("%s/getUpdates?timeout=60&offset=%d", base, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out tgResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram API returned ok=false")
	}
	return out.Result, nil
}

func tgSend(client *http.Client, base string, chatID int64, text string, logger *zap.Logger) {
	if strings.TrimSpace(text) == "" {
		text = "(empty response)"
	}
	for _, part := range chunk(text, telegramMaxLen) {
		body, _ := json.Marshal(map[string]interface{}{
			"chat_id": chatID,
			"text":    part,
		})
		req, err := http.NewRequest(http.MethodPost, base+"/sendMessage", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			logger.Warn("telegram sendMessage failed", zap.Error(err))
			return
		}
		resp.Body.Close()
	}
}
