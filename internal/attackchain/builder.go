package attackchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pyntra/internal/agent"
	"pyntra/internal/config"
	"pyntra/internal/database"
	"pyntra/internal/openai"

	"github.com/google/uuid"
	"go.uber.org/zap"
)
type Builder struct {
	db *database.DB
	logger *zap.Logger
	openAIClient *openai.Client
	openAIConfig *config.OpenAIConfig
	tokenCounter agent.TokenCounter
	maxTokens int
}
type Node = database.AttackChainNode
type Edge = database.AttackChainEdge
type Chain struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
func NewBuilder(db *database.DB, openAIConfig *config.OpenAIConfig, logger *zap.Logger) *Builder {
	transport := &http.Transport{
		MaxIdleConns: 100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout: 90 * time.Second,
	}
	httpClient := &http.Client{Timeout: 5 * time.Minute, Transport: transport}
	maxTokens := 0
	if openAIConfig != nil && openAIConfig.MaxTotalTokens > 0 {
		maxTokens = openAIConfig.MaxTotalTokens
	} else if openAIConfig != nil {
		model := strings.ToLower(openAIConfig.Model)
		if strings.Contains(model, "gpt-4") {
			maxTokens = 128000
		} else if strings.Contains(model, "gpt-3.5") {
			maxTokens = 16000
		} else if strings.Contains(model, "deepseek") {
			maxTokens = 131072
		} else {
			maxTokens = 100000
		}
	} else {
		maxTokens = 100000
	}

	return &Builder{
		db: db,
		logger: logger,
		openAIClient: openai.NewClient(openAIConfig, httpClient, logger),
		openAIConfig: openAIConfig,
		tokenCounter: agent.NewTikTokenCounter(),
		maxTokens: maxTokens,
	}
}
func (b *Builder) BuildChainFromConversation(ctx context.Context, conversationID string) (*Chain, error) {
	b.logger.Info("startbuildattack chain(simplified version)", zap.String("conversationId", conversationID))
	messages, err := b.db.GetMessages(conversationID)
	if err != nil {
		return nil, fmt.Errorf("fetchChatmessagefailed: %w", err)
	}

	if len(messages) == 0 {
		b.logger.Info("Chatno", zap.String("conversationId", conversationID))
		return &Chain{Nodes: []Node{}, Edges: []Edge{}}, nil
	}
	hasToolExecutions := false
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "assistant") {
			if len(messages[i].MCPExecutionIDs) > 0 {
				hasToolExecutions = true
				break
			}
		}
	}
	if !hasToolExecutions {
		if pdOK, err := b.db.ConversationHasToolProcessDetails(conversationID); err != nil {
			b.logger.Warn("queryprocess detailstoolfailed", zap.Error(err))
		} else if pdOK {
			hasToolExecutions = true
		}
	}
	taskCancelled := false
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "assistant") {
			content := strings.ToLower(messages[i].Content)
			if strings.Contains(content, "cancel") || strings.Contains(content, "cancelled") {
				taskCancelled = true
			}
			break
		}
	}
	if taskCancelled && !hasToolExecutions {
		b.logger.Info("Taskscancelnotool,backattack chain",
			zap.String("conversationId", conversationID),
			zap.Bool("taskCancelled", taskCancelled),
			zap.Bool("hasToolExecutions", hasToolExecutions))
		return &Chain{Nodes: []Node{}, Edges: []Edge{}}, nil
	}
	if !hasToolExecutions {
		b.logger.Info("notoolexecution record,backattack chain",
			zap.String("conversationId", conversationID))
		return &Chain{Nodes: []Node{}, Edges: []Edge{}}, nil
	}
	reactInputJSON, modelOutput, err := b.db.GetReActData(conversationID)
	if err != nil {
		b.logger.Warn("fetchsaveReActfailed,message historybuild", zap.Error(err))
		reactInputJSON = ""
		modelOutput = ""
	}

	// var userInput string
	var reactInputFinal string
	var dataSource string
	if reactInputJSON != "" && modelOutput != "" {
		hash := sha256.Sum256([]byte(reactInputJSON))
		reactInputHash := hex.EncodeToString(hash[:])[:16]
		var messageCount int
		var tempMessages []interface{}
		if json.Unmarshal([]byte(reactInputJSON), &tempMessages) == nil {
			messageCount = len(tempMessages)
		}

		dataSource = "database_last_react_input"
		b.logger.Info("saveReActbuildattack chain",
			zap.String("conversationId", conversationID),
			zap.String("dataSource", dataSource),
			zap.Int("reactInputSize", len(reactInputJSON)),
			zap.Int("messageCount", messageCount),
			zap.String("reactInputHash", reactInputHash),
			zap.Int("modelOutputSize", len(modelOutput)))
		// userInput = b.extractUserInputFromReActInput(reactInputJSON)
		reactInputFinal = b.formatReActInputFromJSON(reactInputJSON)
	} else {
		dataSource = "messages_table"
		b.logger.Info("message historybuildReAct",
			zap.String("conversationId", conversationID),
			zap.String("dataSource", dataSource),
			zap.Int("messageCount", len(messages)))
		for i := len(messages) - 1; i >= 0; i-- {
			if strings.EqualFold(messages[i].Role, "user") {
				// userInput = messages[i].Content
				break
			}
		}
		reactInputFinal = b.buildReActInput(messages)
		for i := len(messages) - 1; i >= 0; i-- {
			if strings.EqualFold(messages[i].Role, "assistant") {
				modelOutput = messages[i].Content
				break
			}
		}
	}
	hasMCPOnAssistant := false
	var lastAssistantID string
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "assistant") {
			lastAssistantID = messages[i].ID
			if len(messages[i].MCPExecutionIDs) > 0 {
				hasMCPOnAssistant = true
			}
			break
		}
	}
	if lastAssistantID != "" {
		pdHasTools, _ := b.db.ConversationHasToolProcessDetails(conversationID)
		if pdHasTools && !(hasMCPOnAssistant && reactInputContainsToolTrace(reactInputJSON)) {
			detailsMap, err := b.db.GetProcessDetailsByConversation(conversationID)
			if err != nil {
				b.logger.Warn("loadprocess detailsattack chainfailed", zap.Error(err))
			} else if dets := detailsMap[lastAssistantID]; len(dets) > 0 {
				extra := b.formatProcessDetailsForAttackChain(dets)
				if strings.TrimSpace(extra) != "" {
					reactInputFinal = reactInputFinal + "\n\n## processtoolrecord(multi-agentsubtask)\n\n" + extra
					b.logger.Info("attack chaininputprocess details",
						zap.String("conversationId", conversationID),
						zap.String("messageId", lastAssistantID),
						zap.Int("detailEvents", len(dets)))
				}
			}
		}
	}
	prompt := b.buildSimplePrompt(reactInputFinal, modelOutput)
	// fmt.Println(prompt)
	chainJSON, err := b.callAIForChainGeneration(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AIgeneratefailed: %w", err)
	}
	chainData, err := b.parseChainJSON(chainJSON)
	if err != nil {
		b.logger.Warn("parseattack chainJSONfailed", zap.Error(err), zap.String("raw_json", chainJSON))
		return &Chain{
			Nodes: []Node{},
			Edges: []Edge{},
		}, nil
	}

	b.logger.Info("attack chainbuildcompleted",
		zap.String("conversationId", conversationID),
		zap.String("dataSource", dataSource),
		zap.Int("nodes", len(chainData.Nodes)),
		zap.Int("edges", len(chainData.Edges)))
	if err := b.saveChain(conversationID, chainData.Nodes, chainData.Edges); err != nil {
		b.logger.Warn("saveattack chaindatabasefailed", zap.Error(err))
	}
	return chainData, nil
}
func reactInputContainsToolTrace(reactInputJSON string) bool {
	s := strings.TrimSpace(reactInputJSON)
	if s == "" {
		return false
	}
	return strings.Contains(s, "tool_calls") ||
		strings.Contains(s, "tool_call_id") ||
		strings.Contains(s, `"role":"tool"`) ||
		strings.Contains(s, `"role": "tool"`)
}
func (b *Builder) formatProcessDetailsForAttackChain(details []database.ProcessDetail) string {
	if len(details) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, d := range details {
		if d.EventType == "progress" || d.EventType == "thinking" || d.EventType == "planning" {
			continue
		}
		var dataMap map[string]interface{}
		if strings.TrimSpace(d.Data) != "" {
			_ = json.Unmarshal([]byte(d.Data), &dataMap)
		}
		einoRole := ""
		if v, ok := dataMap["einoRole"]; ok {
			einoRole = strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		}
		toolName := ""
		if v, ok := dataMap["toolName"]; ok {
			toolName = strings.TrimSpace(fmt.Sprint(v))
		}
		if (d.EventType == "tool_call" || d.EventType == "tool_result" || d.EventType == "tool_calls_detected" || d.EventType == "iteration" || d.EventType == "eino_recovery") && einoRole == "orchestrator" {
			sb.WriteString("[")
			sb.WriteString(d.EventType)
			sb.WriteString("] ")
			sb.WriteString(strings.TrimSpace(d.Message))
			sb.WriteString("\n")
			if strings.TrimSpace(d.Data) != "" {
				sb.WriteString(d.Data)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}
		if d.EventType == "tool_call" && strings.EqualFold(toolName, "task") {
			sb.WriteString("[dispatch_subagent_task] ")
			sb.WriteString(strings.TrimSpace(d.Message))
			sb.WriteString("\n")
			if strings.TrimSpace(d.Data) != "" {
				sb.WriteString(d.Data)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}
		if d.EventType == "eino_agent_reply" && einoRole == "sub" {
			sb.WriteString("[subagent_final_reply] ")
			sb.WriteString(strings.TrimSpace(d.Message))
			sb.WriteString("\n")
			if strings.TrimSpace(d.Data) != "" {
				sb.WriteString(d.Data)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}
	}
	return strings.TrimSpace(sb.String())
}
func (b *Builder) buildReActInput(messages []database.Message) string {
	var builder strings.Builder
	for _, msg := range messages {
		builder.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, msg.Content))
	}
	return builder.String()
}
// func (b *Builder) extractUserInputFromReActInput(reactInputJSON string) string {
// 	var messages []map[string]interface{}
// 	if err := json.Unmarshal([]byte(reactInputJSON), &messages); err != nil {
// 		return ""
// 	}
// 	for i := len(messages) - 1; i >= 0; i-- {
// 		if role, ok := messages[i]["role"].(string); ok && strings.EqualFold(role, "user") {
// 			if content, ok := messages[i]["content"].(string); ok {
// 				return content
// 			}
// 		}
// 	}

// 	return ""
// }
func (b *Builder) formatReActInputFromJSON(reactInputJSON string) string {
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(reactInputJSON), &messages); err != nil {
		b.logger.Warn("parseReActinputJSONfailed", zap.Error(err))
		return reactInputJSON
	}

	var builder strings.Builder
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		if role == "assistant" {
			if toolCalls, ok := msg["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
				if content != "" {
					builder.WriteString(fmt.Sprintf("[%s]: %s\n", role, content))
				}
				builder.WriteString(fmt.Sprintf("[%s] toolcall (%d):\n", role, len(toolCalls)))
				for i, toolCall := range toolCalls {
					if tc, ok := toolCall.(map[string]interface{}); ok {
						toolCallID, _ := tc["id"].(string)
						if funcData, ok := tc["function"].(map[string]interface{}); ok {
							toolName, _ := funcData["name"].(string)
							arguments, _ := funcData["arguments"].(string)
							builder.WriteString(fmt.Sprintf(" [toolcall %d]\n", i+1))
							builder.WriteString(fmt.Sprintf(" ID: %s\n", toolCallID))
							builder.WriteString(fmt.Sprintf(" toolname: %s\n", toolName))
							builder.WriteString(fmt.Sprintf(" : %s\n", arguments))
						}
					}
				}
				builder.WriteString("\n")
				continue
			}
		}
		if role == "tool" {
			toolCallID, _ := msg["tool_call_id"].(string)
			if toolCallID != "" {
				builder.WriteString(fmt.Sprintf("[%s] (tool_call_id: %s):\n%s\n\n", role, toolCallID, content))
			} else {
				builder.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, content))
			}
			continue
		}
		builder.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, content))
	}

	return builder.String()
}
func (b *Builder) buildSimplePrompt(reactInput, modelOutput string) string {
	return fmt.Sprintf(`Security testingattack chainbuild.TasksChatrecordtoolresult,build/attack chain,Penetration testingprocesspath.

## 

buildattack chain:
1. Penetration testing(Vulnerability discovery)
2. failedfetch
3. tool
4. Vulnerability discovery

****:.tool,.

## build()

### :
ReActinputtoolcallmodeloutput,:
- (IP//URL)
- tool
- toolback(successfulresult/error/)
- AIprocess

### :
toolexecution record,****:
- **target**:createtarget
- **action**:toolcreateaction(failed/successfulRecon/Vulnerability validation)
- **vulnerability**:Vulnerabilitiescreatevulnerability
- ****:ReActinputtoolcall,toolattack chain

### :build()
**:build,.**
connection,(agent,):
- ****:(:scan,)
- ****:(:Vulnerabilities)
- actionactionresult
- vulnerabilityaction
- failedsuccessful
- ****:,build

### :
- ****:tool,
- ****:action(toolcall)
- **delete**:deletefailed(output/error/failed)
- ****:,.attack chainPenetration testingprocess
- attack chain,

## 

### target()
- ****:
- **create**:(IP/)createtarget
- ****:connection,
- **metadata.target**:record(IP//URL)

### action()
- ****:recordtoolAIresult
- ****:
 * 15-25,
 * successful:result("scan80/443/8080"/"directoryscan/adminpath")
 * failed:failed("SQL(WAF)"/"scan()")
- **ai_analysis**:
 * successful:summarytool,
 * failed:failed//
 * 150,/
- **findings**:
 * toolbackresult
 * finding/
 * successful:(["80", "443", "HTTPApache 2.4"])
 * failed:failed(["WAF", "back403", "Cloudflare"])
- **status**:
 * successful:"success"
 * failed:"failed_insight"
- **risk_score**:0(action)

### vulnerability(Vulnerabilities)
- ****:recordVulnerabilities
- **create**:
 * Vulnerabilities,Vulnerabilities
 * Vulnerabilities(SQLbackdatabaseerror/XSSsuccessful)
- **risk_score**:
 * critical(90-100):(RCE/SQL)
 * high(80-89):
 * medium(60-79):
 * low(40-59):
- **metadata**:
 * vulnerability_type:Vulnerabilities(SQL/XSS/RCE)
 * description:Vulnerabilities//
 * severity:critical/high/medium/low
 * location:Vulnerabilities(URL//Filespath)

## 

### failed
failedcreate,:
- toolbackerror(error/connection/failed)
- connectionfailed(/)
- WAF/(back403/406,)
- toolconfigerror(call)
- (DNSparsefailed/)

### deletefailed
create:
- outputtoolcall
- error(,)
- failed(error)

### 
:
- toolcall(nmapscan,"scan")
- (directoryscantool,"directoryscan")

### 
- ****:tool,delete
- ****:8-15,,(20)
- ****:successful/failed/Vulnerabilities/Recon
- ****:toolcall(nmapscan,"scan")
- **delete**:outputtoolcall/error/failed(error)
- ****:,.attack chainPenetration testingprocess

## 

### 
- **leads_to**:"""",action→action/target→action
 * :scan → directoryscan(80,directoryscan)
- **discovers**:"",**action→vulnerability**
 * :SQL → SQLVulnerabilities
 * ****:action→vulnerabilitydiscovers,actionvulnerability,discovers
- **enables**:"""",**vulnerability→vulnerability/action→action(result)**
 * :Vulnerabilities → Vulnerabilities()
 * ****:enablesaction→vulnerability,action→vulnerabilitydiscovers

### 
- **1-2**:()
- **3-4**:()
- **5-7**:(Vulnerabilities/)
- **8-10**:(Vulnerabilitiessuccessful/)

### DAG()
**:generateDAG(),.**

- ****:id"node_1"start(node_1, node_2, node_3...)
- ****:sourceidtargetid(source < target),
 * :node_1 → node_2 ✓()
 * :node_2 → node_1 ✗(error,)
 * :node_3 → node_5 ✓()
- ****:outputJSON,,nosource >= target
- ****:connection()
- **DAG**:
 * (),:node_2(scan)connectionnode_3/node_4/node_5
 * (),:node_3/node_4/node_5node_6(Vulnerabilities)
 * ,buildDAG
- ****:id,(),

## attack chain

buildattack chain:
1. ****:start？(target)
2. **process**:？(action)
3. **failed**:？(failed_insight)
4. ****:？(actionfindings)
5. **Vulnerabilities**:Vulnerabilities？(action→vulnerability)
6. **path**:path？(targetvulnerabilitypath)

## ReActinput

%s

## modeloutput

%s

## outputFormat

JSONFormatoutput,:

**:,node_2(scan)connection(node_3/node_4),.**

{
 "nodes": [
 {
 "id": "node_1",
 "type": "target",
 "label": ": example.com",
 "risk_score": 40,
 "metadata": {
 "target": "example.com"
 }
 },
 {
 "id": "node_2",
 "type": "action",
 "label": "scan80/443/8080",
 "risk_score": 0,
 "metadata": {
 "tool_name": "nmap",
 "tool_intent": "scan",
 "ai_analysis": "nmapscan,80/443/8080.80HTTP,443HTTPS,8080.Webapply.",
 "findings": ["80", "443", "8080", "HTTPApache 2.4"]
 }
 },
 {
 "id": "node_3",
 "type": "action",
 "label": "directoryscan/admin",
 "risk_score": 0,
 "metadata": {
 "tool_name": "dirsearch",
 "tool_intent": "directoryscan",
 "ai_analysis": "dirsearchdirectoryscan,/admindirectory.directory,.",
 "findings": ["/admindirectory", "back200status", ""]
 }
 },
 {
 "id": "node_4",
 "type": "action",
 "label": "WebApache 2.4",
 "risk_score": 0,
 "metadata": {
 "tool_name": "whatweb",
 "tool_intent": "Web",
 "ai_analysis": "Apache 2.4,Vulnerabilities.",
 "findings": ["Apache 2.4", "PHP"]
 }
 },
 {
 "id": "node_5",
 "type": "action",
 "label": "SQL(WAF)",
 "risk_score": 0,
 "metadata": {
 "tool_name": "sqlmap",
 "tool_intent": "SQL",
 "ai_analysis": "/login.phpSQLWAF,back403error.errorCloudflare.WAF,.",
 "findings": ["WAF", "back403", "Cloudflare", "WAF"],
 "status": "failed_insight"
 }
 },
 {
 "id": "node_6",
 "type": "vulnerability",
 "label": "SQLVulnerabilities",
 "risk_score": 85,
 "metadata": {
 "vulnerability_type": "SQL",
 "description": "/admin/login.phpusernameParameter discoverySQLVulnerabilities,payloadlogin,fetch.Vulnerabilitiesbackdatabaseerror,.",
 "severity": "high",
 "location": "/admin/login.php?username="
 }
 }
 ],
 "edges": [
 {
 "source": "node_1",
 "target": "node_2",
 "type": "leads_to",
 "weight": 3
 },
 {
 "source": "node_2",
 "target": "node_3",
 "type": "leads_to",
 "weight": 4
 },
 {
 "source": "node_2",
 "target": "node_4",
 "type": "leads_to",
 "weight": 3
 },
 {
 "source": "node_3",
 "target": "node_5",
 "type": "leads_to",
 "weight": 4
 },
 {
 "source": "node_5",
 "target": "node_6",
 "type": "discovers",
 "weight": 7
 }
 ]
}

## 

1. ****:ReActinputtoolbackresult.,backnodesedges.
2. **DAG**:buildDAG(),.sourceidtargetid(source < target).
3. ****:,targetnode_1,action,vulnerability.
4. ****:tool,delete.attack chainVulnerability discoveryprocess.
5. ****:attack chain/Penetration testing,.
6. ****:,Penetration testing.
7. ****:,.
8. ****:metadata,sourcetarget,no,no.
9. ****:,(20),.
10. **output**:outputJSON,source < target,DAG.

startbuildattack chain:`, reactInput, modelOutput)
}
func (b *Builder) saveChain(conversationID string, nodes []Node, edges []Edge) error {
	if err := b.db.DeleteAttackChain(conversationID); err != nil {
		b.logger.Warn("deleteattack chainfailed", zap.Error(err))
	}

	for _, node := range nodes {
		metadataJSON, _ := json.Marshal(node.Metadata)
		if err := b.db.SaveAttackChainNode(conversationID, node.ID, node.Type, node.Label, "", string(metadataJSON), node.RiskScore); err != nil {
			b.logger.Warn("saveattack chainfailed", zap.String("nodeId", node.ID), zap.Error(err))
		}
	}
	for _, edge := range edges {
		if err := b.db.SaveAttackChainEdge(conversationID, edge.ID, edge.Source, edge.Target, edge.Type, edge.Weight); err != nil {
			b.logger.Warn("saveattack chainfailed", zap.String("edgeId", edge.ID), zap.Error(err))
		}
	}

	return nil
}
func (b *Builder) LoadChainFromDatabase(conversationID string) (*Chain, error) {
	nodes, err := b.db.LoadAttackChainNodes(conversationID)
	if err != nil {
		return nil, fmt.Errorf("loadattack chainfailed: %w", err)
	}

	edges, err := b.db.LoadAttackChainEdges(conversationID)
	if err != nil {
		return nil, fmt.Errorf("loadattack chainfailed: %w", err)
	}

	return &Chain{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
func (b *Builder) callAIForChainGeneration(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": b.openAIConfig.Model,
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": "Security testing,buildattack chain.JSONFormatbackattack chain.",
			},
			{
				"role": "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
		"max_tokens": 8000,
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if b.openAIClient == nil {
		return "", fmt.Errorf("OpenAI client is not initialized")
	}
	if err := b.openAIClient.ChatCompletion(ctx, requestBody, &apiResponse); err != nil {
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) {
			bodyStr := strings.ToLower(apiErr.Body)
			if strings.Contains(bodyStr, "context") || strings.Contains(bodyStr, "length") || strings.Contains(bodyStr, "too long") {
				return "", fmt.Errorf("context length exceeded")
			}
		} else if strings.Contains(strings.ToLower(err.Error()), "context") || strings.Contains(strings.ToLower(err.Error()), "length") {
			return "", fmt.Errorf("context length exceeded")
		}
		return "", fmt.Errorf("Request failed: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("APIback")
	}

	content := strings.TrimSpace(apiResponse.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return content, nil
}
type ChainJSON struct {
	Nodes []struct {
		ID string `json:"id"`
		Type string `json:"type"`
		Label string `json:"label"`
		RiskScore int `json:"risk_score"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"nodes"`
	Edges []struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type string `json:"type"`
		Weight int `json:"weight"`
	} `json:"edges"`
}
func (b *Builder) parseChainJSON(chainJSON string) (*Chain, error) {
	var chainData ChainJSON
	if err := json.Unmarshal([]byte(chainJSON), &chainData); err != nil {
		return nil, fmt.Errorf("parseJSONfailed: %w", err)
	}
	nodeIDMap := make(map[string]string)
	nodes := make([]Node, 0, len(chainData.Nodes))
	for _, n := range chainData.Nodes {
		newNodeID := fmt.Sprintf("node_%s", uuid.New().String())
		nodeIDMap[n.ID] = newNodeID

		node := Node{
			ID: newNodeID,
			Type: n.Type,
			Label: n.Label,
			RiskScore: n.RiskScore,
			Metadata: n.Metadata,
		}
		if node.Metadata == nil {
			node.Metadata = make(map[string]interface{})
		}
		nodes = append(nodes, node)
	}
	edges := make([]Edge, 0, len(chainData.Edges))
	for _, e := range chainData.Edges {
		sourceID, ok := nodeIDMap[e.Source]
		if !ok {
			continue
		}
		targetID, ok := nodeIDMap[e.Target]
		if !ok {
			continue
		}
		edgeID := fmt.Sprintf("edge_%s", uuid.New().String())

		edges = append(edges, Edge{
			ID: edgeID,
			Source: sourceID,
			Target: targetID,
			Type: e.Type,
			Weight: e.Weight,
		})
	}

	return &Chain{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
