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
    availableRoles: [],
    templates: [],
    isDirty: false,
    filters: {
        ruleSearch: '',
        category: ''
    },
    settings: {
        policyPath: '',
        confirmSave: true
    }
};

// ==================== Role Color System ====================
// Hash-based color generation for consistent role colors
function getRoleColor(roleName) {
    // Preset colors for default roles
    const presetColors = {
        'admin': { bg: 'bg-purple-100', text: 'text-purple-700', border: 'border-purple-300', bgHex: '#f3e8ff' },
        'developer': { bg: 'bg-blue-100', text: 'text-blue-700', border: 'border-blue-300', bgHex: '#dbeafe' },
        'viewer': { bg: 'bg-gray-100', text: 'text-gray-700', border: 'border-gray-300', bgHex: '#f3f4f6' }
    };

    if (presetColors[roleName]) return presetColors[roleName];

    // Dynamic colors for custom roles - generate consistent color from role name hash
    const dynamicColors = [
        { bg: 'bg-green-100', text: 'text-green-700', border: 'border-green-300', bgHex: '#dcfce7' },
        { bg: 'bg-yellow-100', text: 'text-yellow-700', border: 'border-yellow-300', bgHex: '#fef9c3' },
        { bg: 'bg-red-100', text: 'text-red-700', border: 'border-red-300', bgHex: '#fee2e2' },
        { bg: 'bg-pink-100', text: 'text-pink-700', border: 'border-pink-300', bgHex: '#fce7f3' },
        { bg: 'bg-indigo-100', text: 'text-indigo-700', border: 'border-indigo-300', bgHex: '#e0e7ff' },
        { bg: 'bg-teal-100', text: 'text-teal-700', border: 'border-teal-300', bgHex: '#ccfbf1' },
        { bg: 'bg-orange-100', text: 'text-orange-700', border: 'border-orange-300', bgHex: '#ffedd5' },
    ];

    // Calculate hash from role name for consistent color assignment
    const hash = roleName.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
    return dynamicColors[hash % dynamicColors.length];
}

// Get permission badges based on permissions
function getPermissionBadges(permissions) {
    const badges = [];

    if (permissions?.canEditPolicy) {
        badges.push({ text: '정책 편집', icon: '/icons/edit.svg', bg: 'bg-green-100', textColor: 'text-green-700' });
    }
    if (permissions?.canEditRoles) {
        badges.push({ text: '역할 편집', icon: '/icons/users.svg', bg: 'bg-blue-100', textColor: 'text-blue-700' });
    }

    if (badges.length === 0) {
        badges.push({ text: '읽기 전용', icon: '/icons/lock.svg', bg: 'bg-gray-100', textColor: 'text-gray-600' });
    }

    return badges;
}

// Render permission badge HTML with icon
function renderPermissionBadge(badge) {
    return `<span class="text-xs font-medium px-2 py-1 rounded-full ${badge.bg} ${badge.textColor} flex items-center gap-1">
        <img src="${badge.icon}" alt="" class="w-3 h-3 opacity-70">${badge.text}
    </span>`;
}

// ==================== API Calls ====================
const API = {
    async getMe() {
        const res = await fetch('/api/me');
        return await res.json();
    },

    async getProjectInfo() {
        const res = await fetch('/api/project-info');
        return await res.json();
    },

    async getAvailableRoles() {
        const res = await fetch('/api/available-roles');
        return await res.json();
    },

    async selectRole(role) {
        const res = await fetch('/api/select-role', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ role })
        });
        if (!res.ok) throw new Error(await res.text());
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

    async getPolicyPath() {
        const res = await fetch('/api/policy/path');
        return await res.json();
    },

    async setPolicyPath(path) {
        console.log('Sending policy path to server:', path);
        const res = await fetch('/api/policy/path', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ policyPath: path })
        });
        if (!res.ok) throw new Error(await res.text());
        const result = await res.json();
        console.log('Server response:', result);
        return result;
    },

    async convertPolicy() {
        console.log('Requesting policy conversion...');
        const res = await fetch('/api/policy/convert', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        if (!res.ok) throw new Error(await res.text());
        const result = await res.json();
        console.log('Conversion result:', result);
        return result;
    },

    // Category API methods
    async getCategories() {
        const res = await fetch('/api/categories');
        return await res.json();
    },

    async addCategory(name, description) {
        const res = await fetch('/api/categories', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, description })
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    },

    async editCategory(name, newName, description) {
        const res = await fetch(`/api/categories/${encodeURIComponent(name)}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ new_name: newName, description })
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    },

    async deleteCategory(name) {
        const res = await fetch(`/api/categories/${encodeURIComponent(name)}`, {
            method: 'DELETE'
        });
        if (!res.ok) throw new Error(await res.text());
        return await res.json();
    },

    async importConventions(path, mode) {
        const res = await fetch('/api/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path, mode })
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

// ==================== Language Management ====================
function getAvailableLanguages() {
    return appState.policy.defaults?.languages || [];
}

function renderLanguageTags() {
    const container = document.getElementById('defaults-languages-tags');
    const languages = getAvailableLanguages();
    const defaultLanguage = appState.policy.defaults?.defaultLanguage || '';

    if (languages.length === 0) {
        container.innerHTML = '<span class="text-sm text-slate-400">등록된 언어가 없습니다</span>';
    } else {
        container.innerHTML = languages.map(lang => `
            <span class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm ${
                lang === defaultLanguage
                    ? 'bg-blue-100 text-blue-800 border-2 border-blue-500'
                    : 'bg-gray-100 text-gray-700 border border-gray-300'
            }">
                ${lang}
                ${lang === defaultLanguage ? '<span class="text-xs text-blue-600">(기본)</span>' : ''}
                <button type="button" class="remove-language-btn ml-1 text-gray-500 hover:text-red-500" data-language="${lang}">&times;</button>
            </span>
        `).join('');

        // Attach remove event listeners
        document.querySelectorAll('.remove-language-btn').forEach(btn => {
            btn.addEventListener('click', handleRemoveLanguage);
        });
    }

    // Update default language dropdown
    updateDefaultLanguageDropdown();
}

function updateDefaultLanguageDropdown() {
    const select = document.getElementById('defaults-default-language');
    const languages = getAvailableLanguages();
    const currentDefault = appState.policy.defaults?.defaultLanguage || '';

    select.innerHTML = '<option value="">선택 안함</option>' +
        languages.map(lang => `<option value="${lang}" ${lang === currentDefault ? 'selected' : ''}>${lang}</option>`).join('');
}

function handleAddLanguage() {
    const input = document.getElementById('defaults-language-input');
    const language = input.value.trim().toLowerCase();

    if (!language) {
        showToast('언어를 입력해주세요', 'warning');
        return;
    }

    if (!appState.policy.defaults) {
        appState.policy.defaults = {};
    }
    if (!appState.policy.defaults.languages) {
        appState.policy.defaults.languages = [];
    }

    if (appState.policy.defaults.languages.includes(language)) {
        showToast('이미 등록된 언어입니다', 'warning');
        return;
    }

    appState.policy.defaults.languages.push(language);
    input.value = '';

    renderLanguageTags();
    renderRules(); // Update rule language dropdowns
    showToast(`${language} 언어가 추가되었습니다`);
    markDirty();
}

function handleRemoveLanguage(e) {
    const language = e.target.dataset.language;

    if (!appState.policy.defaults?.languages) return;

    // Check if any rules use this language
    const rulesUsingLanguage = appState.policy.rules.filter(r =>
        r.languages && r.languages.includes(language)
    );

    if (rulesUsingLanguage.length > 0) {
        if (!confirm(`${rulesUsingLanguage.length}개의 규칙이 이 언어를 사용 중입니다.\n삭제하면 해당 규칙에서도 이 언어가 제거됩니다.\n계속하시겠습니까?`)) {
            return;
        }

        // Remove language from rules
        rulesUsingLanguage.forEach(rule => {
            rule.languages = rule.languages.filter(l => l !== language);
        });
    }

    // Remove from defaults
    appState.policy.defaults.languages = appState.policy.defaults.languages.filter(l => l !== language);

    // Clear default language if it was removed
    if (appState.policy.defaults.defaultLanguage === language) {
        appState.policy.defaults.defaultLanguage = '';
    }

    renderLanguageTags();
    renderRules(); // Update rule language dropdowns
    showToast(`${language} 언어가 삭제되었습니다`);
    markDirty();
}

function handleDefaultLanguageChange(e) {
    if (!appState.policy.defaults) {
        appState.policy.defaults = {};
    }
    appState.policy.defaults.defaultLanguage = e.target.value;
    renderLanguageTags(); // Update tag highlighting
    markDirty();
}

function addLanguagesToDefaults(languages) {
    if (!appState.policy.defaults) {
        appState.policy.defaults = {};
    }
    if (!appState.policy.defaults.languages) {
        appState.policy.defaults.languages = [];
    }

    let addedCount = 0;
    languages.forEach(lang => {
        const normalizedLang = lang.trim().toLowerCase();
        if (normalizedLang && !appState.policy.defaults.languages.includes(normalizedLang)) {
            appState.policy.defaults.languages.push(normalizedLang);
            addedCount++;
        }
    });

    return addedCount;
}

// ==================== Category Management ====================
function getAvailableCategories() {
    return appState.policy.category || [];
}

function getRulesCountForCategory(categoryName) {
    return appState.policy.rules.filter(rule => rule.category === categoryName).length;
}

function renderCategories() {
    const container = document.getElementById('categories-container');
    const countSpan = document.getElementById('category-count');
    const categories = getAvailableCategories();

    if (countSpan) {
        countSpan.textContent = categories.length;
    }

    if (categories.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">정의된 카테고리가 없습니다. 아래에서 새 카테고리를 추가하세요.</div>';
        updateCategoryFilter();
        return;
    }

    container.innerHTML = categories.map(cat => {
        const rulesCount = getRulesCountForCategory(cat.name);
        return `
            <div class="category-card bg-gray-50 border border-gray-200 p-4 rounded-lg" data-category-name="${cat.name}">
                <div class="category-view flex items-start justify-between">
                    <div class="flex-1">
                        <div class="flex items-center gap-2 mb-1">
                            <span class="font-semibold text-slate-800">${cat.name}</span>
                            <span class="text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700">${rulesCount} 규칙</span>
                        </div>
                        <p class="text-sm text-slate-600">${cat.description || ''}</p>
                    </div>
                    <div class="flex gap-2">
                        <button type="button" class="edit-category-btn px-3 py-1 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded" data-category-name="${cat.name}">편집</button>
                        <button type="button" class="delete-category-btn px-3 py-1 text-sm text-red-600 hover:text-red-800 hover:bg-red-50 rounded" data-category-name="${cat.name}"${rulesCount > 0 ? ' disabled title="규칙이 있는 카테고리는 삭제할 수 없습니다"' : ''}>삭제</button>
                    </div>
                </div>
                <div class="category-edit hidden">
                    <div class="grid grid-cols-[1fr_2fr_auto_auto] gap-3 items-end">
                        <div>
                            <label class="block text-xs font-medium text-slate-500 mb-1">이름</label>
                            <input type="text" class="edit-category-name w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" value="${cat.name}" data-original-name="${cat.name}">
                        </div>
                        <div>
                            <label class="block text-xs font-medium text-slate-500 mb-1">설명</label>
                            <input type="text" class="edit-category-description w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" value="${cat.description || ''}">
                        </div>
                        <button type="button" class="save-category-btn px-3 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-md text-sm font-medium" data-category-name="${cat.name}">저장</button>
                        <button type="button" class="cancel-edit-category-btn px-3 py-2 bg-gray-300 hover:bg-gray-400 text-gray-700 rounded-md text-sm font-medium" data-category-name="${cat.name}">취소</button>
                    </div>
                </div>
            </div>
        `;
    }).join('');

    // Attach event listeners
    document.querySelectorAll('.edit-category-btn').forEach(btn => {
        btn.addEventListener('click', handleEditCategoryClick);
    });

    document.querySelectorAll('.delete-category-btn').forEach(btn => {
        if (!btn.disabled) {
            btn.addEventListener('click', handleDeleteCategory);
        }
    });

    document.querySelectorAll('.save-category-btn').forEach(btn => {
        btn.addEventListener('click', handleSaveCategory);
    });

    document.querySelectorAll('.cancel-edit-category-btn').forEach(btn => {
        btn.addEventListener('click', handleCancelEditCategory);
    });

    // Update category filter dropdown
    updateCategoryFilter();

    // Apply permissions
    if (appState.currentUser?.permissions) {
        applyPermissions();
    }
}

function handleEditCategoryClick(e) {
    const categoryName = e.target.dataset.categoryName;
    const card = document.querySelector(`.category-card[data-category-name="${categoryName}"]`);
    if (!card) return;

    card.querySelector('.category-view').classList.add('hidden');
    card.querySelector('.category-edit').classList.remove('hidden');
}

function handleCancelEditCategory(e) {
    const categoryName = e.target.dataset.categoryName;
    const card = document.querySelector(`.category-card[data-category-name="${categoryName}"]`);
    if (!card) return;

    card.querySelector('.category-view').classList.remove('hidden');
    card.querySelector('.category-edit').classList.add('hidden');

    // Reset input values
    const category = getAvailableCategories().find(c => c.name === categoryName);
    if (category) {
        card.querySelector('.edit-category-name').value = category.name;
        card.querySelector('.edit-category-description').value = category.description || '';
    }
}

async function handleSaveCategory(e) {
    const originalName = e.target.dataset.categoryName;
    const card = document.querySelector(`.category-card[data-category-name="${originalName}"]`);
    if (!card) return;

    const newName = card.querySelector('.edit-category-name').value.trim();
    const newDescription = card.querySelector('.edit-category-description').value.trim();

    if (!newName) {
        showToast('카테고리 이름을 입력해주세요', 'warning');
        return;
    }

    try {
        const result = await API.editCategory(originalName, newName !== originalName ? newName : '', newDescription);

        // Update local state
        const categoryIndex = appState.policy.category.findIndex(c => c.name === originalName);
        if (categoryIndex !== -1) {
            appState.policy.category[categoryIndex].name = newName;
            appState.policy.category[categoryIndex].description = newDescription;

            // Update rule references if name changed
            if (newName !== originalName) {
                appState.policy.rules.forEach(rule => {
                    if (rule.category === originalName) {
                        rule.category = newName;
                    }
                });
            }
        }

        renderCategories();
        updateRuleCategorySelects(); // Update category dropdowns in rule cards
        renderRules(); // Update rules to reflect category changes
        showToast(result.message || '카테고리가 수정되었습니다');
    } catch (error) {
        console.error('Failed to edit category:', error);
        showToast('카테고리 수정에 실패했습니다: ' + error.message, 'error');
    }
}

async function handleAddCategory() {
    const nameInput = document.getElementById('new-category-name');
    const descInput = document.getElementById('new-category-description');
    const name = nameInput.value.trim();
    const description = descInput.value.trim();

    if (!name) {
        showToast('카테고리 이름을 입력해주세요', 'warning');
        nameInput.focus();
        return;
    }

    if (!description) {
        showToast('카테고리 설명을 입력해주세요', 'warning');
        descInput.focus();
        return;
    }

    // Check for duplicate locally
    if (getAvailableCategories().some(c => c.name === name)) {
        showToast(`카테고리 '${name}'이(가) 이미 존재합니다`, 'error');
        return;
    }

    try {
        await API.addCategory(name, description);

        // Update local state
        if (!appState.policy.category) {
            appState.policy.category = [];
        }
        appState.policy.category.push({ name, description });

        // Clear inputs
        nameInput.value = '';
        descInput.value = '';

        renderCategories();
        updateRuleCategorySelects(); // Update category dropdowns in rule cards
        showToast(`카테고리 '${name}'이(가) 추가되었습니다`);
    } catch (error) {
        console.error('Failed to add category:', error);
        showToast('카테고리 추가에 실패했습니다: ' + error.message, 'error');
    }
}

async function handleDeleteCategory(e) {
    const categoryName = e.target.dataset.categoryName;
    const rulesCount = getRulesCountForCategory(categoryName);

    if (rulesCount > 0) {
        showToast(`카테고리 '${categoryName}'은(는) ${rulesCount}개의 규칙에서 사용 중입니다. 먼저 규칙을 삭제하거나 다른 카테고리로 변경해주세요.`, 'error');
        return;
    }

    if (!confirm(`카테고리 '${categoryName}'을(를) 삭제하시겠습니까?`)) return;

    try {
        await API.deleteCategory(categoryName);

        // Update local state
        appState.policy.category = appState.policy.category.filter(c => c.name !== categoryName);

        renderCategories();
        updateRuleCategorySelects(); // Update category dropdowns in rule cards
        showToast(`카테고리 '${categoryName}'이(가) 삭제되었습니다`);
    } catch (error) {
        console.error('Failed to delete category:', error);
        showToast('카테고리 삭제에 실패했습니다: ' + error.message, 'error');
    }
}

function updateCategoryFilter() {
    const filterSelect = document.getElementById('category-filter');
    if (!filterSelect) return;

    const categories = getAvailableCategories();
    const currentValue = filterSelect.value;

    filterSelect.innerHTML = '<option value="">전체 카테고리</option>' +
        categories.map(cat => `<option value="${cat.name}" ${cat.name === currentValue ? 'selected' : ''}>${cat.name}</option>`).join('');
}

// Update all category dropdowns in rule cards
function updateRuleCategorySelects() {
    const categories = getAvailableCategories();
    document.querySelectorAll('.category-select').forEach(select => {
        const currentValue = select.value;
        select.innerHTML = '<option value="">선택 안함</option>' +
            categories.map(cat => `<option value="${cat.name}" ${cat.name === currentValue ? 'selected' : ''}>${cat.name}</option>`).join('');
    });
}

// ==================== Role Selection ====================
function renderRoleSelection() {
    const container = document.getElementById('role-selection-container');
    if (!container) return;

    const availableRoles = appState.availableRoles || [];
    const currentRole = appState.currentUser?.role || '';

    if (availableRoles.length === 0) {
        container.innerHTML = '<div class="text-center text-slate-500 py-4">사용 가능한 역할이 없습니다</div>';
        return;
    }

    container.innerHTML = availableRoles.map(role => {
        const isCurrentRole = role === currentRole;
        const roleConfig = appState.policy.rbac?.roles?.[role] || {};
        const roleColor = getRoleColor(role);
        const permBadges = getPermissionBadges({ canEditPolicy: roleConfig.canEditPolicy, canEditRoles: roleConfig.canEditRoles });
        const badgesHtml = permBadges.map(renderPermissionBadge).join('');

        return `
        <button class="role-select-btn p-4 text-left rounded-lg border-2 transition-all ${
            isCurrentRole
                ? `${roleColor.border} ${roleColor.bg} ring-2 ring-opacity-50`
                : 'border-gray-200 hover:border-gray-300 bg-white hover:bg-gray-50'
        }" data-role="${role}" style="${isCurrentRole ? `--tw-ring-color: ${roleColor.bgHex}` : ''}">
            <div class="flex items-center justify-between mb-2">
                <span class="font-semibold text-lg px-2 py-0.5 rounded ${roleColor.bg} ${roleColor.text}">${role}</span>
                ${isCurrentRole ? '<span class="text-xs bg-slate-700 text-white px-2 py-1 rounded">현재</span>' : ''}
            </div>
            <div class="flex gap-1 flex-wrap">${badgesHtml}</div>
        </button>
        `;
    }).join('');

    // Attach click handlers
    container.querySelectorAll('.role-select-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const selectedRole = e.currentTarget.dataset.role;
            if (selectedRole !== currentRole) {
                await selectRole(selectedRole);
            }
        });
    });
}

async function selectRole(role) {
    try {
        const result = await API.selectRole(role);

        // Update current user state
        appState.currentUser.role = result.role;
        appState.currentUser.permissions = result.permissions;

        // Update header UI
        updateUserRoleBadge(result.role);
        updatePermissionBadges(result.permissions);

        // Re-render all UI sections first
        renderRoleSelection();
        renderRBAC();
        renderRules();
        renderLanguageTags();

        // Apply permissions AFTER all renders so new elements get proper state
        applyPermissions();

        showToast(`역할이 '${result.role}'(으)로 변경되었습니다`);

    } catch (error) {
        console.error('Failed to select role:', error);
        showToast('역할 변경에 실패했습니다: ' + error.message, 'error');
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

    document.querySelectorAll('.language-select').forEach(select => {
        select.addEventListener('change', handleRuleUpdate);
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
    const availableLanguages = getAvailableLanguages();
    const ruleLanguage = rule.languages && rule.languages.length > 0 ? rule.languages[0] : '';

    return `
        <details class="rule-details bg-white rounded-lg border border-l-4 ${getCategoryColorClass(rule.category)} transition-shadow hover:shadow-md" data-rule-id="${rule.id}">
            <summary class="p-4 cursor-pointer font-semibold flex justify-between items-center text-slate-800">
                <div class="flex items-center overflow-hidden min-w-0">
                    <span class="font-mono text-slate-400 mr-3">${rule.id}.</span>
                    <span class="rule-summary-text truncate">${rule.say || '새 규칙 (내용을 입력하세요)'}</span>
                    ${ruleLanguage ? `<span class="ml-2 px-2 py-0.5 text-xs rounded bg-gray-200 text-gray-600">${ruleLanguage}</span>` : ''}
                </div>
                <button type="button" class="delete-rule-btn ml-4 text-gray-500 hover:text-red-500 text-xl font-bold flex-shrink-0" data-rule-id="${rule.id}">&times;</button>
            </summary>
            <div class="p-6 border-t border-gray-200">
                <div class="space-y-4">
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-600 mb-1">자연어 규칙 (필수)</label>
                        <input type="text" value="${rule.say || ''}" placeholder="예: 한 줄은 120자 이하" class="say-input w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-id="${rule.id}">
                    </div>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-600 mb-1">카테고리</label>
                            <select class="category-select w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-id="${rule.id}">
                                <option value="">선택 안함</option>
                                ${getAvailableCategories().map(cat => `
                                    <option value="${cat.name}" ${rule.category === cat.name ? 'selected' : ''}>${cat.name}</option>
                                `).join('')}
                            </select>
                            ${getAvailableCategories().length === 0 ? '<p class="text-xs text-amber-600 mt-1">카테고리 관리 섹션에서 카테고리를 먼저 추가해주세요.</p>' : ''}
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-600 mb-1">대상 언어</label>
                            <select class="language-select w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500" data-rule-id="${rule.id}">
                                <option value="">선택 안함</option>
                                ${availableLanguages.map(lang => `
                                    <option value="${lang}" ${ruleLanguage === lang ? 'selected' : ''}>${lang}</option>
                                `).join('')}
                            </select>
                            ${availableLanguages.length === 0 ? '<p class="text-xs text-amber-600 mt-1">전역 설정에서 언어를 먼저 추가해주세요.</p>' : ''}
                        </div>
                    </div>
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-600 mb-1">예시 코드 (선택사항)</label>
                        <textarea class="example-input w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 font-mono" rows="4" placeholder="// 좋은 예:\nconst userName = 'John';\n\n// 나쁜 예:\nconst user_name = 'John';" data-rule-id="${rule.id}">${rule.example || ''}</textarea>
                    </div>
                </div>
            </div>
        </details>
    `;
}

function handleRuleUpdate(e) {
    const ruleId = e.target.dataset.ruleId;
    const rule = appState.policy.rules.find(r => r.id === ruleId);
    if (!rule) return;

    const ruleElement = document.querySelector(`.rule-details[data-rule-id="${ruleId}"]`);

    if (e.target.classList.contains('say-input')) {
        rule.say = e.target.value.trim();
        ruleElement.querySelector('.rule-summary-text').textContent = rule.say || '새 규칙 (내용을 입력하세요)';
    } else if (e.target.classList.contains('category-select')) {
        rule.category = e.target.value;
        ruleElement.className = `rule-details bg-white rounded-lg border border-l-4 ${getCategoryColorClass(rule.category)} transition-shadow hover:shadow-md`;
    } else if (e.target.classList.contains('language-select')) {
        const selectedLanguage = e.target.value;
        rule.languages = selectedLanguage ? [selectedLanguage] : [];
        // Update the language badge in summary
        const summaryDiv = ruleElement.querySelector('summary .flex.items-center');
        const existingBadge = summaryDiv.querySelector('.bg-gray-200');
        if (existingBadge) existingBadge.remove();
        if (selectedLanguage) {
            const badge = document.createElement('span');
            badge.className = 'ml-2 px-2 py-0.5 text-xs rounded bg-gray-200 text-gray-600';
            badge.textContent = selectedLanguage;
            summaryDiv.appendChild(badge);
        }
    } else if (e.target.classList.contains('example-input')) {
        rule.example = e.target.value.trim() || undefined; // Store undefined if empty
    }

    markDirty();
}

function handleDeleteRule(e) {
    e.preventDefault();
    const ruleId = e.target.dataset.ruleId;

    if (!confirm('이 규칙을 정말 삭제하시겠습니까?')) return;

    appState.policy.rules = appState.policy.rules.filter(r => r.id !== ruleId);

    console.log('Rule deleted. Current rules count:', appState.policy.rules.length);

    renderRules();
    showToast('규칙이 삭제되었습니다');
    markDirty();
}

function handleAddRule() {
    // Generate a unique ID by finding the maximum existing ID and adding 1
    const maxId = appState.policy.rules.reduce((max, rule) => {
        const ruleId = parseInt(rule.id, 10);
        return isNaN(ruleId) ? max : Math.max(max, ruleId);
    }, 0);
    const newId = String(maxId + 1);
    // Use default language from global settings if available
    const defaultLanguage = appState.policy.defaults?.defaultLanguage || '';
    const newRule = {
        id: newId,
        say: '',
        category: '',
        languages: defaultLanguage ? [defaultLanguage] : [],
        example: ''
    };
    appState.policy.rules.push(newRule);

    console.log('Rule added. Current rules count:', appState.policy.rules.length);

    renderRules();

    // Open the new rule details
    setTimeout(() => {
        const newRuleElement = document.querySelector(`.rule-details[data-rule-id="${newId}"]`);
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

            roleCard.dataset.roleName = newRoleName;
            // Update all related elements
            roleCard.querySelectorAll('[data-role-name]').forEach(el => {
                el.dataset.roleName = newRoleName;
            });

            // Update available roles list
            const index = appState.availableRoles.indexOf(oldRoleName);
            if (index !== -1) {
                appState.availableRoles[index] = newRoleName;
            }
            renderRoleSelection();
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

    // Check if this is the currently selected role
    if (roleName === appState.currentUser?.role) {
        alert(`❌ 현재 선택된 역할은 삭제할 수 없습니다.\n\n다른 역할을 선택한 후 삭제해주세요.`);
        return;
    }

    if (!confirm(`${roleName} 역할을 삭제하시겠습니까?`)) return;

    delete appState.policy.rbac.roles[roleName];

    // Update available roles list
    const index = appState.availableRoles.indexOf(roleName);
    if (index !== -1) {
        appState.availableRoles.splice(index, 1);
    }

    renderRBAC();
    renderRoleSelection();
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

        // Preserve existing RBAC only (category is now overwritten from template)
        const currentRBAC = appState.policy.rbac;

        // Collect all languages from template (defaults and rules)
        const templateLanguages = new Set();

        // Add languages from template defaults
        if (template.defaults?.languages) {
            template.defaults.languages.forEach(lang => templateLanguages.add(lang.toLowerCase()));
        }

        // Add languages from template rules
        if (template.rules) {
            template.rules.forEach(rule => {
                if (rule.languages) {
                    rule.languages.forEach(lang => templateLanguages.add(lang.toLowerCase()));
                }
            });
        }

        // Determine default language for rules without explicit languages
        const languageArray = Array.from(templateLanguages);
        const templateDefaultLang = template.defaults?.defaultLanguage?.toLowerCase() ||
            (languageArray.length > 0 ? languageArray[0] : '');

        // Apply template to policy (category is overwritten from template)
        appState.policy = {
            version: template.version || '1.0.0',
            rbac: currentRBAC || { roles: {} },    // Keep current RBAC
            category: template.category || [],      // Use template's category (overwrite)
            defaults: {
                ...template.defaults,
                languages: languageArray, // Normalized languages
                defaultLanguage: templateDefaultLang
            },
            rules: (template.rules || []).map(rule => {
                // If rule has languages, use first one; otherwise use template default
                let ruleLanguage = '';
                if (rule.languages && rule.languages.length > 0) {
                    ruleLanguage = rule.languages[0].toLowerCase();
                } else if (templateDefaultLang) {
                    ruleLanguage = templateDefaultLang;
                }
                return {
                    ...rule,
                    languages: ruleLanguage ? [ruleLanguage] : []
                };
            })
        };

        console.log('Template applied to appState:', appState.policy);
        console.log('Languages from template:', Array.from(templateLanguages));
        console.log('Categories from template:', template.category?.length || 0);

        renderAll();
        hideModal('template-modal');

        const langCount = templateLanguages.size;
        const catCount = template.category?.length || 0;
        showToast(`템플릿이 적용되었습니다 (${langCount}개 언어, ${catCount}개 카테고리)`);
        markDirty();
    } catch (error) {
        console.error('Failed to apply template:', error);
        console.error('Error details:', error.message, error.stack);
        showToast(`템플릿 적용에 실패했습니다: ${error.message}`, 'error');
    }
}

// ==================== Import Management ====================
async function handleImport() {
    const pathInput = document.getElementById('import-file-path');
    const path = pathInput.value.trim();

    if (!path) {
        showToast('파일 경로를 입력해주세요', 'warning');
        pathInput.focus();
        return;
    }

    const mode = document.querySelector('input[name="import-mode"]:checked').value;

    if (mode === 'clear') {
        if (!confirm('Clear 모드는 기존 카테고리와 규칙을 모두 삭제합니다. 계속하시겠습니까?')) {
            return;
        }
    }

    try {
        showToast('Import 진행 중... LLM이 문서를 분석하고 있습니다', 'info');
        const result = await API.importConventions(path, mode);

        if (result.status === 'error') {
            throw new Error(result.error);
        }

        // Reload policy to reflect changes
        await loadPolicy();

        hideModal('import-modal');
        pathInput.value = '';

        // Build success message
        let msg = 'Import 완료! ';
        if (result.categoriesAdded?.length > 0) {
            msg += `${result.categoriesAdded.length}개 카테고리, `;
        }
        if (result.rulesAdded?.length > 0) {
            msg += `${result.rulesAdded.length}개 규칙 추가`;
        }
        if (result.warnings?.length > 0) {
            msg += ` (${result.warnings.length}개 경고)`;
        }

        showToast(msg, 'success');
        markDirty();

    } catch (error) {
        console.error('Import failed:', error);
        showToast('Import 실패: ' + error.message, 'error');
    }
}

// ==================== Save & Load ====================
async function savePolicy() {
    if (appState.settings.confirmSave) {
        if (!confirm('정책을 저장하시겠습니까?')) return;
    }

    try {
        // Validate: At least one role must have canEditPolicy permission
        if (appState.policy.rbac && appState.policy.rbac.roles) {
            const hasAtLeastOneEditor = Object.values(appState.policy.rbac.roles).some(role => role.canEditPolicy);
            if (!hasAtLeastOneEditor) {
                alert('❌ 최소 하나의 역할은 정책 편집 권한을 가져야 합니다.\n\n모든 역할에서 정책 편집 권한을 제거하면 아무도 정책을 수정할 수 없게 됩니다.');
                return;
            }
        }

        // Check if current role is losing policy edit privileges
        const currentRole = appState.currentUser.role;
        const currentRoleData = appState.policy.rbac?.roles?.[currentRole];

        if (currentRoleData && !currentRoleData.canEditPolicy) {
            const confirmLoss = confirm(
                `⚠️ 경고: 현재 역할(${currentRole})에서 정책 편집 권한을 제거하려고 합니다.\n\n` +
                `정책 편집 권한을 잃으면 정책을 수정할 수 없게 됩니다.\n계속하시겠습니까?`
            );
            if (!confirmLoss) return;
        }

        // Update defaults from UI
        if (!appState.policy.defaults) {
            appState.policy.defaults = {};
        }
        appState.policy.defaults.severity = document.getElementById('defaults-severity').value || undefined;
        appState.policy.defaults.defaultLanguage = document.getElementById('defaults-default-language').value || undefined;

        console.log('[DEBUG] Current role:', appState.currentUser.role);
        console.log('[DEBUG] Policy RBAC roles:', Object.keys(appState.policy.rbac?.roles || {}));

        // Save policy
        await API.savePolicy(appState.policy);

        // Save policy path if changed
        const newPath = document.getElementById('policy-path-input').value.trim();
        if (newPath && newPath !== appState.settings.policyPath) {
            console.log('Saving policy path:', newPath);
            await API.setPolicyPath(newPath);
            appState.settings.policyPath = newPath;
        }

        // Update current user info
        updateUserInfo();

        appState.isDirty = false;
        document.getElementById('save-btn').classList.remove('ring-2', 'ring-yellow-400');
        document.getElementById('floating-save-btn').classList.remove('ring-4', 'ring-yellow-400', 'animate-pulse');
        showToast('정책이 성공적으로 저장되었습니다!');

        // Always ask user about conversion after save
        const message =
            '저장이 완료되었습니다.\n\n' +
            'linter 설정 파일을 생성/갱신하시겠습니까?\n' +
            '(code-policy.json, ESLint, Checkstyle, PMD 등)\n\n' +
            '• 확인: LLM을 사용하여 설정 파일 생성\n' +
            '• 취소: 현재 설정 유지';

        if (confirm(message)) {
            try {
                showToast('정책 변환 중... 잠시만 기다려주세요', 'info');
                const convertResult = await API.convertPolicy();
                showToast(
                    `변환 완료! ${convertResult.filesWritten.length}개의 파일이 생성되었습니다.`,
                    'success'
                );
                console.log('Convert result:', convertResult);
            } catch (error) {
                console.error('Conversion failed:', error);
                showToast('변환 실패: ' + error.message, 'error');
            }
        }
    } catch (error) {
        console.error('Failed to save policy:', error);
        showToast('저장에 실패했습니다: ' + error.message, 'error');
    }
}

async function loadPolicy() {
    try {
        appState.policy = await API.getPolicy();
        console.log('Policy loaded. Rules count:', appState.policy.rules.length);
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
        const savedConfirmSave = localStorage.getItem('confirmSave');

        if (savedConfirmSave !== null) {
            appState.settings.confirmSave = savedConfirmSave === 'true';
        }

        // Update checkboxes
        const confirmSaveCheckbox = document.getElementById('confirm-save-checkbox');
        if (confirmSaveCheckbox) {
            confirmSaveCheckbox.checked = appState.settings.confirmSave;
        }
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

async function saveSettings() {
    try {
        const newPath = document.getElementById('policy-path-input').value.trim();
        console.log('Current policy path:', appState.settings.policyPath);
        console.log('New policy path from input:', newPath);

        if (newPath && newPath !== appState.settings.policyPath) {
            console.log('Path changed, saving to server...');
            await API.setPolicyPath(newPath);
            appState.settings.policyPath = newPath;
            showToast('설정이 저장되었습니다');
        } else if (!newPath) {
            console.log('Empty path, skipping save');
        } else {
            console.log('Path unchanged, skipping save');
        }

        const confirmSaveCheckbox = document.getElementById('confirm-save-checkbox');
        if (confirmSaveCheckbox) {
            appState.settings.confirmSave = confirmSaveCheckbox.checked;
            localStorage.setItem('confirmSave', appState.settings.confirmSave);
        }

        hideModal('settings-modal');
    } catch (error) {
        console.error('Failed to save settings:', error);
        showToast('설정 저장에 실패했습니다', 'error');
    }
}

// Update user role badge display
function updateUserRoleBadge(role) {
    const roleBadge = document.getElementById('user-role-badge');
    const roleColor = getRoleColor(role);
    roleBadge.textContent = role;
    roleBadge.className = `font-semibold text-sm px-3 py-1 rounded-full ${roleColor.bg} ${roleColor.text}`;
}

function updatePermissionBadges(permissions) {
    const container = document.getElementById('permission-badges');
    if (!container) return;
    const badges = getPermissionBadges(permissions);
    container.innerHTML = badges.map(renderPermissionBadge).join('');
}

// Update user info display in header
function updateUserInfo() {
    // In role-based mode, just update the display with current role
    const currentRole = appState.currentUser.role;
    if (currentRole) {
        // Update badge display
        updateUserRoleBadge(currentRole);
    }
}

// ==================== Render All ====================
function renderAll() {
    // Defaults
    const defaults = appState.policy.defaults || {};
    document.getElementById('defaults-severity').value = defaults.severity || '';

    // Language tags
    renderLanguageTags();

    // RBAC
    renderRBAC();

    // Categories
    renderCategories();

    // Rules
    renderRules();

    // Role selection
    renderRoleSelection();
}

// ==================== Permission-Based UI ====================
function applyPermissions() {
    const permissions = appState.currentUser?.permissions || {};
    const canEditPolicy = permissions.canEditPolicy === true; // Default to false if undefined (secure default)
    const canEditRoles = permissions.canEditRoles === true;   // Default to false if undefined (secure default)

    console.log('[Permissions] canEditPolicy:', canEditPolicy, 'canEditRoles:', canEditRoles);

    if (canEditPolicy) {
        // Show save buttons
        document.getElementById('save-btn')?.classList.remove('hidden');
        document.getElementById('floating-save-btn')?.classList.remove('hidden');

        // Show template button
        document.getElementById('template-btn')?.classList.remove('hidden');

        // Show import button
        document.getElementById('import-btn')?.classList.remove('hidden');

        // Enable RBAC inputs and show add/delete buttons
        document.querySelectorAll('.role-name-input, .role-allowWrite-input, .role-denyWrite-input').forEach(el => {
            el.disabled = false;
            el.classList.remove('bg-gray-200', 'cursor-not-allowed');
        });
        document.querySelectorAll('.role-canEditPolicy-input, .role-canEditRoles-input').forEach(el => {
            el.disabled = false;
            el.classList.remove('cursor-not-allowed');
        });
        document.querySelectorAll('.delete-rbac-role-btn').forEach(el => el.classList.remove('hidden'));
        document.getElementById('add-role-btn')?.classList.remove('hidden');

        // Enable defaults inputs
        const langInput = document.getElementById('defaults-language-input');
        if (langInput) {
            langInput.disabled = false;
            langInput.classList.remove('bg-gray-200', 'cursor-not-allowed');
        }
        document.getElementById('add-language-btn')?.classList.remove('hidden');
        const defaultLang = document.getElementById('defaults-default-language');
        if (defaultLang) {
            defaultLang.disabled = false;
            defaultLang.classList.remove('bg-gray-200', 'cursor-not-allowed');
        }
        const severity = document.getElementById('defaults-severity');
        if (severity) {
            severity.disabled = false;
            severity.classList.remove('bg-gray-200', 'cursor-not-allowed');
        }
        // Show remove buttons on language tags
        document.querySelectorAll('.remove-language-btn').forEach(el => el.classList.remove('hidden'));

        // Show category add form and edit/delete buttons
        document.getElementById('add-category-form')?.classList.remove('hidden');
        document.querySelectorAll('.edit-category-btn, .delete-category-btn').forEach(el => el.classList.remove('hidden'));
        // Enable category edit inputs
        document.querySelectorAll('.edit-category-name, .edit-category-description').forEach(el => {
            el.disabled = false;
            el.classList.remove('bg-gray-200', 'cursor-not-allowed');
        });
        const newCategoryName = document.getElementById('new-category-name');
        const newCategoryDesc = document.getElementById('new-category-description');
        if (newCategoryName) {
            newCategoryName.disabled = false;
            newCategoryName.classList.remove('bg-gray-200', 'cursor-not-allowed');
        }
        if (newCategoryDesc) {
            newCategoryDesc.disabled = false;
            newCategoryDesc.classList.remove('bg-gray-200', 'cursor-not-allowed');
        }

        // Show rule add/edit/delete buttons
        document.getElementById('add-rule-btn')?.classList.remove('hidden');
        document.getElementById('add-rule-btn-bottom')?.classList.remove('hidden');
        document.querySelectorAll('.delete-rule-btn').forEach(el => el.classList.remove('hidden'));

        // Enable rule inputs
        document.querySelectorAll('.say-input, .category-select, .language-select, .example-input').forEach(el => {
            el.disabled = false;
            el.classList.remove('bg-gray-200', 'cursor-not-allowed');
        });
    } else {
        // Hide save buttons
        document.getElementById('save-btn')?.classList.add('hidden');
        document.getElementById('floating-save-btn')?.classList.add('hidden');

        // Hide template button
        document.getElementById('template-btn')?.classList.add('hidden');

        // Hide import button
        document.getElementById('import-btn')?.classList.add('hidden');

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
        const langInput = document.getElementById('defaults-language-input');
        if (langInput) {
            langInput.disabled = true;
            langInput.classList.add('bg-gray-200', 'cursor-not-allowed');
        }
        document.getElementById('add-language-btn')?.classList.add('hidden');
        const defaultLang = document.getElementById('defaults-default-language');
        if (defaultLang) {
            defaultLang.disabled = true;
            defaultLang.classList.add('bg-gray-200', 'cursor-not-allowed');
        }
        const severity = document.getElementById('defaults-severity');
        if (severity) {
            severity.disabled = true;
            severity.classList.add('bg-gray-200', 'cursor-not-allowed');
        }
        // Hide remove buttons on language tags
        document.querySelectorAll('.remove-language-btn').forEach(el => el.classList.add('hidden'));

        // Hide category add form and edit/delete buttons
        document.getElementById('add-category-form')?.classList.add('hidden');
        document.querySelectorAll('.edit-category-btn, .delete-category-btn').forEach(el => el.classList.add('hidden'));
        // Disable category edit inputs
        document.querySelectorAll('.edit-category-name, .edit-category-description').forEach(el => {
            el.disabled = true;
            el.classList.add('bg-gray-200', 'cursor-not-allowed');
        });

        // Hide rule add/edit/delete buttons
        document.getElementById('add-rule-btn')?.classList.add('hidden');
        document.getElementById('add-rule-btn-bottom')?.classList.add('hidden');
        document.querySelectorAll('.delete-rule-btn').forEach(el => el.classList.add('hidden'));

        // Disable rule inputs
        document.querySelectorAll('.say-input, .category-select, .language-select, .example-input').forEach(el => {
            el.disabled = true;
            el.classList.add('bg-gray-200', 'cursor-not-allowed');
        });
    }

    // Clean up deprecated UI elements
    const existingReadonlyBadge = document.getElementById('readonly-badge');
    if (existingReadonlyBadge) existingReadonlyBadge.remove();
}

// ==================== Initialize ====================
async function init() {
    try {
        // Load current user and role
        appState.currentUser = await API.getMe();

        // Display current role in header
        updateUserRoleBadge(appState.currentUser.role || '역할 미선택');
        updatePermissionBadges(appState.currentUser.permissions);

        // Load project info
        const projectInfo = await API.getProjectInfo();
        document.getElementById('repo-info').textContent = `📁 ${projectInfo.project}`;

        // Load available roles for selection
        appState.availableRoles = await API.getAvailableRoles();

        // Load policy
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

    // Import button
    document.getElementById('import-btn').addEventListener('click', () => {
        showModal('import-modal');
    });

    // Import modal buttons
    document.getElementById('close-import-modal').addEventListener('click', () => hideModal('import-modal'));
    document.getElementById('cancel-import-btn').addEventListener('click', () => hideModal('import-modal'));
    document.getElementById('execute-import-btn').addEventListener('click', handleImport);

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

    // Category management
    document.getElementById('add-category-btn').addEventListener('click', handleAddCategory);
    document.getElementById('new-category-name').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleAddCategory();
        }
    });
    document.getElementById('new-category-description').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleAddCategory();
        }
    });

    // Language management
    document.getElementById('add-language-btn').addEventListener('click', handleAddLanguage);
    document.getElementById('defaults-language-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleAddLanguage();
        }
    });
    document.getElementById('defaults-default-language').addEventListener('change', handleDefaultLanguageChange);

    // Settings checkboxes - save to localStorage on change
    const confirmSaveCheckbox = document.getElementById('confirm-save-checkbox');
    if (confirmSaveCheckbox) {
        confirmSaveCheckbox.addEventListener('change', (e) => {
            appState.settings.confirmSave = e.target.checked;
            localStorage.setItem('confirmSave', appState.settings.confirmSave);
        });
    }

    // Modal close buttons
    document.getElementById('close-template-modal').addEventListener('click', () => hideModal('template-modal'));

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
