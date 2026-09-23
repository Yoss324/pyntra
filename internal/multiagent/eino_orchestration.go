package multiagent

import (
	"context"
	"fmt"
	"strings"

	"pyntra/internal/config"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// PlanExecuteRootArgs parameters required to build Eino adk/prebuilt/planexecute root Agent.
type PlanExecuteRootArgs struct {
	MainToolCallingModel *openai.ChatModel
	ExecModel            *openai.ChatModel
	OrchInstruction      string
	ToolsCfg             adk.ToolsConfig
	ExecMaxIter          int
	LoopMaxIter          int
	// When AppCfg / Logger is non-empty, mount Eino summarization middleware on Executor consistent with Deep/Supervisor.
	AppCfg *config.Config
	Logger *zap.Logger
}

// NewPlanExecuteRoot returns the plan → execute → replan preset orchestration root node (at same level as Deep / Supervisor).
func NewPlanExecuteRoot(ctx context.Context, a *PlanExecuteRootArgs) (adk.ResumableAgent, error) {
	if a == nil {
		return nil, fmt.Errorf("plan_execute: args is empty")
	}
	if a.MainToolCallingModel == nil || a.ExecModel == nil {
		return nil, fmt.Errorf("plan_execute: model is empty")
	}
	tcm, ok := interface{}(a.MainToolCallingModel).(model.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("plan_execute: main model must implement ToolCallingChatModel")
	}
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: tcm,
	})
	if err != nil {
		return nil, fmt.Errorf("plan_execute planner: %w", err)
	}
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel:  tcm,
		GenInputFn: planExecuteReplannerGenInput,
	})
	if err != nil {
		return nil, fmt.Errorf("plan_execute replanner: %w", err)
	}
	var execHandlers []adk.ChatModelAgentMiddleware
	if a.AppCfg != nil {
		sumMw, sumErr := newEinoSummarizationMiddleware(ctx, a.ExecModel, a.AppCfg, a.Logger)
		if sumErr != nil {
			return nil, fmt.Errorf("plan_execute executor summarization: %w", sumErr)
		}
		execHandlers = append(execHandlers, sumMw)
	}
	executor, err := newPlanExecuteExecutor(ctx, &planexecute.ExecutorConfig{
		Model:         a.ExecModel,
		ToolsConfig:   a.ToolsCfg,
		MaxIterations: a.ExecMaxIter,
		GenInputFn:    planExecuteExecutorGenInput(a.OrchInstruction),
	}, execHandlers)
	if err != nil {
		return nil, fmt.Errorf("plan_execute executor: %w", err)
	}
	loopMax := a.LoopMaxIter
	if loopMax <= 0 {
		loopMax = 10
	}
	return planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: loopMax,
	})
}

func planExecuteExecutorGenInput(orchInstruction string) planexecute.GenModelInputFn {
	oi := strings.TrimSpace(orchInstruction)
	return func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
		planContent, err := in.Plan.MarshalJSON()
		if err != nil {
			return nil, err
		}
		userMsgs, err := planexecute.ExecutorPrompt.Format(ctx, map[string]any{
			"input":          planExecuteFormatInput(in.UserInput),
			"plan":           string(planContent),
			"executed_steps": planExecuteFormatExecutedSteps(in.ExecutedSteps),
			"step":           in.Plan.FirstStep(),
		})
		if err != nil {
			return nil, err
		}
		if oi != "" {
			userMsgs = append([]adk.Message{schema.SystemMessage(oi)}, userMsgs...)
		}
		return userMsgs, nil
	}
}

func planExecuteFormatInput(input []adk.Message) string {
	var sb strings.Builder
	for _, msg := range input {
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

func planExecuteFormatExecutedSteps(results []planexecute.ExecutedStep) string {
	capped := capPlanExecuteExecutedSteps(results)
	var sb strings.Builder
	for _, result := range capped {
		sb.WriteString(fmt.Sprintf("Step: %s\nResult: %s\n\n", result.Step, result.Result))
	}
	return sb.String()
}

// planExecuteReplannerGenInput is consistent with Eino default Replanner input, but executed_steps are capped before being written to prompt.
func planExecuteReplannerGenInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planContent, err := in.Plan.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return planexecute.ReplannerPrompt.Format(ctx, map[string]any{
		"plan":           string(planContent),
		"input":          planExecuteFormatInput(in.UserInput),
		"executed_steps": planExecuteFormatExecutedSteps(in.ExecutedSteps),
		"plan_tool":      planexecute.PlanToolInfo.Name,
		"respond_tool":   planexecute.RespondToolInfo.Name,
	})
}

// planExecuteStreamsMainAssistant maps assistant streaming output from planning/execution/replanning stages to the main conversation area.
func planExecuteStreamsMainAssistant(agent string) bool {
	if agent == "" {
		return true
	}
	switch agent {
	case "planner", "executor", "replanner", "execute_replan", "plan_execute_replan":
		return true
	default:
		return false
	}
}

func planExecuteEinoRoleTag(agent string) string {
	_ = agent
	return "orchestrator"
}
