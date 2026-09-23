// Package multiagent uses CloudWeGo Eino adk/prebuilt (deep / plan_execute / supervisor) to orchestrate multi-agents; MCP tools are bridged to existing Agent via einomcp.
package multiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"pyntra/internal/agent"
	"pyntra/internal/agents"
	"pyntra/internal/config"
	"pyntra/internal/einomcp"
	"pyntra/internal/openai"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// RunResult aligns fields with single Agent loop results for convenient reuse of storage and SSE completion logic.
type RunResult struct {
	Response string
	MCPExecutionIDs []string
	LastReActInput string
	LastReActOutput string
}

// toolCallPendingInfo tracks a tool_call emitted to the UI so we can later
// correlate tool_result events (even when the framework omits ToolCallID) and
// avoid leaving the UI stuck in "running" state on recoverable errors.
type toolCallPendingInfo struct {
	ToolCallID string
	ToolName string
	EinoAgent string
	EinoRole string
}

// RunDeepAgent executes one round of conversation using Eino multi-agent preset orchestration (deep / plan_execute / supervisor; streaming events output via progress callback).
// When orchestrationOverride is non-empty, it takes priority (e.g., from chat/WebShell request body); otherwise uses multi_agent.orchestration (legacy yaml); if both empty, defaults to deep.
func RunDeepAgent(
	ctx context.Context,
	appCfg *config.Config,
	ma *config.MultiAgentConfig,
	ag *agent.Agent,
	logger *zap.Logger,
	conversationID string,
	userMessage string,
	history []agent.ChatMessage,
	roleTools []string,
	progress func(eventType, message string, data interface{}),
	agentsMarkdownDir string,
	orchestrationOverride string,
) (*RunResult, error) {
	if appCfg == nil || ma == nil || ag == nil {
		return nil, fmt.Errorf("multiagent: config or Agent is empty")
	}

	effectiveSubs := ma.SubAgents
	var markdownLoad *agents.MarkdownDirLoad
	var orch *agents.OrchestratorMarkdown
	if strings.TrimSpace(agentsMarkdownDir) != "" {
		load, merr := agents.LoadMarkdownAgentsDir(agentsMarkdownDir)
		if merr != nil {
			if logger != nil {
				logger.Warn("failed to load agents directory Markdown, will use sub_agents from config", zap.Error(merr))
			}
		} else {
			markdownLoad = load
			effectiveSubs = agents.MergeYAMLAndMarkdown(ma.SubAgents, load.SubAgents)
			orch = load.Orchestrator
		}
	}
	orchMode := config.NormalizeMultiAgentOrchestration(ma.Orchestration)
	if o := strings.TrimSpace(orchestrationOverride); o != "" {
		orchMode = config.NormalizeMultiAgentOrchestration(o)
	}
	if orchMode != "plan_execute" && ma.WithoutGeneralSubAgent && len(effectiveSubs) == 0 {
		return nil, fmt.Errorf("when multi_agent.without_general_sub_agent is true, must configure at least one sub-agent in multi_agent.sub_agents or agents directory Markdown")
	}
	if orchMode == "supervisor" && len(effectiveSubs) == 0 {
		return nil, fmt.Errorf("when multi_agent.orchestration=supervisor, must configure at least one sub-agent (sub_agents or agents directory Markdown)")
	}

	einoLoc, einoSkillMW, einoFSTools, skillsRoot, einoErr := prepareEinoSkills(ctx, appCfg.SkillsDir, ma, logger)
	if einoErr != nil {
		return nil, einoErr
	}

	holder := &einomcp.ConversationHolder{}
	holder.Set(conversationID)

	var mcpIDsMu sync.Mutex
	var mcpIDs []string
	recorder := func(id string) {
		if id == "" {
			return
		}
		mcpIDsMu.Lock()
		mcpIDs = append(mcpIDs, id)
		mcpIDsMu.Unlock()
	}

	// Consistent with single-agent streaming: include current mcpExecutionIds in data of response_start / response_delta for main chat binding copy and display.
	snapshotMCPIDs := func() []string {
		mcpIDsMu.Lock()
		defer mcpIDsMu.Unlock()
		out := make([]string, len(mcpIDs))
		copy(out, mcpIDs)
		return out
	}

	mainDefs := ag.ToolsForRole(roleTools)
	toolOutputChunk := func(toolName, toolCallID, chunk string) {
		// When toolCallId is missing, frontend ignores tool_result_delta.
		if progress == nil || toolCallID == "" {
			return
		}
		progress("tool_result_delta", chunk, map[string]interface{}{
			"toolName": toolName,
			"toolCallId": toolCallID,
			// index/total/iteration are optional for UI; we don't know them in this bridge.
			"index": 0,
			"total": 0,
			"iteration": 0,
			"source": "eino",
		})
	}

	mainTools, err := einomcp.ToolsFromDefinitions(ag, holder, mainDefs, recorder, toolOutputChunk)
	if err != nil {
		return nil, err
	}

	mainToolsForCfg, mainOrchestratorPre, err := prependEinoMiddlewares(ctx, &ma.EinoMiddleware, einoMWMain, mainTools, einoLoc, skillsRoot, conversationID, logger)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Minute,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 300 * time.Second,
				KeepAlive: 300 * time.Second,
			}).DialContext,
			MaxIdleConns: 100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout: 90 * time.Second,
			TLSHandshakeTimeout: 30 * time.Second,
			ResponseHeaderTimeout: 60 * time.Minute,
		},
	}

	// If configured as Claude provider, inject automatic bridge transport for transparent Anthropic Messages API access to Eino
	httpClient = openai.NewEinoHTTPClient(&appCfg.OpenAI, httpClient)

	baseModelCfg := &einoopenai.ChatModelConfig{
		APIKey: appCfg.OpenAI.APIKey,
		BaseURL: strings.TrimSuffix(appCfg.OpenAI.BaseURL, "/"),
		Model: appCfg.OpenAI.Model,
		HTTPClient: httpClient,
	}

	deepMaxIter := ma.MaxIteration
	if deepMaxIter <= 0 {
		deepMaxIter = appCfg.Agent.MaxIterations
	}
	if deepMaxIter <= 0 {
		deepMaxIter = 40
	}

	subDefaultIter := ma.SubAgentMaxIterations
	if subDefaultIter <= 0 {
		subDefaultIter = 20
	}

	var subAgents []adk.Agent
	if orchMode != "plan_execute" {
		subAgents = make([]adk.Agent, 0, len(effectiveSubs))
		for _, sub := range effectiveSubs {
			id := strings.TrimSpace(sub.ID)
			if id == "" {
				return nil, fmt.Errorf("multi_agent.sub_agents contains empty id")
			}
			name := strings.TrimSpace(sub.Name)
			if name == "" {
				name = id
			}
			desc := strings.TrimSpace(sub.Description)
			if desc == "" {
				desc = fmt.Sprintf("Specialist agent %s for penetration testing workflow.", id)
			}
			instr := strings.TrimSpace(sub.Instruction)
			if instr == "" {
				instr = "You are a specialist sub-agent in Pyntra, assisting to complete sub-tasks delegated by users in authorized penetration testing scenarios. Prioritize using available tools to gather evidence, provide concise and professional answers."
			}

			roleTools := sub.RoleTools
			bind := strings.TrimSpace(sub.BindRole)
			if bind != "" && appCfg.Roles != nil {
				if r, ok := appCfg.Roles[bind]; ok && r.Enabled {
					if len(roleTools) == 0 && len(r.Tools) > 0 {
						roleTools = r.Tools
					}
				if len(r.Skills) > 0 {
					var b strings.Builder
					b.WriteString(instr)
					b.WriteString("\n\nThis role recommends prioritizing skill packages loaded via Eino `skill` tool (progressive disclosure) with names: ")
					for i, s := range r.Skills {
						if i > 0 {
							b.WriteString(", ")
						}
						b.WriteString(s)
					}
					b.WriteString(".")
					instr = b.String()
				}
				}
			}

			subModel, err := einoopenai.NewChatModel(ctx, baseModelCfg)
			if err != nil {
				return nil, fmt.Errorf(" %q ChatModel: %w", id, err)
			}

			subDefs := ag.ToolsForRole(roleTools)
			subTools, err := einomcp.ToolsFromDefinitions(ag, holder, subDefs, recorder, toolOutputChunk)
			if err != nil {
				return nil, fmt.Errorf(" %q tool: %w", id, err)
			}

			subToolsForCfg, subPre, err := prependEinoMiddlewares(ctx, &ma.EinoMiddleware, einoMWSub, subTools, einoLoc, skillsRoot, conversationID, logger)
			if err != nil {
				return nil, fmt.Errorf(" %q eino : %w", id, err)
			}

			subMax := sub.MaxIterations
			if subMax <= 0 {
				subMax = subDefaultIter
			}

			subSumMw, err := newEinoSummarizationMiddleware(ctx, subModel, appCfg, logger)
			if err != nil {
				return nil, fmt.Errorf(" %q summarization : %w", id, err)
			}

			var subHandlers []adk.ChatModelAgentMiddleware
			if len(subPre) > 0 {
				subHandlers = append(subHandlers, subPre...)
			}
			if einoSkillMW != nil {
				if einoFSTools && einoLoc != nil {
					subFs, fsErr := subAgentFilesystemMiddleware(ctx, einoLoc)
					if fsErr != nil {
						return nil, fmt.Errorf(" %q filesystem : %w", id, fsErr)
					}
					subHandlers = append(subHandlers, subFs)
				}
				subHandlers = append(subHandlers, einoSkillMW)
			}
			subHandlers = append(subHandlers, subSumMw)

			sa, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
				Name: id,
				Description: desc,
				Instruction: instr,
				Model: subModel,
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: subToolsForCfg,
						UnknownToolsHandler: einomcp.UnknownToolReminderHandler(),
						ToolCallMiddlewares: []compose.ToolMiddleware{
							{Invokable: softRecoveryToolCallMiddleware()},
						},
					},
					EmitInternalEvents: true,
				},
				MaxIterations: subMax,
				Handlers: subHandlers,
			})
			if err != nil {
				return nil, fmt.Errorf(" %q: %w", id, err)
			}
			subAgents = append(subAgents, sa)
		}
	}

	mainModel, err := einoopenai.NewChatModel(ctx, baseModelCfg)
	if err != nil {
		return nil, fmt.Errorf("multi-agentmodel: %w", err)
	}

	mainSumMw, err := newEinoSummarizationMiddleware(ctx, mainModel, appCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("multi-agent summarization : %w", err)
	}
	orchestratorName := "cyberstrike-deep"
	orchDescription := "Coordinates specialist agents and MCP tools for authorized security testing."
	orchInstruction, orchMeta := resolveMainOrchestratorInstruction(orchMode, ma, markdownLoad)
	if orchMeta != nil {
		if strings.TrimSpace(orchMeta.EinoName) != "" {
			orchestratorName = strings.TrimSpace(orchMeta.EinoName)
		}
		if d := strings.TrimSpace(orchMeta.Description); d != "" {
			orchDescription = d
		}
	} else if orchMode == "deep" && orch != nil {
		if strings.TrimSpace(orch.EinoName) != "" {
			orchestratorName = strings.TrimSpace(orch.EinoName)
		}
		if d := strings.TrimSpace(orch.Description); d != "" {
			orchDescription = d
		}
	}

	supInstr := strings.TrimSpace(orchInstruction)
	if orchMode == "supervisor" {
		var sb strings.Builder
		if supInstr != "" {
			sb.WriteString(supInstr)
			sb.WriteString("\n\n")
		}
		sb.WriteString(":Tasks transfer tool( Agent name).:")
		for _, sa := range subAgents {
			sb.WriteString("\n- ")
			sb.WriteString(sa.Name(ctx))
		}
		sb.WriteString("\n\nCompleted, exit tool.")
		supInstr = sb.String()
	}

	var deepBackend filesystem.Backend
	var deepShell filesystem.StreamingShell
	if einoLoc != nil && einoFSTools {
		deepBackend = einoLoc
		deepShell = einoLoc
	}

	deepHandlers := []adk.ChatModelAgentMiddleware{}
	if len(mainOrchestratorPre) > 0 {
		deepHandlers = append(deepHandlers, mainOrchestratorPre...)
	}
	if einoSkillMW != nil {
		deepHandlers = append(deepHandlers, einoSkillMW)
	}
	deepHandlers = append(deepHandlers, newNoNestedTaskMiddleware(), mainSumMw)

	supHandlers := []adk.ChatModelAgentMiddleware{}
	if len(mainOrchestratorPre) > 0 {
		supHandlers = append(supHandlers, mainOrchestratorPre...)
	}
	if einoSkillMW != nil {
		supHandlers = append(supHandlers, einoSkillMW)
	}
	supHandlers = append(supHandlers, mainSumMw)

	mainToolsCfg := adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: mainToolsForCfg,
			UnknownToolsHandler: einomcp.UnknownToolReminderHandler(),
			ToolCallMiddlewares: []compose.ToolMiddleware{
				{Invokable: softRecoveryToolCallMiddleware()},
			},
		},
		EmitInternalEvents: true,
	}

	deepOutKey, modelRetry, taskGen := deepExtrasFromConfig(ma)

	var da adk.Agent
	switch orchMode {
	case "plan_execute":
		execModel, perr := einoopenai.NewChatModel(ctx, baseModelCfg)
		if perr != nil {
			return nil, fmt.Errorf("plan_execute model: %w", perr)
		}
		peRoot, perr := NewPlanExecuteRoot(ctx, &PlanExecuteRootArgs{
			MainToolCallingModel: mainModel,
			ExecModel: execModel,
			OrchInstruction: orchInstruction,
			ToolsCfg: mainToolsCfg,
			ExecMaxIter: deepMaxIter,
			LoopMaxIter: ma.PlanExecuteLoopMaxIterations,
			AppCfg: appCfg,
			Logger: logger,
		})
		if perr != nil {
			return nil, perr
		}
		da = peRoot
	case "supervisor":
		supCfg := &adk.ChatModelAgentConfig{
			Name: orchestratorName,
			Description: orchDescription,
			Instruction: supInstr,
			Model: mainModel,
			ToolsConfig: mainToolsCfg,
			MaxIterations: deepMaxIter,
			Handlers: supHandlers,
			Exit: &adk.ExitTool{},
		}
		if modelRetry != nil {
			supCfg.ModelRetryConfig = modelRetry
		}
		if deepOutKey != "" {
			supCfg.OutputKey = deepOutKey
		}
		superChat, serr := adk.NewChatModelAgent(ctx, supCfg)
		if serr != nil {
			return nil, fmt.Errorf("supervisor : %w", serr)
		}
		supRoot, serr := supervisor.New(ctx, &supervisor.Config{
			Supervisor: superChat,
			SubAgents: subAgents,
		})
		if serr != nil {
			return nil, fmt.Errorf("supervisor.New: %w", serr)
		}
		da = supRoot
	default:
		dcfg := &deep.Config{
			Name: orchestratorName,
			Description: orchDescription,
			ChatModel: mainModel,
			Instruction: orchInstruction,
			SubAgents: subAgents,
			WithoutGeneralSubAgent: ma.WithoutGeneralSubAgent,
			WithoutWriteTodos: ma.WithoutWriteTodos,
			MaxIteration: deepMaxIter,
			Backend: deepBackend,
			StreamingShell: deepShell,
			Handlers: deepHandlers,
			ToolsConfig: mainToolsCfg,
		}
		if deepOutKey != "" {
			dcfg.OutputKey = deepOutKey
		}
		if modelRetry != nil {
			dcfg.ModelRetryConfig = modelRetry
		}
		if taskGen != nil {
			dcfg.TaskToolDescriptionGenerator = taskGen
		}
		dDeep, derr := deep.New(ctx, dcfg)
		if derr != nil {
			return nil, fmt.Errorf("deep.New: %w", derr)
		}
		da = dDeep
	}

	baseMsgs := historyToMessages(history)
	baseMsgs = append(baseMsgs, schema.UserMessage(userMessage))

	streamsMainAssistant := func(agent string) bool {
		if orchMode == "plan_execute" {
			return planExecuteStreamsMainAssistant(agent)
		}
		return agent == "" || agent == orchestratorName
	}
	einoRoleTag := func(agent string) string {
		if orchMode == "plan_execute" {
			return planExecuteEinoRoleTag(agent)
		}
		if streamsMainAssistant(agent) {
			return "orchestrator"
		}
		return "sub"
	}

	return runEinoADKAgentLoop(ctx, &einoADKRunLoopArgs{
		OrchMode: orchMode,
		OrchestratorName: orchestratorName,
		ConversationID: conversationID,
		Progress: progress,
		Logger: logger,
		SnapshotMCPIDs: snapshotMCPIDs,
		StreamsMainAssistant: streamsMainAssistant,
		EinoRoleTag: einoRoleTag,
		CheckpointDir: ma.EinoMiddleware.CheckpointDir,
		McpIDsMu: &mcpIDsMu,
		McpIDs: &mcpIDs,
		DA: da,
		EmptyResponseMessage: "(Eino multi-agentCompleted,output.process detailslogs.)",
	}, baseMsgs)
}

func historyToMessages(history []agent.ChatMessage) []adk.Message {
	if len(history) == 0 {
		return nil
	}
	const maxHistoryMessages = 300
	start := 0
	if len(history) > maxHistoryMessages {
		start = len(history) - maxHistoryMessages
	}
	out := make([]adk.Message, 0, len(history[start:]))
	for _, h := range history[start:] {
		switch h.Role {
		case "user":
			if strings.TrimSpace(h.Content) != "" {
				out = append(out, schema.UserMessage(h.Content))
			}
		case "assistant":
			if strings.TrimSpace(h.Content) == "" && len(h.ToolCalls) > 0 {
				continue
			}
			if strings.TrimSpace(h.Content) != "" {
				out = append(out, schema.AssistantMessage(h.Content, nil))
			}
		default:
			continue
		}
	}
	return out
}
func mergeStreamingToolCallFragments(fragments []schema.ToolCall) []schema.ToolCall {
	if len(fragments) == 0 {
		return nil
	}
	m, err := schema.ConcatMessages([]*schema.Message{{ToolCalls: fragments}})
	if err != nil || m == nil {
		return fragments
	}
	return m.ToolCalls
}
func mergeMessageToolCalls(msg *schema.Message) *schema.Message {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return msg
	}
	m, err := schema.ConcatMessages([]*schema.Message{msg})
	if err != nil || m == nil {
		return msg
	}
	out := *msg
	out.ToolCalls = m.ToolCalls
	return &out
}
func toolCallStableID(tc schema.ToolCall) string {
	if tc.ID != "" {
		return tc.ID
	}
	if tc.Index != nil {
		return fmt.Sprintf("idx:%d", *tc.Index)
	}
	return ""
}
func toolCallDisplayName(tc schema.ToolCall) string {
	if n := strings.TrimSpace(tc.Function.Name); n != "" {
		return n
	}
	if n := strings.TrimSpace(tc.Type); n != "" && !strings.EqualFold(n, "function") {
		return n
	}
	return "task"
}
func toolCallsSignatureFlush(msg *schema.Message) string {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return ""
	}
	parts := make([]string, 0, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		id := toolCallStableID(tc)
		if id == "" {
			id = fmt.Sprintf("pos:%d", i)
		}
		parts = append(parts, id+"|"+toolCallDisplayName(tc))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}
func toolCallsRichSignature(msg *schema.Message) string {
	base := toolCallsSignatureFlush(msg)
	if base == "" {
		return ""
	}
	parts := make([]string, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		id := toolCallStableID(tc)
		arg := tc.Function.Arguments
		if len(arg) > 240 {
			arg = arg[:240]
		}
		parts = append(parts, id+":"+arg)
	}
	sort.Strings(parts)
	return base + "|" + strings.Join(parts, ";")
}

func tryEmitToolCallsOnce(
	msg *schema.Message,
	agentName, orchestratorName, conversationID string,
	progress func(string, string, interface{}),
	seen map[string]struct{},
	subAgentToolStep map[string]int,
	markPending func(toolCallPendingInfo),
) {
	if msg == nil || len(msg.ToolCalls) == 0 || progress == nil || seen == nil {
		return
	}
	if toolCallsSignatureFlush(msg) == "" {
		return
	}
	sig := agentName + "\x1e" + toolCallsRichSignature(msg)
	if _, ok := seen[sig]; ok {
		return
	}
	seen[sig] = struct{}{}
	emitToolCallsFromMessage(msg, agentName, orchestratorName, conversationID, progress, subAgentToolStep, markPending)
}

func emitToolCallsFromMessage(
	msg *schema.Message,
	agentName, orchestratorName, conversationID string,
	progress func(string, string, interface{}),
	subAgentToolStep map[string]int,
	markPending func(toolCallPendingInfo),
) {
	if msg == nil || len(msg.ToolCalls) == 0 || progress == nil {
		return
	}
	if subAgentToolStep == nil {
		subAgentToolStep = make(map[string]int)
	}
	isSubToolRound := agentName != "" && agentName != orchestratorName
	if isSubToolRound {
		subAgentToolStep[agentName]++
		n := subAgentToolStep[agentName]
		progress("iteration", "", map[string]interface{}{
			"iteration": n,
			"einoScope": "sub",
			"einoRole": "sub",
			"einoAgent": agentName,
			"conversationId": conversationID,
			"source": "eino",
		})
	}
	role := "orchestrator"
	if isSubToolRound {
		role = "sub"
	}
	progress("tool_calls_detected", fmt.Sprintf(" %d toolcall", len(msg.ToolCalls)), map[string]interface{}{
		"count": len(msg.ToolCalls),
		"conversationId": conversationID,
		"source": "eino",
		"einoAgent": agentName,
		"einoRole": role,
	})
	for idx, tc := range msg.ToolCalls {
		argStr := strings.TrimSpace(tc.Function.Arguments)
		if argStr == "" && len(tc.Extra) > 0 {
			if b, mErr := json.Marshal(tc.Extra); mErr == nil {
				argStr = string(b)
			}
		}
		var argsObj map[string]interface{}
		if argStr != "" {
			if uErr := json.Unmarshal([]byte(argStr), &argsObj); uErr != nil || argsObj == nil {
				argsObj = map[string]interface{}{"_raw": argStr}
			}
		}
		display := toolCallDisplayName(tc)
		toolCallID := tc.ID
		if toolCallID == "" && tc.Index != nil {
			toolCallID = fmt.Sprintf("eino-stream-%d", *tc.Index)
		}
		// Record pending tool calls for later tool_result correlation / recovery flushing.
		// We intentionally record even for unknown tools to avoid "running" badge getting stuck.
		if markPending != nil && toolCallID != "" {
			markPending(toolCallPendingInfo{
				ToolCallID: toolCallID,
				ToolName: display,
				EinoAgent: agentName,
				EinoRole: role,
			})
		}
		progress("tool_call", fmt.Sprintf("calltool: %s", display), map[string]interface{}{
			"toolName": display,
			"arguments": argStr,
			"argumentsObj": argsObj,
			"toolCallId": toolCallID,
			"index": idx + 1,
			"total": len(msg.ToolCalls),
			"conversationId": conversationID,
			"source": "eino",
			"einoAgent": agentName,
			"einoRole": role,
		})
	}
}
func dedupeRepeatedParagraphs(s string, minLen int) string {
	if s == "" || minLen <= 0 {
		return s
	}
	paras := strings.Split(s, "\n\n")
	var out []string
	seen := make(map[string]bool)
	for _, p := range paras {
		t := strings.TrimSpace(p)
		if len(t) < minLen {
			out = append(out, p)
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, p)
	}
	return strings.TrimSpace(strings.Join(out, "\n\n"))
}
func dedupeParagraphsByLineFingerprint(s string, minParaLen int) string {
	if s == "" || minParaLen <= 0 {
		return s
	}
	paras := strings.Split(s, "\n\n")
	var out []string
	seen := make(map[string]bool)
	for _, p := range paras {
		t := strings.TrimSpace(p)
		if len(t) < minParaLen {
			out = append(out, p)
			continue
		}
		fp := paragraphLineFingerprint(t)
		if fp == "" {
			out = append(out, p)
			continue
		}
		if seen[fp] {
			continue
		}
		seen[fp] = true
		out = append(out, p)
	}
	return strings.TrimSpace(strings.Join(out, "\n\n"))
}

func paragraphLineFingerprint(t string) string {
	lines := strings.Split(t, "\n")
	norm := make([]string, 0, len(lines))
	for _, L := range lines {
		s := strings.TrimSpace(L)
		if s == "" {
			continue
		}
		norm = append(norm, s)
	}
	if len(norm) < 4 {
		return ""
	}
	sort.Strings(norm)
	return strings.Join(norm, "\x1e")
}
