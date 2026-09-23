package handler

var apiDocI18nTagToKey = map[string]string{
	"authentication": "auth", "conversationmanage": "conversationManagement", "conversation interaction": "conversationInteraction",
	"batch task": "batchTasks", "conversation group": "conversationGroups", "vulnerabilitymanage": "vulnerabilityManagement",
	"rolemanage": "roleManagement", "Skillsmanage": "skillsManagement", "monitor": "monitoring",
	"configuremanage": "configManagement", "externalMCPmanage": "externalMCPManagement", "attack chain": "attackChain",
	"knowledge base": "knowledgeBase", "MCP": "mcp",
}

var apiDocI18nSummaryToKey = map[string]string{
	"login": "login", "": "logout", "modifypassword": "changePassword", "validateToken": "validateToken",
	"createconversation": "createConversation", "listconversation": "listConversations", "conversationdetails": "getConversationDetail",
	"morenew conversation": "updateConversation", "deleteconversation": "deleteConversation", "getconversationresult": "getConversationResult",
	"send message and get AI reply()": "sendMessageNonStream", "send message and get AI reply (stream)()": "sendMessageStream",
	"canceltask": "cancelTask", "list in of task": "listRunningTasks", "list completed of task": "listCompletedTasks",
	"create batch task queue": "createBatchQueue", "list batch task queue": "listBatchQueues", "get batch task queue": "getBatchQueue",
	"delete batch task queue": "deleteBatchQueue", "start batch task queue": "startBatchQueue", "batch task queue": "pauseBatchQueue",
	"task queue": "addTaskToQueue", "SQL injection scan": "sqlInjectionScan", "port scan": "portScan",
	"update batch task": "updateBatchTask", "delete batch task": "deleteBatchTask",
	"create group": "createGroup", "list group": "listGroups", "get group": "getGroup", "update group": "updateGroup",
	"delete group": "deleteGroup", "get group in of conversation": "getGroupConversations", "conversation group": "addConversationToGroup",
	"from group conversation": "removeConversationFromGroup",
	"list vulnerability": "listVulnerabilities", "create vulnerability": "createVulnerability", "get vulnerability statistics": "getVulnerabilityStats",
	"get vulnerability": "getVulnerability", "update vulnerability": "updateVulnerability", "delete vulnerability": "deleteVulnerability",
	"list role": "listRoles", "create role": "createRole", "get role": "getRole", "update role": "updateRole", "delete role": "deleteRole",
	"get Skills list": "getAvailableSkills", "list Skills": "listSkills", "create Skill": "createSkill",
	"get Skill statistics": "getSkillStats", "clear Skill statistics": "clearSkillStats", "get Skill": "getSkill",
	"update Skill": "updateSkill", "delete Skill": "deleteSkill", "get bind role": "getBoundRoles",
	"get monitor information": "getMonitorInfo", "get execution record": "getExecutionRecords", "delete execution record": "deleteExecutionRecord",
	"batch delete execution record": "batchDeleteExecutionRecords", "get statistics": "getStats",
	"get configure": "getConfig", "update configure": "updateConfig", "get tool configure": "getToolConfig", "apply configure": "applyConfig",
	"list external MCP": "listExternalMCP", "get external MCP statistics": "getExternalMCPStats", "get external MCP": "getExternalMCP",
	"or update external MCP": "addOrUpdateExternalMCP", "stdio mode configure": "stdioModeConfig", "SSE mode configure": "sseModeConfig",
	"delete external MCP": "deleteExternalMCP", "start external MCP": "startExternalMCP", "stop external MCP": "stopExternalMCP",
	"get attack chain": "getAttackChain", "generate attack chain": "regenerateAttackChain",
	"conversation pin": "pinConversation", "group pin": "pinGroup", "group in conversation of pin": "pinGroupConversation",
	"get category": "getCategories", "list knowledge item": "listKnowledgeItems", "create knowledge item": "createKnowledgeItem",
	"get knowledge item": "getKnowledgeItem", "update knowledge item": "updateKnowledgeItem", "delete knowledge item": "deleteKnowledgeItem",
	"get status": "getIndexStatus", "_rebuild": "rebuildIndex", "scan knowledge base": "scanKnowledgeBase",
	"search knowledge base": "searchKnowledgeBase", "search": "basicSearch", "risk type search": "searchByRiskType",
	"get retrieval log": "getRetrievalLogs", "delete retrieval log": "deleteRetrievalLog",
	"MCP": "mcpEndpoint", "list tool": "listAllTools", "call tool": "invokeTool", "initialize connection": "initConnection",
	"successful response": "successResponse", "error response": "errorResponse",
}

var apiDocI18nResponseDescToKey = map[string]string{
	"getsuccessful": "getSuccess", "unauthorized": "unauthorized", "unauthorized,need validToken": "unauthorizedToken",
	"createsuccessful": "createSuccess", "request parameter error": "badRequest", "conversation does not exist": "conversationNotFound",
	"conversation does not existorresultdoes not exist": "conversationOrResultNotFound", "request parameter error(such astaskis)": "badRequestTaskEmpty",
	"request parameter errororgroup namealready exists": "badRequestGroupNameExists", "group does not exist": "groupNotFound",
	"request parameter error(such asconfigureFormat/needfield)": "badRequestConfig",
	"request parameter error(such asqueryis)": "badRequestQueryEmpty", "(supportPOSTrequest)": "methodNotAllowed",
	"loginsuccessful": "loginSuccess", "passworderror": "invalidPassword", "successful logout": "logoutSuccess",
	"passwordmodifysuccessful": "passwordChanged", "Token": "tokenValid", "Tokeninvalidorhas": "tokenInvalid",
	"conversationcreatesuccessful": "conversationCreated", "internal server error": "internalError", "updatesuccessful": "updateSuccess",
	"deletesuccessful": "deleteSuccess", "queue does not exist": "queueNotFound", "startsuccessful": "startSuccess",
	"successful pause": "pauseSuccess", "successful add": "addSuccess",
	"taskdoes not exist": "taskNotFound", "conversationorgroup does not exist": "conversationOrGroupNotFound",
	"cancelrequesthas": "cancelSubmitted", "not foundexecuteoftask": "noRunningTask",
	"messagesuccessful,returnAIreply": "messageSent", "response(Server-Sent Events)": "streamResponse",
}
func enrichSpecWithI18nKeys(spec map[string]interface{}) {
	paths, _ := spec["paths"].(map[string]interface{})
	if paths == nil {
		return
	}
	for _, pathItem := range paths {
		pm, _ := pathItem.(map[string]interface{})
		if pm == nil {
			continue
		}
		for _, method := range []string{"get", "post", "put", "delete", "patch"} {
			opVal, ok := pm[method]
			if !ok {
				continue
			}
			op, _ := opVal.(map[string]interface{})
			if op == nil {
				continue
			}
			switch tags := op["tags"].(type) {
			case []string:
				if len(tags) > 0 {
					keys := make([]string, 0, len(tags))
					for _, s := range tags {
						if k := apiDocI18nTagToKey[s]; k != "" {
							keys = append(keys, k)
						} else {
							keys = append(keys, s)
						}
					}
					op["x-i18n-tags"] = keys
				}
			case []interface{}:
				if len(tags) > 0 {
					keys := make([]interface{}, 0, len(tags))
					for _, t := range tags {
						if s, ok := t.(string); ok {
							if k := apiDocI18nTagToKey[s]; k != "" {
								keys = append(keys, k)
							} else {
								keys = append(keys, s)
							}
						}
					}
					if len(keys) > 0 {
						op["x-i18n-tags"] = keys
					}
				}
			}
			// x-i18n-summary
			if summary, _ := op["summary"].(string); summary != "" {
				if k := apiDocI18nSummaryToKey[summary]; k != "" {
					op["x-i18n-summary"] = k
				}
			}
			// responses -> each/peritems status -> x-i18n-description
			if respMap, _ := op["responses"].(map[string]interface{}); respMap != nil {
				for _, rv := range respMap {
					if r, _ := rv.(map[string]interface{}); r != nil {
						if desc, _ := r["description"].(string); desc != "" {
							if k := apiDocI18nResponseDescToKey[desc]; k != "" {
								r["x-i18n-description"] = k
							}
						}
					}
				}
			}
		}
	}
}
