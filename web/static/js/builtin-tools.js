/**
 * toolname
 * toolname,
 * 
 * Note: internal/mcp/builtin/constants.go 
 */
const BuiltinTools = {
 RECORD_VULNERABILITY: 'record_vulnerability',
 LIST_KNOWLEDGE_RISK_TYPES: 'list_knowledge_risk_types',
 SEARCH_KNOWLEDGE_BASE: 'search_knowledge_base'
};
function isBuiltinTool(toolName) {
 return Object.values(BuiltinTools).includes(toolName);
}
function getAllBuiltinTools() {
 return Object.values(BuiltinTools);
}

