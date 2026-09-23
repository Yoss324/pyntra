package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"pyntra/internal/mcp"
	"pyntra/internal/mcp/builtin"

	"go.uber.org/zap"
)
func RegisterKnowledgeTool(
	mcpServer *mcp.Server,
	retriever *Retriever,
	manager *Manager,
	logger *zap.Logger,
) {
	listRiskTypesTool := mcp.Tool{
		Name: builtin.ToolListKnowledgeRiskTypes,
		Description: "fetchknowledge base(risk_type).knowledge base,calltoolfetch,,retrievalretrieval.",
		ShortDescription: "fetchknowledge base",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
			"required": []string{},
		},
	}

	listRiskTypesHandler := func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		categories, err := manager.GetCategories()
		if err != nil {
			logger.Error("fetchfailed", zap.Error(err))
			return &mcp.ToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: fmt.Sprintf("fetchfailed: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		if len(categories) == 0 {
			return &mcp.ToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: "knowledge base.",
					},
				},
			}, nil
		}

		var resultText strings.Builder
		resultText.WriteString(fmt.Sprintf("knowledge base %d :\n\n", len(categories)))
		for i, category := range categories {
			resultText.WriteString(fmt.Sprintf("%d. %s\n", i+1, category))
		}
		resultText.WriteString("\nhint:call " + builtin.ToolSearchKnowledgeBase + " tool, risk_type ,retrieval.")

		return &mcp.ToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: resultText.String(),
				},
			},
		}, nil
	}

	mcpServer.RegisterTool(listRiskTypesTool, listRiskTypesHandler)
	logger.Info("toolregister", zap.String("toolName", listRiskTypesTool.Name))
	searchTool := mcp.Tool{
		Name: builtin.ToolSearchKnowledgeBase,
		Description: "knowledge baseKnowledge.Vulnerabilities//Knowledge,toolretrieval.toolretrieval( Eino retriever ).:call " + builtin.ToolListKnowledgeRiskTypes + " toolfetch, risk_type ,retrieval.",
		ShortDescription: "knowledge baseKnowledge(retrieval)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
					"description": "query,Knowledge",
				},
				"risk_type": map[string]interface{}{
					"type": "string",
					"description": ":(:SQL/XSS/Files).call " + builtin.ToolListKnowledgeRiskTypes + " toolfetch,,retrieval..",
				},
			},
			"required": []string{"query"},
		},
	}

	searchHandler := func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return &mcp.ToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: "error: querycannot be empty",
					},
				},
				IsError: true,
			}, nil
		}

		riskType := ""
		if rt, ok := args["risk_type"].(string); ok && rt != "" {
			riskType = rt
		}

		logger.Info("knowledge baseretrieval",
			zap.String("query", query),
			zap.String("riskType", riskType),
		)
		searchReq := &SearchRequest{
			Query: query,
			RiskType: riskType,
			TopK: 5,
		}

		results, err := retriever.Search(ctx, searchReq)
		if err != nil {
			logger.Error("knowledge baseretrievalfailed", zap.Error(err))
			return &mcp.ToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: fmt.Sprintf("retrievalfailed: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		if len(results) == 0 {
			return &mcp.ToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: fmt.Sprintf("query '%s' Knowledge.:\n1. \n2. \n3. knowledge base", query),
					},
				},
			}, nil
		}
		var resultText strings.Builder
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
		type itemGroup struct {
			itemID string
			results []*RetrievalResult
			maxScore float64
		}
		itemGroups := make([]*itemGroup, 0)
		itemMap := make(map[string]*itemGroup)

		for _, result := range results {
			itemID := result.Item.ID
			group, exists := itemMap[itemID]
			if !exists {
				group = &itemGroup{
					itemID: itemID,
					results: make([]*RetrievalResult, 0),
					maxScore: result.Score,
				}
				itemMap[itemID] = group
				itemGroups = append(itemGroups, group)
			}
			group.results = append(group.results, result)
			if result.Score > group.maxScore {
				group.maxScore = result.Score
			}
		}
		sort.Slice(itemGroups, func(i, j int) bool {
			return itemGroups[i].maxScore > itemGroups[j].maxScore
		})
		retrievedItemIDs := make([]string, 0, len(itemGroups))

		resultText.WriteString(fmt.Sprintf(" %d Knowledge:\n\n", len(results)))

		resultIndex := 1
		for _, group := range itemGroups {
			itemResults := group.results
			mainResult := itemResults[0]
			maxScore := mainResult.Score
			for _, result := range itemResults {
				if result.Score > maxScore {
					maxScore = result.Score
					mainResult = result
				}
			}
			sort.Slice(itemResults, func(i, j int) bool {
				return itemResults[i].Chunk.ChunkIndex < itemResults[j].Chunk.ChunkIndex
			})

			resultText.WriteString(fmt.Sprintf("--- result %d (: %.2f%%) ---\n",
				resultIndex, mainResult.Similarity*100))
			resultText.WriteString(fmt.Sprintf(": [%s] %s (ID: %s)\n", mainResult.Item.Category, mainResult.Item.Title, mainResult.Item.ID))
			if len(itemResults) == 1 {
				resultText.WriteString(fmt.Sprintf(":\n%s\n", mainResult.Chunk.ChunkText))
			} else {
				resultText.WriteString("():\n")
				for i, result := range itemResults {
					marker := ""
					if result.Chunk.ID == mainResult.Chunk.ID {
						marker = " []"
					}
					resultText.WriteString(fmt.Sprintf(" [ %d%s]\n%s\n", i+1, marker, result.Chunk.ChunkText))
				}
			}
			resultText.WriteString("\n")

			if !contains(retrievedItemIDs, group.itemID) {
				retrievedItemIDs = append(retrievedItemIDs, group.itemID)
			}
			resultIndex++
		}
		if len(retrievedItemIDs) > 0 {
			metadataJSON, _ := json.Marshal(map[string]interface{}{
				"_metadata": map[string]interface{}{
					"retrievedItemIDs": retrievedItemIDs,
				},
			})
			resultText.WriteString(fmt.Sprintf("\n<!-- METADATA: %s -->", string(metadataJSON)))
		}

		return &mcp.ToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: resultText.String(),
				},
			},
		}, nil
	}

	mcpServer.RegisterTool(searchTool, searchHandler)
	logger.Info("Knowledgeretrievaltoolregister", zap.String("toolName", searchTool.Name))
}
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
func GetRetrievalMetadata(args map[string]interface{}) (query string, riskType string) {
	if q, ok := args["query"].(string); ok {
		query = q
	}
	if rt, ok := args["risk_type"].(string); ok {
		riskType = rt
	}
	return
}
func FormatRetrievalResults(results []*RetrievalResult) string {
	if len(results) == 0 {
		return "result"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("retrieval %d results:\n", len(results)))

	itemIDs := make(map[string]bool)
	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. [%s] %s (: %.2f%%)\n",
			i+1, result.Item.Category, result.Item.Title, result.Similarity*100))
		itemIDs[result.Item.ID] = true
	}
	ids := make([]string, 0, len(itemIDs))
	for id := range itemIDs {
		ids = append(ids, id)
	}
	idsJSON, _ := json.Marshal(ids)
	builder.WriteString(fmt.Sprintf("\nretrievalKnowledgeID: %s", string(idsJSON)))

	return builder.String()
}
