package bot

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"

	"pyntra/internal/config"
)

const discordMaxLen = 2000

// runDiscord opens a Discord gateway connection and replies through the agent.
func runDiscord(ctx context.Context, cfg config.DiscordBotConfig, proc Processor, logger *zap.Logger) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		logger.Warn("discord session create failed", zap.Error(err))
		return
	}
	sessions := newSessionStore()
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			return
		}
		key := "discord:" + m.ChannelID + ":" + m.Author.ID
		switch parseCommand(text) {
		case cmdNew:
			sessions.reset(key)
			discordSend(s, m.ChannelID, "🧹 Started a new conversation.", logger)
			return
		case cmdHelp:
			discordSend(s, m.ChannelID, botGreeting, logger)
			return
		}
		go func() {
			conv := sessions.get(key)
			reply, newConv, perr := proc.ProcessMessage(context.Background(), conv, text, cfg.Role)
			if perr != nil {
				logger.Warn("discord process failed", zap.Error(perr))
				discordSend(s, m.ChannelID, "⚠️ Error: "+perr.Error(), logger)
				return
			}
			sessions.set(key, newConv)
			discordSend(s, m.ChannelID, reply, logger)
		}()
	})

	if err := session.Open(); err != nil {
		logger.Warn("discord open failed", zap.Error(err))
		return
	}
	logger.Info("discord bot started")

	<-ctx.Done()
	_ = session.Close()
	logger.Info("discord bot stopped")
}

func discordSend(s *discordgo.Session, channelID, text string, logger *zap.Logger) {
	if strings.TrimSpace(text) == "" {
		text = "(empty response)"
	}
	for _, part := range chunk(text, discordMaxLen) {
		if _, err := s.ChannelMessageSend(channelID, part); err != nil {
			logger.Warn("discord send failed", zap.Error(err))
			return
		}
	}
}
