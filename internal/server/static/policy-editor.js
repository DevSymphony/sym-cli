// Symphony Policy Editor - Client-side Logic

// ==================== State Management ====================
let appState = {
    currentUser: null,
    policy: {
        version: '1.0.0',
        rbac: { roles: {} },
        defaults: {},
        rules: []
    },
    users: [],
    templates: [],
    history: [],
    isDirty: false,
    filters: {
        ruleSearch: '',
        category: ''
    },
    settings: {
        policyPath: '',
        autoSave: false,
        confirmSave: true
    }
};

// ==================== API Calls ====================
const API = {
    async getMe() {
        const res = await fetch('/api/me');
        return await res.json();
    },

    async getRepoInfo() {
        const res = await fetch('/api/repo-info');
        return await res.json();
    },

    async getPolicy() {
        const res = await fetch('/api/policy');
        return await res.json();
    },

    async savePolicy(policy) {
        const res = await fetch('/api/policy', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(policy)
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    },

    async getRoles() {
        const res = await fetch('/api/roles');
        return await res.json();
    },

    async saveRoles(roles) {
        const res = await fetch('/api/roles', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(roles)
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    },

    async getUsers() {
        const res = await fetch('/api/users');
        return await res.json();
    },

    async getTemplates() {
        const res = await fetch('/api/policy/templates');
        return await res.json();
    },

    async getTemplate(name) {
        const res = await fetch(`/api/policy/templates/${name}`);
        if (!res.ok) {
            const errorText = await res.text();
            throw new Error(`Failed to load template: ${res.status} ${res.statusText} - ${errorText}`);
        }
        return await res.json();
    },

    async getHistory() {
        const res = await fetch('/api/policy/history');
        return await res.json();
    },

    async getPolicyPath() {
        const res = await fetch('/api/policy/path');
        return await res.json();
    },

    async setPolicyPath(path) {
        const res = await fetch('/api/policy/path', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ policyPath: path })
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    }
};

// ==================== UI Helpers ====================
function showToast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    const bgColor = {
        'success': 'bg-green-500',
        'error': 'bg-red-500',
        'info': 'bg-blue-500',
        'warning': 'bg-yellow-500'
    }[type] || 'bg-gray-500';

    toast.className = `toast ${bgColor} text-white px-6 py-3 rounded-lg shadow-lg mb-2`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function showModal(modalId) {
    document.getElementById(modalId).classList.remove('hidden');
}

function hideModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
}

function markDirty() {
    appState.isDirty = true;
    document.getElementById('save-btn').classList.add('ring-2', 'ring-yellow-400');
    document.getElementById('floating-save-btn').classList.add('ring-4', 'ring-yellow-400', 'animate-pulse');
}

// Get all available roles (only RBAC-defined roles)
function getAvailableRoles() {
    const rbacRoles = appState.policy.rbac?.roles ? Object.keys(appState.policy.rbac.roles) : [];
    return rbacRoles.sort();
}

const CATEGORY_COLORS = {
    'naming': 'border-gray-400',
    'formatting': 'border-gray-400',
    'error_handling': 'border-gray-400',
    'security': 'border-gray-400',
    'testing': 'border-gray-400',
    'documentation': 'border-gray-400',
    'dependency': 'border-gray-400',
    'commit': 'border-gray-400',
    'performance': 'border-gray-400',
    'custom': 'border-gray-400',
    'default': 'border-gray-400'
};

function getCategoryColorClass(category) {
    return CATEGORY_COLORS[category] || CATEGORY_COLORS['default'];
}

// ==================== User Management ====================
function renderUsers() {
    const container = document.getElementById('users-container');
    const searchTerm = document.getElementById('user-search').value.toLowerCase();

    const filteredUsers = appState.users.filter(u =>
        u.username.toLowerCase().includes(searchTerm)
    );

    if (filteredUsers.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">사용자가 없습니다</div>';
        return;
    }

    container.innerHTML = filteredUsers.map(user => {
        // Get all available roles from RBAC + default roles
        const availableRoles = getAvailableRoles();
        const roleOptions = availableRoles.map(role =>
            `<option value="${role}" ${user.role === role ? 'selected' : ''}>${role.charAt(0).toUpperCase() + role.slice(1)}</option>`
        ).join('');

        return `
        <div class="flex items-center justify-between p-3 bg-gray-50 rounded-md border border-gray-200 hover:bg-gray-100">
            <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-full bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center text-white font-bold text-sm">
                    ${user.username.charAt(0).toUpperCase()}
                </div>
                <div>
                    <div class="font-medium text-slate-800">${user.username}</div>
                    <div class="text-xs text-slate-500">GitHub ID</div>
                </div>
            </div>
            <div class="flex items-center gap-2">
                <select class="user-role-select px-3 py-1 border border-gray-300 rounded-md text-sm" data-username="${user.username}">
                    ${roleOptions}
                </select>
                <button class="delete-user-btn px-2 py-1 text-red-500 hover:bg-red-50 rounded" data-username="${user.username}">&times;</button>
            </div>
        </div>
        `;
    }).join('');

    // Attach event listeners
    document.querySelectorAll('.user-role-select').forEach(select => {
        select.addEventListener('change', handleUserRoleChange);
    });

    document.querySelectorAll('.delete-user-btn').forEach(btn => {
        btn.addEventListener('click', handleDeleteUser);
    });

    // Apply permissions to dynamically rendered elements
    if (appState.currentUser?.permissions) {
        applyPermissions();
    }
}

async function handleUserRoleChange(e) {
    const username = e.target.dataset.username;
    const newRole = e.target.value;

    const user = appState.users.find(u => u.username === username);
    if (user) {
        user.role = newRole;

        // If this is the current user, update the badge immediately
        if (username === appState.currentUser.username) {
            appState.currentUser.role = newRole;
            updateUserRoleBadge(newRole);
        }

        await syncUsersToRoles();
        showToast(`${username}의 역할이 ${newRole}로 변경되었습니다`);
        markDirty();
    }
}

async function handleDeleteUser(e) {
    const username = e.target.dataset.username;

    // Safety check: Ensure at least one policy editor remains
    const userToDelete = appState.users.find(u => u.username === username);
    if (!userToDelete) return;

    // Check if this user has policy edit permission
    const userRole = appState.policy.rbac?.roles?.[userToDelete.role];
    if (userRole && userRole.canEditPolicy) {
        // Count how many users with policy edit permission exist
        const policyEditors = appState.users.filter(u => {
            const role = appState.policy.rbac?.roles?.[u.role];
            return role && role.canEditPolicy;
        });

        // If this is the last policy editor, prevent deletion
        if (policyEditors.length === 1) {
            alert(`❌ ${username}은(는) 유일한 정책 편집자입니다.\n\n최소 한 명의 정책 편집자는 남아있어야 합니다.\n먼저 다른 사용자에게 정책 편집 권한을 부여한 후 삭제해주세요.`);
            return;
        }
    }

    if (!confirm(`${username}을(를) 삭제하시겠습니까?`)) return;

    appState.users = appState.users.filter(u => u.username !== username);
    await syncUsersToRoles();
    renderUsers();
    showToast(`${username}이(가) 삭제되었습니다`);
    markDirty();
}

async function handleAddUser() {
    const username = prompt('새 사용자의 GitHub ID를 입력하세요:');
    if (!username || !username.trim()) return;

    const trimmedUsername = username.trim();

    if (appState.users.some(u => u.username === trimmedUsername)) {
        showToast('이미 존재하는 사용자입니다', 'error');
        return;
    }

    appState.users.push({ username: trimmedUsername, role: 'viewer' });
    await syncUsersToRoles();
    renderUsers();
    showToast(`${trimmedUsername}이(가) 추가되었습니다`);
    markDirty();
}

async function syncUsersToRoles() {
    const roles = { admin: [], developer: [], viewer: [] };

    appState.users.forEach(user => {
        if (roles[user.role]) {
            roles[user.role].push(user.username);
        }
    });

    // Update policy RBAC if needed
    // (This function updates appState.users based on roles.json)
}

async function loadUsersFromRoles() {
    try {
        const rolesData = await API.getRoles();
        const users = [];

        // Load all roles dynamically, not just admin/developer/viewer
        Object.entries(rolesData).forEach(([role, usernames]) => {
            if (Array.isArray(usernames)) {
                usernames.forEach(username => {
                    users.push({ username, role });
                });
            }
        });

        appState.users = users;
        renderUsers();
    } catch (error) {
        console.error('Failed to load users:', error);
        showToast('사용자 목록을 불러오는데 실패했습니다', 'error');
    }
}

// ==================== Rules Management ====================
function renderRules() {
    const container = document.getElementById('rules-container');
    const searchTerm = appState.filters.ruleSearch.toLowerCase();
    const categoryFilter = appState.filters.category;

    let filteredRules = appState.policy.rules.filter(rule => {
        const matchesSearch = !searchTerm ||
            rule.say.toLowerCase().includes(searchTerm) ||
            (rule.category && rule.category.toLowerCase().includes(searchTerm));

        const matchesCategory = !categoryFilter || rule.category === categoryFilter;

        return matchesSearch && matchesCategory;
    });

    document.getElementById('total-rules-count').textContent = appState.policy.rules.length;
    document.getElementById('filtered-rules-count').textContent = filteredRules.length;

    if (filteredRules.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-8">규칙이 없습니다</div>';
        return;
    }

    container.innerHTML = filteredRules.map((rule, index) => createRuleElement(rule, index)).join('');

    // Attach event listeners
    document.querySelectorAll('.delete-rule-btn').forEach(btn => {
        btn.addEventListener('click', handleDeleteRule);
    });

    document.querySelectorAll('.say-input').forEach(input => {
        input.addEventListener('input', handleRuleUpdate);
    });

    document.querySelectorAll('.category-select').forEach(select => {
        select.addEventListener('change', handleRuleUpdate);
    });

    document.querySelectorAll('.languages-input').forEach(input => {
        input.addEventListener('input', handleRuleUpdate);
    });

    document.querySelectorAll('.example-input').forEach(textarea => {
        textarea.addEventListener('input', handleRuleUpdate);
    });

    // Apply permissions to dynamically rendered elements
    if (appState.currentUser?.permissions) {
        applyPermissions();
    }
}

function createRuleElement(rule, index) {
    const actualIndex = appState.policy.rules.findIndex(r => r.no === rule.no);

    return `
        <details class="rule-details bg-white rounded-lg border border-l-4 ${getCategoryColorClass(rule.category)} transition-shadow hover:shadow-md" data-rule-no="${rule.no}">
            <summary class="p-4 cursor-pointer font-semibold flex justify-between items-center text-slate-800">
                <div class="flex items-center overflow-hidden min-w-0">
                    <span class="font-mono text-slate-400 mr-3">${rule.no}.</span>
                    <span class="rule-summary-text truncate">${rule.say || '새 규칙 (내용을 입력하세요)'}</span>
                </div>
                <button type="button" class="delete-rule-btn ml-4 text-gray-500 hover:text-red-500 text-xl font-bold flex-shrink-0" data-rule-no="${rule.no}">&times;</button>
            </summary>
            <div class="p-6 border-t border-gray-200">
                <div class="space-y-4">
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-600 mb-1">자연어 규칙 (필수)</label>
                        <input type="text" value="${rule.say || ''}" placeholder="예: 한 줄은 120자 이하" class="say-input w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-no="${rule.no}">
                    </div>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-600 mb-1">카테고리</label>
                            <select class="category-select w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-no="${rule.no}">
                                <option value="">선택 안함</option>
                                ${Object.keys(CATEGORY_COLORS).filter(c => c !== 'default').map(cat => `
                                    <option value="${cat}" ${rule.category === cat ? 'selected' : ''}>${cat}</option>
                                `).join('')}
                            </select>
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-600 mb-1">대상 언어</label>
                            <input type="text" value="${(rule.languages || []).join(', ')}" placeholder="javascript, python" class="languages-input w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-no="${rule.no}">
                        </div>
                    </div>
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-600 mb-1">예시 코드 (선택사항)</label>
                        <textarea class="example-input w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 font-mono" rows="4" placeholder="// 좋은 예:\nconst userName = 'John';\n\n// 나쁜 예:\nconst user_name = 'John';" data-rule-no="${rule.no}">${rule.example || ''}</textarea>
                    </div>
                </div>
            </div>
        </details>
    `;
}

function handleRuleUpdate(e) {
    const ruleNo = parseInt(e.target.dataset.ruleNo);
    const rule = appState.policy.rules.find(r => r.no === ruleNo);
    if (!rule) return;

    const ruleElement = document.querySelector(`.rule-details[data-rule-no="${ruleNo}"]`);

    if (e.target.classList.contains('say-input')) {
        rule.say = e.target.value.trim();
        ruleElement.querySelector('.rule-summary-text').textContent = rule.say || '새 규칙 (내용을 입력하세요)';
    } else if (e.target.classList.contains('category-select')) {
        rule.category = e.target.value;
        ruleElement.className = `rule-details bg-white rounded-lg border border-l-4 ${getCategoryColorClass(rule.category)} transition-shadow hover:shadow-md`;
    } else if (e.target.classList.contains('languages-input')) {
        const languagesStr = e.target.value.trim();
        rule.languages = languagesStr ? languagesStr.split(',').map(s => s.trim()).filter(Boolean) : [];
    } else if (e.target.classList.contains('example-input')) {
        rule.example = e.target.value.trim() || undefined; // Store undefined if empty
    }

    markDirty();
}

function handleDeleteRule(e) {
    e.preventDefault();
    const ruleNo = parseInt(e.target.dataset.ruleNo);

    if (!confirm('이 규칙을 정말 삭제하시겠습니까?')) return;

    appState.policy.rules = appState.policy.rules.filter(r => r.no !== ruleNo);

    // Renumber rules
    appState.policy.rules.forEach((rule, index) => {
        rule.no = index + 1;
    });

    renderRules();
    showToast('규칙이 삭제되었습니다');
    markDirty();
}

function handleAddRule() {
    const newNo = appState.policy.rules.length + 1;
    const newRule = { no: newNo, say: '', category: '', languages: [], example: '' };
    appState.policy.rules.push(newRule);
    renderRules();

    // Open the new rule details
    setTimeout(() => {
        const newRuleElement = document.querySelector(`.rule-details[data-rule-no="${newNo}"]`);
        if (newRuleElement) {
            newRuleElement.setAttribute('open', '');
            newRuleElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
            newRuleElement.querySelector('.say-input').focus();
        }
    }, 100);

    markDirty();
}

// ==================== RBAC Management ====================
function renderRBAC() {
    const container = document.getElementById('rbac-roles-container');
    const rbacRoles = appState.policy.rbac?.roles || {};

    if (Object.keys(rbacRoles).length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">정의된 RBAC 역할이 없습니다</div>';
        return;
    }

    container.innerHTML = Object.entries(rbacRoles).map(([roleName, roleData]) => `
        <div class="role-card bg-gray-50 border border-gray-200 p-4 rounded-lg" data-role-name="${roleName}">
            <div class="flex items-center justify-between mb-3">
                <input type="text" placeholder="역할 이름" value="${roleName}" class="role-name-input text-lg font-semibold bg-transparent focus:outline-none focus:ring-1 focus:ring-blue-500 rounded-md px-2 py-1 -ml-2 w-full">
                <button type="button" class="delete-rbac-role-btn text-gray-400 hover:text-red-500 text-xl flex-shrink-0" data-role-name="${roleName}">&times;</button>
            </div>

            <!-- System Permissions -->
            <div class="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md">
                <div class="text-xs font-medium text-blue-800 mb-2">시스템 권한</div>
                <div class="space-y-2">
                    <label class="flex items-center space-x-2 text-sm text-slate-700 cursor-pointer">
                        <input type="checkbox" ${roleData.canEditPolicy ? 'checked' : ''} class="role-canEditPolicy-input h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500" data-role-name="${roleName}">
                        <span>정책 편집 권한</span>
                    </label>
                    <label class="flex items-center space-x-2 text-sm text-slate-700 cursor-pointer">
                        <input type="checkbox" ${roleData.canEditRoles ? 'checked' : ''} class="role-canEditRoles-input h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500" data-role-name="${roleName}">
                        <span>역할 편집 권한</span>
                    </label>
                </div>
            </div>

            <!-- File Permissions -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                    <label class="block text-xs font-medium text-slate-500 mb-1">쓰기 허용 경로 (Allow)</label>
                    <textarea rows="3" placeholder="src/public/**" class="role-allowWrite-input w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-xs focus:ring-2 focus:ring-blue-500" data-role-name="${roleName}">${(roleData.allowWrite || []).join('\n')}</textarea>
                </div>
                <div>
                    <label class="block text-xs font-medium text-slate-500 mb-1">쓰기 금지 경로 (Deny)</label>
                    <textarea rows="3" placeholder="src/secure/**" class="role-denyWrite-input w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-xs focus:ring-2 focus:ring-blue-500" data-role-name="${roleName}">${(roleData.denyWrite || []).join('\n')}</textarea>
                </div>
            </div>
        </div>
    `).join('');

    // Attach event listeners
    document.querySelectorAll('.role-name-input').forEach(input => {
        input.addEventListener('change', handleRBACUpdate);
    });

    document.querySelectorAll('.role-allowWrite-input, .role-denyWrite-input').forEach(textarea => {
        textarea.addEventListener('input', handleRBACUpdate);
    });

    document.querySelectorAll('.role-canEditPolicy-input, .role-canEditRoles-input').forEach(checkbox => {
        checkbox.addEventListener('change', handleRBACUpdate);
    });

    document.querySelectorAll('.delete-rbac-role-btn').forEach(btn => {
        btn.addEventListener('click', handleDeleteRBACRole);
    });

    // Update disabled state for policy edit checkboxes
    updatePolicyEditCheckboxStates();

    // Apply permissions to dynamically rendered elements
    if (appState.currentUser?.permissions) {
        applyPermissions();
    }
}

// Update the disabled state of canEditPolicy checkboxes
// If only one role has canEditPolicy=true, disable that checkbox
function updatePolicyEditCheckboxStates() {
    const rbacRoles = appState.policy.rbac?.roles || {};
    const rolesWithEditPolicy = Object.entries(rbacRoles).filter(([_, role]) => role.canEditPolicy);

    // If only one role has policy edit permission, disable its checkbox
    if (rolesWithEditPolicy.length === 1) {
        const [singleRoleName, _] = rolesWithEditPolicy[0];
        document.querySelectorAll('.role-canEditPolicy-input').forEach(checkbox => {
            const roleName = checkbox.dataset.roleName;
            if (roleName === singleRoleName) {
                checkbox.disabled = true;
                checkbox.title = '최소 하나의 역할은 정책 편집 권한을 가져야 합니다';
            } else {
                checkbox.disabled = false;
                checkbox.title = '';
            }
        });
    } else {
        // Enable all checkboxes
        document.querySelectorAll('.role-canEditPolicy-input').forEach(checkbox => {
            checkbox.disabled = false;
            checkbox.title = '';
        });
    }
}

function handleRBACUpdate(e) {
    const oldRoleName = e.target.dataset.roleName;
    const roleCard = e.target.closest('.role-card');

    if (!appState.policy.rbac) appState.policy.rbac = { roles: {} };

    if (e.target.classList.contains('role-name-input')) {
        const newRoleName = e.target.value.trim();
        if (newRoleName && newRoleName !== oldRoleName) {
            // Check if new role name already exists
            if (appState.policy.rbac.roles[newRoleName]) {
                showToast(`역할 이름 "${newRoleName}"은(는) 이미 존재합니다`, 'error');
                e.target.value = oldRoleName; // Revert
                return;
            }

            // Update RBAC roles
            appState.policy.rbac.roles[newRoleName] = appState.policy.rbac.roles[oldRoleName];
            delete appState.policy.rbac.roles[oldRoleName];

            // Update all users with this role
            appState.users.forEach(user => {
                if (user.role === oldRoleName) {
                    user.role = newRoleName;
                }
            });

            roleCard.dataset.roleName = newRoleName;
            // Update all related elements
            roleCard.querySelectorAll('[data-role-name]').forEach(el => {
                el.dataset.roleName = newRoleName;
            });
            renderUsers(); // Update user role dropdowns when role name changes
            showToast(`역할 이름이 "${oldRoleName}"에서 "${newRoleName}"(으)로 변경되었습니다`);
        }
    } else {
        const role = appState.policy.rbac.roles[oldRoleName];
        if (!role) return;

        if (e.target.classList.contains('role-allowWrite-input')) {
            role.allowWrite = e.target.value.split('\n').map(s => s.trim()).filter(Boolean);
        } else if (e.target.classList.contains('role-denyWrite-input')) {
            role.denyWrite = e.target.value.split('\n').map(s => s.trim()).filter(Boolean);
        } else if (e.target.classList.contains('role-canEditPolicy-input')) {
            role.canEditPolicy = e.target.checked;
            // Update checkbox states when policy edit permission changes
            updatePolicyEditCheckboxStates();
        } else if (e.target.classList.contains('role-canEditRoles-input')) {
            role.canEditRoles = e.target.checked;
        }
    }

    markDirty();
}

function handleDeleteRBACRole(e) {
    const roleName = e.target.dataset.roleName;

    // Check if any users have this role
    const usersWithRole = appState.users.filter(u => u.role === roleName);
    if (usersWithRole.length > 0) {
        const usernames = usersWithRole.map(u => u.username).join(', ');
        alert(`❌ 이 역할을 사용 중인 사용자가 있어 삭제할 수 없습니다.\n\n사용자: ${usernames}\n\n먼저 해당 사용자들의 역할을 변경한 후 삭제해주세요.`);
        return;
    }

    if (!confirm(`${roleName} 역할을 삭제하시겠습니까?`)) return;

    delete appState.policy.rbac.roles[roleName];
    renderRBAC();
    renderUsers(); // Update user role dropdowns
    showToast(`${roleName} 역할이 삭제되었습니다`);
    markDirty();
}

function handleAddRBACRole() {
    const roleName = prompt('새 RBAC 역할 이름을 입력하세요:');
    if (!roleName || !roleName.trim()) return;

    if (!appState.policy.rbac) appState.policy.rbac = { roles: {} };

    if (appState.policy.rbac.roles[roleName.trim()]) {
        showToast('이미 존재하는 역할입니다', 'error');
        return;
    }

    appState.policy.rbac.roles[roleName.trim()] = {
        allowWrite: [],
        denyWrite: ['**/*'], // Default: deny all writes (read-only)
        canEditPolicy: false, // Default: no policy edit permission
        canEditRoles: false   // Default: no roles edit permission
    };

    renderRBAC();
    renderUsers(); // Update user role dropdowns
    showToast(`${roleName.trim()} 역할이 추가되었습니다 (기본: 읽기 전용)`);
    markDirty();
}

// ==================== Template Management ====================
async function loadTemplates() {
    try {
        appState.templates = await API.getTemplates();
        renderTemplates();
    } catch (error) {
        console.error('Failed to load templates:', error);
        showToast('템플릿을 불러오는데 실패했습니다', 'error');
    }
}

function renderTemplates() {
    const container = document.getElementById('templates-list');

    if (appState.templates.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">사용 가능한 템플릿이 없습니다</div>';
        return;
    }

    container.innerHTML = appState.templates.map(template => `
        <div class="template-item p-4 border border-gray-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 cursor-pointer transition-all" data-template-name="${template.name}">
            <div class="font-semibold text-slate-800 mb-1 flex items-center gap-2">
                <img src="/icons/template.svg" alt="Template" class="w-5 h-5">
                ${template.framework || template.language}
            </div>
            <div class="text-sm text-slate-600">${template.description}</div>
        </div>
    `).join('');

    document.querySelectorAll('.template-item').forEach(item => {
        item.addEventListener('click', handleApplyTemplate);
    });
}

async function handleApplyTemplate(e) {
    const templateName = e.currentTarget.dataset.templateName;
    console.log('Applying template:', templateName);

    if (!confirm('템플릿을 적용하면 현재 정책이 덮어써집니다. 계속하시겠습니까?')) return;

    try {
        console.log('Fetching template from API:', `/api/policy/templates/${templateName}`);
        const template = await API.getTemplate(templateName);
        console.log('Template loaded successfully:', template);

        // Validate template structure
        if (!template || typeof template !== 'object') {
            throw new Error('Invalid template format received');
        }

        // Apply template to policy (preserve existing RBAC)
        const currentRBAC = appState.policy.rbac; // Preserve existing RBAC
        appState.policy = {
            version: template.version || '1.0.0',
            rbac: currentRBAC || { roles: {} }, // Keep current RBAC, don't use template's RBAC
            defaults: template.defaults || {},
            rules: template.rules || []
        };

        console.log('Template applied to appState (RBAC preserved):', appState.policy);
        renderAll();
        hideModal('template-modal');
        showToast('템플릿이 적용되었습니다 (RBAC 유지됨)');
        markDirty();
    } catch (error) {
        console.error('Failed to apply template:', error);
        console.error('Error details:', error.message, error.stack);
        showToast(`템플릿 적용에 실패했습니다: ${error.message}`, 'error');
    }
}

// ==================== History Management ====================
async function loadHistory() {
    try {
        appState.history = await API.getHistory();
        renderHistory();
    } catch (error) {
        console.error('Failed to load history:', error);
        showToast('히스토리를 불러오는데 실패했습니다', 'error');
    }
}

function renderHistory() {
    const container = document.getElementById('history-list');

    if (appState.history.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">변경 이력이 없습니다</div>';
        return;
    }

    container.innerHTML = appState.history.map(commit => {
        const date = new Date(commit.date);
        return `
            <div class="history-item p-4 border border-gray-200 rounded-lg">
                <div class="flex justify-between items-start mb-2">
                    <div class="font-semibold text-slate-800">${commit.message}</div>
                    <div class="text-xs text-slate-500">${date.toLocaleString('ko-KR')}</div>
                </div>
                <div class="text-sm text-slate-600">
                    <span class="font-mono text-xs text-blue-600">${commit.hash.substring(0, 7)}</span>
                    <span class="mx-2">•</span>
                    <span>${commit.author} (${commit.email})</span>
                </div>
            </div>
        `;
    }).join('');
}

// ==================== Save & Load ====================
async function savePolicy() {
    if (appState.settings.confirmSave) {
        if (!confirm('정책을 저장하시겠습니까?')) return;
    }

    try {
        // Sync current user role first (in case it was changed in the UI)
        const currentUserInList = appState.users.find(u => u.username === appState.currentUser.username);
        if (currentUserInList) {
            appState.currentUser.role = currentUserInList.role;
        }

        // Validate: At least one role must have canEditPolicy permission
        if (appState.policy.rbac && appState.policy.rbac.roles) {
            const hasAtLeastOneEditor = Object.values(appState.policy.rbac.roles).some(role => role.canEditPolicy);
            if (!hasAtLeastOneEditor) {
                alert('❌ 최소 하나의 역할은 정책 편집 권한을 가져야 합니다.\n\n모든 역할에서 정책 편집 권한을 제거하면 아무도 정책을 수정할 수 없게 됩니다.');
                return;
            }
        }

        // Check if current user is losing policy edit privileges
        const currentUser = appState.users.find(u => u.username === appState.currentUser.username);
        const currentRole = appState.currentUser.role;
        const newRole = currentUser?.role;

        if (currentRole && newRole && currentRole !== newRole) {
            // Check if current user is losing policy edit permission
            const currentRoleData = appState.policy.rbac?.roles?.[currentRole];
            const newRoleData = appState.policy.rbac?.roles?.[newRole];

            if (currentRoleData?.canEditPolicy && !newRoleData?.canEditPolicy) {
                const confirmLoss = confirm(
                    `⚠️ 경고: 자신의 정책 편집 권한을 제거하려고 합니다.\n\n` +
                    `현재 역할: ${currentRole}\n` +
                    `변경될 역할: ${newRole}\n\n` +
                    `정책 편집 권한을 잃으면 정책을 수정할 수 없게 됩니다.\n계속하시겠습니까?`
                );
                if (!confirmLoss) return;
            }
        }

        // Update from UI
        const defaultsLanguages = document.getElementById('defaults-languages').value.trim();
        if (defaultsLanguages) {
            appState.policy.defaults.languages = defaultsLanguages.split(',').map(s => s.trim()).filter(Boolean);
        }

        appState.policy.defaults.severity = document.getElementById('defaults-severity').value || undefined;
        appState.policy.defaults.autofix = document.getElementById('defaults-autofix').checked || undefined;

        // Collect roles from users
        const roles = {};
        appState.users.forEach(user => {
            if (!roles[user.role]) {
                roles[user.role] = [];
            }
            roles[user.role].push(user.username);
        });

        // Ensure default roles exist even if empty
        if (!roles.admin) roles.admin = [];
        if (!roles.developer) roles.developer = [];
        if (!roles.viewer) roles.viewer = [];

        console.log('[DEBUG] Saving roles:', roles);
        console.log('[DEBUG] Current user role:', appState.currentUser.role);
        console.log('[DEBUG] Policy RBAC roles:', Object.keys(appState.policy.rbac?.roles || {}));

        // IMPORTANT: Save policy FIRST (so RBAC roles are defined)
        // Then save roles (so permission checks can find the role in RBAC)
        await API.savePolicy(appState.policy);
        await API.saveRoles(roles);

        // Update current user info
        updateUserInfo();

        appState.isDirty = false;
        document.getElementById('save-btn').classList.remove('ring-2', 'ring-yellow-400');
        document.getElementById('floating-save-btn').classList.remove('ring-4', 'ring-yellow-400', 'animate-pulse');
        showToast('정책이 성공적으로 저장되었습니다!');
    } catch (error) {
        console.error('Failed to save policy:', error);
        showToast('저장에 실패했습니다: ' + error.message, 'error');
    }
}

async function loadPolicy() {
    try {
        appState.policy = await API.getPolicy();
        await loadUsersFromRoles();
        renderAll();
        showToast('정책을 불러왔습니다');
    } catch (error) {
        console.error('Failed to load policy:', error);
        showToast('정책을 불러오는데 실패했습니다', 'error');
    }
}

// ==================== Settings ====================
async function loadSettings() {
    try {
        const pathData = await API.getPolicyPath();
        appState.settings.policyPath = pathData.policyPath || '';
        document.getElementById('policy-path-input').value = appState.settings.policyPath;

        // Load settings from localStorage
        const savedAutoSave = localStorage.getItem('autoSave');
        const savedConfirmSave = localStorage.getItem('confirmSave');

        if (savedAutoSave !== null) {
            appState.settings.autoSave = savedAutoSave === 'true';
        }
        if (savedConfirmSave !== null) {
            appState.settings.confirmSave = savedConfirmSave === 'true';
        }

        // Update checkboxes
        document.getElementById('auto-save-checkbox').checked = appState.settings.autoSave;
        document.getElementById('confirm-save-checkbox').checked = appState.settings.confirmSave;
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

async function saveSettings() {
    try {
        const newPath = document.getElementById('policy-path-input').value.trim();

        if (newPath && newPath !== appState.settings.policyPath) {
            await API.setPolicyPath(newPath);
            appState.settings.policyPath = newPath;
            showToast('설정이 저장되었습니다');
        }

        appState.settings.autoSave = document.getElementById('auto-save-checkbox').checked;
        appState.settings.confirmSave = document.getElementById('confirm-save-checkbox').checked;

        // Save to localStorage
        localStorage.setItem('autoSave', appState.settings.autoSave);
        localStorage.setItem('confirmSave', appState.settings.confirmSave);

        hideModal('settings-modal');
    } catch (error) {
        console.error('Failed to save settings:', error);
        showToast('설정 저장에 실패했습니다', 'error');
    }
}

// Update user role badge display
function updateUserRoleBadge(role) {
    const roleBadge = document.getElementById('user-role-badge');
    roleBadge.textContent = role;
    roleBadge.className = `badge text-xs px-2 py-1 rounded-full ${
        role === 'admin' ? 'bg-yellow-400 text-slate-900' :
        role === 'developer' ? 'bg-blue-500 text-white' :
        'bg-gray-400 text-white'
    }`;
}

// Update user info display in header
function updateUserInfo() {
    const currentUser = appState.users.find(u => u.username === appState.currentUser.username);
    if (currentUser) {
        // Update appState.currentUser.role
        appState.currentUser.role = currentUser.role;

        // Update badge display
        updateUserRoleBadge(currentUser.role);
    }
}

// ==================== Render All ====================
function renderAll() {
    // Defaults
    const defaults = appState.policy.defaults || {};
    document.getElementById('defaults-languages').value = (defaults.languages || []).join(', ');
    document.getElementById('defaults-severity').value = defaults.severity || '';
    document.getElementById('defaults-autofix').checked = defaults.autofix || false;

    // RBAC
    renderRBAC();

    // Rules
    renderRules();

    // Users (to update role dropdowns if RBAC roles changed)
    renderUsers();
}

// ==================== Permission-Based UI ====================
function applyPermissions() {
    const permissions = appState.currentUser?.permissions || {};
    const canEditPolicy = permissions.canEditPolicy === true; // Default to false if undefined (secure default)
    const canEditRoles = permissions.canEditRoles === true;   // Default to false if undefined (secure default)

    console.log('[Permissions] canEditPolicy:', canEditPolicy, 'canEditRoles:', canEditRoles);

    // When canEditPolicy = false: Read-only policy UI
    if (!canEditPolicy) {
        // Hide save buttons
        document.getElementById('save-btn')?.classList.add('hidden');
        document.getElementById('floating-save-btn')?.classList.add('hidden');

        // Hide template button
        document.getElementById('template-btn')?.classList.add('hidden');

        // Disable RBAC inputs and hide add/delete buttons
        document.querySelectorAll('.role-name-input, .role-allowWrite-input, .role-denyWrite-input').forEach(el => {
            el.disabled = true;
            el.classList.add('bg-gray-200', 'cursor-not-allowed');
        });
        document.querySelectorAll('.role-canEditPolicy-input, .role-canEditRoles-input').forEach(el => {
            el.disabled = true;
            el.classList.add('cursor-not-allowed');
        });
        document.querySelectorAll('.delete-rbac-role-btn').forEach(el => el.classList.add('hidden'));
        document.getElementById('add-role-btn')?.classList.add('hidden');

        // Disable defaults inputs
        document.getElementById('defaults-languages').disabled = true;
        document.getElementById('defaults-languages').classList.add('bg-gray-200', 'cursor-not-allowed');
        document.getElementById('defaults-severity').disabled = true;
        document.getElementById('defaults-severity').classList.add('bg-gray-200', 'cursor-not-allowed');
        document.getElementById('defaults-autofix').disabled = true;
        document.getElementById('defaults-autofix').classList.add('cursor-not-allowed');

        // Hide rule add/edit/delete buttons
        document.getElementById('add-rule-btn')?.classList.add('hidden');
        document.getElementById('add-rule-btn-bottom')?.classList.add('hidden');
        document.querySelectorAll('.delete-rule-btn').forEach(el => el.classList.add('hidden'));

        // Disable rule inputs
        document.querySelectorAll('.say-input, .category-select, .languages-input, .example-input').forEach(el => {
            el.disabled = true;
            el.classList.add('bg-gray-200', 'cursor-not-allowed');
        });
    }

    // When canEditRoles = false: Read-only user management
    if (!canEditRoles) {
        // Hide "Add User" button
        document.getElementById('add-user-btn')?.classList.add('hidden');

        // Hide user delete buttons
        document.querySelectorAll('.delete-user-btn').forEach(el => el.classList.add('hidden'));

        // Disable role dropdown
        document.querySelectorAll('.user-role-select').forEach(el => {
            el.disabled = true;
            el.classList.add('bg-gray-200', 'cursor-not-allowed');
        });
    }

    // Add read-only badge if any permission is missing
    if (!canEditPolicy || !canEditRoles) {
        const roleBadgeContainer = document.getElementById('user-role-badge').parentElement;
        if (!document.getElementById('readonly-badge')) {
            const readonlyBadge = document.createElement('span');
            readonlyBadge.id = 'readonly-badge';
            readonlyBadge.className = 'badge text-xs px-2 py-1 rounded-full bg-gray-500 text-white flex items-center gap-1';
            readonlyBadge.innerHTML = `
                <img src="/icons/lock.svg" alt="Lock" class="w-3 h-3">
                읽기 전용
            `;
            readonlyBadge.title = !canEditPolicy && !canEditRoles ? '정책과 역할 모두 읽기 전용입니다' :
                                  !canEditPolicy ? '정책이 읽기 전용입니다' :
                                  '역할이 읽기 전용입니다';
            roleBadgeContainer.appendChild(readonlyBadge);
        }
    }
}

// ==================== Initialize ====================
async function init() {
    try {
        // Load current user
        appState.currentUser = await API.getMe();
        document.getElementById('user-name').textContent = appState.currentUser.username;
        document.getElementById('user-avatar').textContent = appState.currentUser.username.charAt(0).toUpperCase();

        const roleBadge = document.getElementById('user-role-badge');
        roleBadge.textContent = appState.currentUser.role;
        roleBadge.className = `badge text-xs px-2 py-1 rounded-full ${
            appState.currentUser.role === 'admin' ? 'bg-yellow-400 text-slate-900' :
            appState.currentUser.role === 'developer' ? 'bg-blue-500 text-white' :
            'bg-gray-400 text-white'
        }`;

        // Load repo info
        const repoInfo = await API.getRepoInfo();
        document.getElementById('repo-info').textContent = `📁 ${repoInfo.owner}/${repoInfo.repo}`;

        // Load policy and users
        await loadPolicy();
        await loadSettings();
        await loadTemplates();

        // Apply permissions after loading all data
        applyPermissions();

    } catch (error) {
        console.error('Failed to initialize:', error);
        showToast('초기화에 실패했습니다', 'error');
    }
}

// ==================== Event Listeners ====================
document.addEventListener('DOMContentLoaded', () => {
    // Initialize
    init();

    // Auto-save timer (30 seconds)
    let autoSaveTimer = null;
    function startAutoSave() {
        if (autoSaveTimer) clearInterval(autoSaveTimer);
        autoSaveTimer = setInterval(async () => {
            if (appState.settings.autoSave && appState.isDirty) {
                try {
                    // Temporarily disable confirmation for auto-save
                    const originalConfirmSave = appState.settings.confirmSave;
                    appState.settings.confirmSave = false;

                    await savePolicy();
                    showToast('자동 저장되었습니다', 'info');

                    // Restore confirmation setting
                    appState.settings.confirmSave = originalConfirmSave;
                } catch (error) {
                    console.error('Auto-save failed:', error);
                    showToast('자동 저장 실패: ' + error.message, 'error');
                }
            }
        }, 30000); // 30 seconds
    }
    startAutoSave();

    // Save button
    document.getElementById('save-btn').addEventListener('click', savePolicy);

    // Floating save button
    document.getElementById('floating-save-btn').addEventListener('click', savePolicy);

    // Reload button
    document.getElementById('reload-btn').addEventListener('click', () => {
        if (appState.isDirty && !confirm('저장하지 않은 변경사항이 있습니다. 계속하시겠습니까?')) return;
        loadPolicy();
    });

    // Template button
    document.getElementById('template-btn').addEventListener('click', () => {
        showModal('template-modal');
    });

    // History button
    document.getElementById('history-btn').addEventListener('click', async () => {
        await loadHistory();
        showModal('history-modal');
    });

    // User management
    document.getElementById('add-user-btn').addEventListener('click', handleAddUser);
    document.getElementById('user-search').addEventListener('input', () => renderUsers());

    // Rules management
    document.getElementById('add-rule-btn').addEventListener('click', handleAddRule);
    document.getElementById('add-rule-btn-bottom').addEventListener('click', handleAddRule);
    document.getElementById('rule-search').addEventListener('input', (e) => {
        appState.filters.ruleSearch = e.target.value;
        renderRules();
    });
    document.getElementById('category-filter').addEventListener('change', (e) => {
        appState.filters.category = e.target.value;
        renderRules();
    });

    // RBAC management
    document.getElementById('add-role-btn').addEventListener('click', handleAddRBACRole);

    // Settings checkboxes - save to localStorage on change
    document.getElementById('auto-save-checkbox').addEventListener('change', (e) => {
        appState.settings.autoSave = e.target.checked;
        localStorage.setItem('autoSave', appState.settings.autoSave);
        showToast(appState.settings.autoSave ? '자동 저장이 활성화되었습니다' : '자동 저장이 비활성화되었습니다', 'info');
    });
    document.getElementById('confirm-save-checkbox').addEventListener('change', (e) => {
        appState.settings.confirmSave = e.target.checked;
        localStorage.setItem('confirmSave', appState.settings.confirmSave);
    });

    // Modal close buttons
    document.getElementById('close-template-modal').addEventListener('click', () => hideModal('template-modal'));
    document.getElementById('close-history-modal').addEventListener('click', () => hideModal('history-modal'));

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey && e.key === 's') {
            e.preventDefault();
            savePolicy();
        }
    });

    // Warn before leaving if dirty
    window.addEventListener('beforeunload', (e) => {
        if (appState.isDirty) {
            e.preventDefault();
            e.returnValue = '';
        }
    });
});
