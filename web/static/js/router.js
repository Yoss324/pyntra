let currentPage = 'dashboard';
function initRouter() {
 const hash = window.location.hash.slice(1);
 if (hash) {
 const hashParts = hash.split('?');
 const pageId = hashParts[0];
 if (pageId && ['dashboard', 'chat', 'vulnerabilities', 'webshell', 'chat-files', 'mcp-monitor', 'mcp-management', 'knowledge-management', 'knowledge-retrieval-logs', 'roles-management', 'skills-monitor', 'skills-management', 'agents-management', 'settings', 'tasks'].includes(pageId)) {
 switchPage(pageId);
 if (pageId === 'chat' && hashParts.length > 1) {
 const params = new URLSearchParams(hashParts[1]);
 const conversationId = params.get('conversation');
 if (conversationId) {
 setTimeout(() => {
 if (typeof loadConversation === 'function') {
 loadConversation(conversationId);
 } else if (typeof window.loadConversation === 'function') {
 window.loadConversation(conversationId);
 } else {
 console.warn('loadConversation function not found');
 }
 }, 500);
 }
 }
 return;
 }
 }
 switchPage('dashboard');
}
function switchPage(pageId) {
 document.querySelectorAll('.page').forEach(page => {
 page.classList.remove('active');
 });
 const targetPage = document.getElementById(`page-${pageId}`);
 if (targetPage) {
 targetPage.classList.add('active');
 currentPage = pageId;
 window.location.hash = pageId;
 updateNavState(pageId);
 initPage(pageId);
 }
}
function updateNavState(pageId) {
 document.querySelectorAll('.nav-item').forEach(item => {
 item.classList.remove('active');
 });
 
 document.querySelectorAll('.nav-submenu-item').forEach(item => {
 item.classList.remove('active');
 });
 if (pageId === 'mcp-monitor' || pageId === 'mcp-management') {
 const mcpItem = document.querySelector('.nav-item[data-page="mcp"]');
 if (mcpItem) {
 mcpItem.classList.add('active');
 mcpItem.classList.add('expanded');
 }
 
 const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
 if (submenuItem) {
 submenuItem.classList.add('active');
 }
 } else if (pageId === 'knowledge-management' || pageId === 'knowledge-retrieval-logs') {
 const knowledgeItem = document.querySelector('.nav-item[data-page="knowledge"]');
 if (knowledgeItem) {
 knowledgeItem.classList.add('active');
 knowledgeItem.classList.add('expanded');
 }
 
 const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
 if (submenuItem) {
 submenuItem.classList.add('active');
 }
 } else if (pageId === 'skills-monitor' || pageId === 'skills-management') {
 const skillsItem = document.querySelector('.nav-item[data-page="skills"]');
 if (skillsItem) {
 skillsItem.classList.add('active');
 skillsItem.classList.add('expanded');
 }
 
 const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
 if (submenuItem) {
 submenuItem.classList.add('active');
 }
 } else if (pageId === 'agents-management') {
 const agentsItem = document.querySelector('.nav-item[data-page="agents"]');
 if (agentsItem) {
 agentsItem.classList.add('active');
 agentsItem.classList.add('expanded');
 }
 const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
 if (submenuItem) {
 submenuItem.classList.add('active');
 }
 } else if (pageId === 'roles-management') {
 const rolesItem = document.querySelector('.nav-item[data-page="roles"]');
 if (rolesItem) {
 rolesItem.classList.add('active');
 rolesItem.classList.add('expanded');
 }
 
 const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
 if (submenuItem) {
 submenuItem.classList.add('active');
 }
 } else {
 const navItem = document.querySelector(`.nav-item[data-page="${pageId}"]`);
 if (navItem) {
 navItem.classList.add('active');
 }
 }
}
function toggleSubmenu(menuId) {
 const sidebar = document.getElementById('main-sidebar');
 const navItem = document.querySelector(`.nav-item[data-page="${menuId}"]`);
 
 if (!navItem) return;
 if (sidebar && sidebar.classList.contains('collapsed')) {
 showSubmenuPopup(navItem, menuId);
 } else {
 navItem.classList.toggle('expanded');
 }
}
function showSubmenuPopup(navItem, menuId) {
 const existingPopup = document.querySelector('.submenu-popup');
 if (existingPopup) {
 existingPopup.remove();
 return; // ,
 }
 
 const navItemContent = navItem.querySelector('.nav-item-content');
 const submenu = navItem.querySelector('.nav-submenu');
 
 if (!submenu) return;
 const rect = navItemContent.getBoundingClientRect();
 const popup = document.createElement('div');
 popup.className = 'submenu-popup';
 popup.style.position = 'fixed';
 popup.style.left = (rect.right + 8) + 'px';
 popup.style.top = rect.top + 'px';
 popup.style.zIndex = '1000';
 const submenuItems = submenu.querySelectorAll('.nav-submenu-item');
 submenuItems.forEach(item => {
 const popupItem = document.createElement('div');
 popupItem.className = 'submenu-popup-item';
 popupItem.textContent = item.textContent.trim();
 const pageId = item.getAttribute('data-page');
 if (pageId && document.querySelector(`.nav-submenu-item[data-page="${pageId}"].active`)) {
 popupItem.classList.add('active');
 }
 
 popupItem.onclick = function(e) {
 e.stopPropagation();
 e.preventDefault();
 const pageId = item.getAttribute('data-page');
 if (pageId) {
 switchPage(pageId);
 }
 popup.remove();
 document.removeEventListener('click', closePopup);
 };
 popup.appendChild(popupItem);
 });
 
 document.body.appendChild(popup);
 const closePopup = function(e) {
 if (!popup.contains(e.target) && !navItem.contains(e.target)) {
 popup.remove();
 document.removeEventListener('click', closePopup);
 }
 };
 setTimeout(() => {
 document.addEventListener('click', closePopup);
 }, 0);
}
async function initPage(pageId) {
 if (window.i18nReady) await window.i18nReady;
 switch(pageId) {
 case 'dashboard':
 if (typeof refreshDashboard === 'function') {
 refreshDashboard();
 }
 break;
 case 'chat':
 initConversationSidebarState();
 break;
 case 'tasks':
 if (typeof initTasksPage === 'function') {
 initTasksPage();
 }
 break;
 case 'mcp-monitor':
 if (typeof refreshMonitorPanel === 'function') {
 refreshMonitorPanel();
 }
 break;
 case 'mcp-management':
 if (typeof loadExternalMCPs === 'function') {
 loadExternalMCPs().catch(err => {
 console.warn('loadMCPfailed:', err);
 });
 }
 if (typeof loadToolsList === 'function') {
 if (typeof getToolsPageSize === 'function' && typeof toolsPagination !== 'undefined') {
 toolsPagination.pageSize = getToolsPageSize();
 }
 setTimeout(() => {
 loadToolsList(1, '').catch(err => {
 console.error('Failed to load tool list:', err);
 });
 }, 100);
 }
 break;
 case 'vulnerabilities':
 if (typeof initVulnerabilityPage === 'function') {
 initVulnerabilityPage();
 }
 break;
 case 'webshell':
 if (typeof initWebshellPage === 'function') {
 initWebshellPage();
 }
 break;
 case 'chat-files':
 if (typeof initChatFilesPage === 'function') {
 initChatFilesPage();
 }
 break;
 case 'settings':
 if (typeof loadConfig === 'function') {
 loadConfig(false);
 }
 break;
 case 'roles-management':
 const rolesSearchInput = document.getElementById('roles-search');
 if (rolesSearchInput) {
 rolesSearchInput.value = '';
 }
 const rolesSearchClear = document.getElementById('roles-search-clear');
 if (rolesSearchClear) {
 rolesSearchClear.style.display = 'none';
 }
 if (typeof loadRoles === 'function') {
 loadRoles().then(() => {
 if (typeof renderRolesList === 'function') {
 renderRolesList();
 }
 });
 }
 break;
 case 'skills-monitor':
 if (typeof loadSkillsMonitor === 'function') {
 loadSkillsMonitor();
 }
 break;
 case 'skills-management':
 const skillsSearchInput = document.getElementById('skills-search');
 if (skillsSearchInput) {
 skillsSearchInput.value = '';
 }
 const skillsSearchClear = document.getElementById('skills-search-clear');
 if (skillsSearchClear) {
 skillsSearchClear.style.display = 'none';
 }
 if (typeof initSkillsPagination === 'function') {
 initSkillsPagination();
 }
 if (typeof loadSkills === 'function') {
 loadSkills();
 }
 break;
 case 'agents-management':
 if (typeof loadMarkdownAgents === 'function') {
 loadMarkdownAgents();
 }
 break;
 }
 if (pageId !== 'tasks' && typeof cleanupTasksPage === 'function') {
 cleanupTasksPage();
 }
}
document.addEventListener('DOMContentLoaded', function() {
 initRouter();
 initSidebarState();
 window.addEventListener('hashchange', function() {
 const hash = window.location.hash.slice(1);
 const hashParts = hash.split('?');
 const pageId = hashParts[0];
 
 if (pageId && ['chat', 'tasks', 'vulnerabilities', 'webshell', 'chat-files', 'mcp-monitor', 'mcp-management', 'knowledge-management', 'knowledge-retrieval-logs', 'roles-management', 'skills-monitor', 'skills-management', 'agents-management', 'settings'].includes(pageId)) {
 switchPage(pageId);
 if (pageId === 'chat' && hashParts.length > 1) {
 const params = new URLSearchParams(hashParts[1]);
 const conversationId = params.get('conversation');
 if (conversationId) {
 setTimeout(() => {
 if (typeof loadConversation === 'function') {
 loadConversation(conversationId);
 } else if (typeof window.loadConversation === 'function') {
 window.loadConversation(conversationId);
 } else {
 console.warn('loadConversation function not found');
 }
 }, 200);
 }
 }
 }
 });
 const hash = window.location.hash.slice(1);
 if (hash) {
 const hashParts = hash.split('?');
 const pageId = hashParts[0];
 if (pageId === 'chat' && hashParts.length > 1) {
 const params = new URLSearchParams(hashParts[1]);
 const conversationId = params.get('conversation');
 if (conversationId && typeof loadConversation === 'function') {
 setTimeout(() => {
 loadConversation(conversationId);
 }, 500);
 }
 }
 }
});
function toggleSidebar() {
 const sidebar = document.getElementById('main-sidebar');
 if (sidebar) {
 sidebar.classList.toggle('collapsed');
 const isCollapsed = sidebar.classList.contains('collapsed');
 localStorage.setItem('sidebarCollapsed', isCollapsed ? 'true' : 'false');
 }
}
function initSidebarState() {
 const sidebar = document.getElementById('main-sidebar');
 if (sidebar) {
 const savedState = localStorage.getItem('sidebarCollapsed');
 if (savedState === 'true') {
 sidebar.classList.add('collapsed');
 }
 }
 initConversationSidebarState();
}
function toggleConversationSidebar() {
 const sidebar = document.getElementById('conversation-sidebar');
 if (sidebar) {
 sidebar.classList.toggle('collapsed');
 const isCollapsed = sidebar.classList.contains('collapsed');
 localStorage.setItem('conversationSidebarCollapsed', isCollapsed ? 'true' : 'false');
 }
}
function initConversationSidebarState() {
 const sidebar = document.getElementById('conversation-sidebar');
 if (sidebar) {
 const savedState = localStorage.getItem('conversationSidebarCollapsed');
 if (savedState === 'true') {
 sidebar.classList.add('collapsed');
 } else {
 sidebar.classList.remove('collapsed');
 }
 }
}
window.switchPage = switchPage;
window.toggleSubmenu = toggleSubmenu;
window.toggleSidebar = toggleSidebar;
window.toggleConversationSidebar = toggleConversationSidebar;
window.currentPage = function() { return currentPage; };

