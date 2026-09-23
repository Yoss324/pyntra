package multiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pyntra/internal/agent"
	"pyntra/internal/config"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// einoSummarizeUserInstruction has the same goal as single Agent MemoryCompressor: compress conversation history while preserving penetration test key information.
const einoSummarizeUserInstruction = `Compress conversation history while keeping all critical security testing information intact.

Must retain: confirmed vulnerabilities and attack paths, core findings from tool output, credentials and authentication details, architecture and weaknesses, current progress, failed attempts and dead ends, strategic decisions.
Preserve exact technical details (URLs, paths, parameters, payloads, versions, error messages may be summarized but key points must not be lost).
Summarize lengthy scan output into conclusions; merge repeated findings.
Enumerated assets must retain **inheritable summary**: main domain, key subdomains/short host list (or count + representative samples), high-value targets and identified services/port key points, avoiding subsequent sub-agents being unable to see the list and repeating full enumeration.

Output must enable subsequent agents to seamlessly continue the same authorized penetration test task.`

// newEinoSummarizationMiddleware uses Eino ADK Summarization middleware (see https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_summarization/).
// Trigger threshold is consistent with single Agent MemoryCompressor: summarize when estimated tokens exceed 90% of openai.max_total_tokens.
func newEinoSummarizationMiddleware(
	ctx context.Context,
	summaryModel model.BaseChatModel,
	appCfg *config.Config,
	logger *zap.Logger,
) (adk.ChatModelAgentMiddleware, error) {
	if summaryModel == nil || appCfg == nil {
		return nil, fmt.Errorf("multiagent: summarization requires model and configuration")
	}
	maxTotal := appCfg.OpenAI.MaxTotalTokens
	if maxTotal <= 0 {
		maxTotal = 120000
	}
	trigger := int(float64(maxTotal) * 0.9)
	if trigger < 4096 {
		trigger = maxTotal
		if trigger < 4096 {
			trigger = 4096
		}
	}
	preserveMax := trigger / 3
	if preserveMax < 2048 {
		preserveMax = 2048
	}

	modelName := strings.TrimSpace(appCfg.OpenAI.Model)
	if modelName == "" {
		modelName = "gpt-4o"
	}

	mw, err := summarization.New(ctx, &summarization.Config{
		Model: summaryModel,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: trigger,
		},
		TokenCounter:       einoSummarizationTokenCounter(modelName),
		UserInstruction:    einoSummarizeUserInstruction,
		EmitInternalEvents: false,
		PreserveUserMessages: &summarization.PreserveUserMessages{
			Enabled:   true,
			MaxTokens: preserveMax,
		},
		Callback: func(ctx context.Context, before, after adk.ChatModelAgentState) error {
			if logger == nil {
				return nil
			}
			logger.Info("eino summarization compressed context",
				zap.Int("messages_before", len(before.Messages)),
				zap.Int("messages_after", len(after.Messages)),
				zap.Int("max_total_tokens", maxTotal),
				zap.Int("trigger_context_tokens", trigger),
			)
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("summarization.New: %w", err)
	}
	return mw, nil
}

func einoSummarizationTokenCounter(openAIModel string) summarization.TokenCounterFunc {
	tc := agent.NewTikTokenCounter()
	return func(ctx context.Context, input *summarization.TokenCounterInput) (int, error) {
		var sb strings.Builder
		for _, msg := range input.Messages {
			if msg == nil {
				continue
			}
			sb.WriteString(string(msg.Role))
			sb.WriteByte('\n')
			if msg.Content != "" {
				sb.WriteString(msg.Content)
				sb.WriteByte('\n')
			}
			if msg.ReasoningContent != "" {
				sb.WriteString(msg.ReasoningContent)
				sb.WriteByte('\n')
			}
			if len(msg.ToolCalls) > 0 {
				if b, err := json.Marshal(msg.ToolCalls); err == nil {
					sb.Write(b)
					sb.WriteByte('\n')
				}
			}
			for _, part := range msg.UserInputMultiContent {
				if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
					sb.WriteString(part.Text)
					sb.WriteByte('\n')
				}
			}
		}
		for _, tl := range input.Tools {
			if tl == nil {
				continue
			}
			cp := *tl
			cp.Extra = nil
			if text, err := json.Marshal(cp); err == nil {
				sb.Write(text)
				sb.WriteByte('\n')
			}
		}
		text := sb.String()
		n, err := tc.Count(openAIModel, text)
		if err != nil {
			return (len(text) + 3) / 4, nil
		}
		return n, nil
	}
}
