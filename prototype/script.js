// Mock Data
const MOCK_TASKS = [
    {
        id: 1,
        title: "审查 Q4 营销计划",
        status: "In Progress",
        group: "Marketing Q4",
        topic: "Campaigns",
        dueDate: "2023-11-25T18:00:00Z",
        assignee: "me",
        creator: "alice"
    },
    {
        id: 2,
        title: "修复 iOS 登录 Bug",
        status: "To Do",
        group: "Dev Squad",
        topic: "Bugs",
        dueDate: "2023-11-20T10:00:00Z", // Overdue
        assignee: "me",
        creator: "bob"
    },
    {
        id: 3,
        title: "更新隐私政策",
        status: "Done",
        group: "Personal Life",
        topic: null,
        dueDate: "2023-11-15T09:00:00Z",
        assignee: "me",
        creator: "me"
    },
    {
        id: 4,
        title: "设计新着陆页",
        status: "To Do",
        group: "Marketing Q4",
        topic: "Website",
        dueDate: "2023-11-30T12:00:00Z",
        assignee: "charlie",
        creator: "me"
    },
    {
        id: 5,
        title: "准备月度报告",
        status: "In Progress",
        group: "Dev Squad",
        topic: "Reports",
        dueDate: "2023-11-28T17:00:00Z",
        assignee: "me",
        creator: "dave"
    }
];

// --- Animation & Interaction Logic ---

// Page Transitions
function navigateTo(url) {
    document.body.classList.add('page-exit');
    setTimeout(() => {
        window.location.href = url;
    }, 400);
}

// Intercept Links
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            navigateTo(link.href);
        });
    });

    // Pull to Refresh (Home Only)
    if (document.getElementById('taskList')) {
        initPullToRefresh();
    }
});

// Pull to Refresh Logic
function initPullToRefresh() {
    const container = document.querySelector('.app-container');
    const ptr = document.createElement('div');
    ptr.className = 'ptr-container';
    ptr.innerHTML = `
        <i class="ri-refresh-line ptr-icon"></i>
        <span class="ptr-text">SYNCING NEURAL LINK...</span>
    `;
    container.insertBefore(ptr, container.firstChild);

    let startY = 0;
    let isPulling = false;

    window.addEventListener('touchstart', (e) => {
        if (window.scrollY === 0) {
            startY = e.touches[0].clientY;
            isPulling = true;
        }
    }, { passive: true });

    window.addEventListener('touchmove', (e) => {
        if (!isPulling) return;
        const y = e.touches[0].clientY;
        const diff = y - startY;

        if (diff > 0 && diff < 150) {
            ptr.style.opacity = diff / 100;
            ptr.style.transform = `translateY(${diff / 2}px)`;
        }
    }, { passive: true });

    window.addEventListener('touchend', (e) => {
        if (!isPulling) return;
        isPulling = false;
        const y = e.changedTouches[0].clientY;
        const diff = y - startY;

        if (diff > 80) {
            // Trigger Refresh
            ptr.classList.add('ptr-active');
            ptr.style.transform = 'translateY(60px)';

            // Simulate Load
            renderSkeletons();
            setTimeout(() => {
                ptr.classList.remove('ptr-active');
                ptr.style.opacity = '0';
                ptr.style.transform = 'translateY(-20px)';
                renderTasks(); // Restore content
            }, 1500);
        } else {
            // Reset
            ptr.style.opacity = '0';
            ptr.style.transform = 'translateY(-20px)';
        }
    });
}

// Skeleton Loading
function renderSkeletons() {
    if (!taskListEl) return;
    taskListEl.innerHTML = '';

    // Group Header Skeleton
    const groupHeader = document.createElement('div');
    groupHeader.className = 'group-header skeleton';
    groupHeader.style.width = '100px';
    groupHeader.style.height = '14px';
    taskListEl.appendChild(groupHeader);

    // Card Skeletons
    for (let i = 0; i < 3; i++) {
        const card = document.createElement('div');
        card.className = 'skeleton-card';
        card.innerHTML = `
            <div style="padding: 16px;">
                <div class="skeleton skeleton-text" style="width: 60%;"></div>
                <div class="skeleton skeleton-text" style="width: 30%; height: 10px;"></div>
                <div style="display:flex; gap:10px; margin-top:10px;">
                    <div class="skeleton skeleton-text" style="width: 20%; height: 10px;"></div>
                    <div class="skeleton skeleton-text" style="width: 20%; height: 10px;"></div>
                </div>
            </div>
        `;
        taskListEl.appendChild(card);
    }
}

// --- Shared Logic ---

function showToast(message) {
    const toast = document.createElement('div');
    toast.style.position = 'fixed';
    toast.style.bottom = '100px';
    toast.style.left = '50%';
    toast.style.transform = 'translateX(-50%)';
    toast.style.background = 'var(--neon-green)';
    toast.style.color = 'black';
    toast.style.padding = '12px 24px';
    toast.style.borderRadius = '8px';
    toast.style.fontWeight = '700';
    toast.style.fontFamily = 'var(--font-mono)';
    toast.style.boxShadow = '0 0 20px var(--neon-green-dim)';
    toast.style.zIndex = '1000';
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s, transform 0.3s';
    toast.innerText = message;

    document.body.appendChild(toast);

    // Animate In
    requestAnimationFrame(() => {
        toast.style.opacity = '1';
        toast.style.transform = 'translateX(-50%) translateY(-10px)';
    });

    // Remove
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(-50%) translateY(0)';
        setTimeout(() => toast.remove(), 300);
    }, 2000);
}

// --- Home Page Logic ---
const taskListEl = document.getElementById('taskList');

if (taskListEl) {
    // State
    let currentTab = 'assigned';
    let currentDbFilter = null;

    // Elements
    const filterBarEl = document.getElementById('filterBar');
    const currentDbNameEl = document.getElementById('currentDbName');
    const tabs = document.querySelectorAll('.segment');
    const filterBtn = document.getElementById('filterBtn');
    const clearFilterBtn = document.getElementById('clearFilterBtn');
    const dbModal = document.getElementById('dbModal');
    const closeModalBtn = document.querySelector('.close-modal-btn');
    const dbItems = document.querySelectorAll('.db-item');
    const fabBtn = document.querySelector('.fab-btn');
    const settingsBtn = document.querySelector('.icon-btn[aria-label="设置"]');

    // Helper: Format Date
    function formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = date - now;
        const isOverdue = diff < 0;
        const options = { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' };
        const text = date.toLocaleDateString('zh-CN', options);
        return { text, isOverdue };
    }

    // Helper: Get Status Info
    function getStatusInfo(status) {
        switch (status.toLowerCase()) {
            case 'to do': return { class: 'status-todo', text: '待办' };
            case 'in progress': return { class: 'status-inprogress', text: '进行中' };
            case 'done': return { class: 'status-done', text: '已完成' };
            default: return { class: 'status-todo', text: '待办' };
        }
    }

    // Render Tasks
    function renderTasks() {
        taskListEl.innerHTML = '';

        let filteredTasks = MOCK_TASKS.filter(task => {
            if (currentTab === 'assigned' && task.assignee !== 'me') return false;
            if (currentTab === 'created' && task.creator !== 'me') return false;
            if (currentDbFilter && task.group !== currentDbFilter) return false;
            return true;
        });

        // Grouping: Active (To Do + In Progress) vs Completed (Done)
        const activeTasks = filteredTasks.filter(t => t.status !== 'Done');
        const doneTasks = filteredTasks.filter(t => t.status === 'Done');

        // 1. Render Active Tasks
        if (activeTasks.length > 0) {
            const groupHeader = document.createElement('div');
            groupHeader.className = 'group-header';
            groupHeader.innerHTML = `待办事项 <span class="group-count">${activeTasks.length}</span>`;
            taskListEl.appendChild(groupHeader);

            activeTasks.forEach(task => renderTaskCard(task));
        } else if (doneTasks.length === 0) {
            // Empty State (No Active, No Done)
            taskListEl.innerHTML = `
                <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                    <i class="ri-inbox-line" style="font-size: 48px; opacity: 0.5;"></i>
                    <p style="margin-top: 16px;">暂无任务</p>
                </div>
            `;
            return;
        }

        // 2. Render Completed Tasks (Collapsible)
        if (doneTasks.length > 0) {
            // Toggle Button
            const toggleBtn = document.createElement('div');
            toggleBtn.className = 'completed-toggle';
            toggleBtn.innerHTML = `
                <span>已完成 (${doneTasks.length})</span>
                <i class="ri-arrow-down-s-line" style="transition: transform 0.3s;"></i>
            `;

            const completedContainer = document.createElement('div');
            completedContainer.className = 'completed-list';
            completedContainer.style.display = 'none'; // Default collapsed

            toggleBtn.onclick = () => {
                const isHidden = completedContainer.style.display === 'none';
                completedContainer.style.display = isHidden ? 'block' : 'none';
                toggleBtn.querySelector('i').style.transform = isHidden ? 'rotate(180deg)' : 'rotate(0deg)';
                toggleBtn.classList.toggle('active', isHidden);
            };

            taskListEl.appendChild(toggleBtn);
            taskListEl.appendChild(completedContainer);

            doneTasks.forEach(task => renderTaskCard(task, completedContainer));
        }
    }

    function renderTaskCard(task, container = taskListEl) {
        const dateInfo = formatDate(task.dueDate);
        const statusInfo = getStatusInfo(task.status);

        const card = document.createElement('div');
        card.className = 'task-card';
        if (task.status === 'Done') card.classList.add('task-done-dim'); // Visual dimming

        card.onclick = () => openActionModal(task);

        card.innerHTML = `
            <div class="task-header">
                <div class="task-title">${task.title}</div>
                <div class="task-status" title="${task.group}">${task.group}</div>
            </div>
            <div class="task-meta">
                <div class="meta-item">
                    <i class="ri-user-3-line"></i>
                    <span>${task.creator === 'me' ? '我' : task.creator}</span>
                </div>
                <div class="meta-item ${dateInfo.isOverdue && task.status !== 'Done' ? 'date-overdue' : ''}">
                    <i class="ri-time-line"></i>
                    <span>${dateInfo.text}</span>
                </div>
            </div>
        `;
        container.appendChild(card);
    }

    // Action Modal Logic
    const actionModal = document.getElementById('actionModal');
    const closeActionModalBtn = document.getElementById('closeActionModal');
    let currentActionTask = null;

    function openActionModal(task) {
        currentActionTask = task;
        document.getElementById('actionModalTitle').innerText = `操作: ${task.title}`;
        actionModal.style.display = 'flex';
    }

    closeActionModalBtn.addEventListener('click', () => {
        actionModal.style.display = 'none';
        currentActionTask = null;
    });

    // Handle Actions
    document.querySelectorAll('.action-item').forEach(item => {
        item.addEventListener('click', () => {
            const action = item.dataset.action;
            actionModal.style.display = 'none';

            if (!currentActionTask) return;

            switch (action) {
                case 'reply':
                    // Go to detail page (simulate focusing comment)
                    navigateTo(`detail.html?id=${currentActionTask.id}&focus=comment`);
                    break;
                case 'jump':
                    showToast('已跳转到 Telegram 消息');
                    break;
                case 'done':
                    currentActionTask.status = 'Done';
                    showToast('任务已标记为完成');
                    renderTasks();
                    break;
                case 'detail':
                    navigateTo(`detail.html?id=${currentActionTask.id}`);
                    break;
            }
        });
    });

    // Event Listeners
    tabs.forEach((tab, index) => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            document.querySelector('.active-border').style.transform = `translateX(${index * 100}%)`;
            currentTab = tab.dataset.tab;

            // Simulate Loading on Tab Switch
            renderSkeletons();
            setTimeout(renderTasks, 600);
        });
    });

    filterBtn.addEventListener('click', () => dbModal.style.display = 'flex');
    closeModalBtn.addEventListener('click', () => dbModal.style.display = 'none');

    dbItems.forEach(item => {
        item.addEventListener('click', () => {
            const dbName = item.dataset.db;
            currentDbFilter = dbName;
            currentDbNameEl.textContent = dbName;
            filterBarEl.style.display = 'flex';
            dbModal.style.display = 'none';

            // Simulate Loading on Filter
            renderSkeletons();
            setTimeout(renderTasks, 600);
        });
    });

    clearFilterBtn.addEventListener('click', () => {
        currentDbFilter = null;
        filterBarEl.style.display = 'none';
        renderSkeletons();
        setTimeout(renderTasks, 600);
    });

    // Navigation
    fabBtn.addEventListener('click', () => navigateTo('detail.html'));
    settingsBtn.addEventListener('click', () => navigateTo('settings.html'));

    // Initial Render with Skeleton
    renderSkeletons();
    setTimeout(renderTasks, 800);

    // Scroll Effect
    window.addEventListener('scroll', () => {
        const header = document.querySelector('.header');
        if (window.scrollY > 10) {
            header.style.background = 'rgba(0,0,0,0.95)';
            header.style.borderBottom = '1px solid var(--border-dim)';
        } else {
            header.style.background = 'linear-gradient(180deg, rgba(0,0,0,0.9) 0%, rgba(0,0,0,0) 100%)';
            header.style.borderBottom = 'none';
        }
    });
}

// --- Binding Page Logic ---
function checkManualDb() {
    const input = document.getElementById('manualDbId');
    const val = input.value.trim();

    if (val.length < 10) {
        showToast('请输入有效的 Database ID');
        return;
    }

    // Simulate Check
    document.getElementById('schemaModal').style.display = 'flex';
    setTimeout(() => {
        document.getElementById('schemaLoading').style.display = 'none';
        document.getElementById('schemaResult').style.display = 'block';
    }, 1500);
}

// --- Detail Page Logic ---
const backBtn = document.getElementById('backBtn');
const taskTitleInput = document.getElementById('taskTitle');

if (backBtn) {
    // Auto-save on Back
    backBtn.addEventListener('click', () => {
        // Simulate Save
        showToast('任务已自动保存');
        setTimeout(() => {
            navigateTo('index.html');
        }, 800);
    });

    // Context Snapshot Toggle
    const toggleContext = document.getElementById('toggleContext');
    const contextContent = document.getElementById('contextContent');
    if (toggleContext) {
        toggleContext.addEventListener('click', (e) => {
            // Prevent toggle when clicking the history link
            if (e.target.closest('.history-link')) return;

            const isHidden = contextContent.style.display === 'none';
            contextContent.style.display = isHidden ? 'block' : 'none';

            const icon = toggleContext.querySelector('.toggle-icon');
            if (icon) {
                icon.style.transform = isHidden ? 'rotate(180deg)' : 'rotate(0deg)';
            }
        });
    }

    // Mock Threaded Comments
    const MOCK_COMMENTS = [
        {
            id: 1,
            author: 'Alice',
            avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Alice',
            time: '2小时前',
            text: '我看了一下日志，确实是 Token 过期的问题。',
            replies: [
                {
                    id: 2,
                    author: 'Bob',
                    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Bob',
                    time: '1小时前',
                    text: '是 Refresh Token 没生效吗？',
                    replies: []
                }
            ]
        },
        {
            id: 3,
            author: 'Charlie',
            avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Charlie',
            time: '30分钟前',
            text: '我已经提交了修复补丁，正在跑 CI。',
            replies: []
        }
    ];

    // Render Comments
    const commentListEl = document.getElementById('commentList');
    if (commentListEl) {
        function renderComments(comments, container) {
            comments.forEach(comment => {
                const item = document.createElement('div');
                item.className = 'comment-item';
                item.innerHTML = `
                    <img src="${comment.avatar}" class="comment-avatar">
                    <div class="comment-content">
                        <div class="comment-header">
                            <span class="comment-author">${comment.author}</span>
                            <span class="comment-time">${comment.time}</span>
                        </div>
                        <div class="comment-text">${comment.text}</div>
                        <div class="comment-actions">
                            <button class="comment-action-btn"><i class="ri-reply-line"></i> 回复</button>
                            <button class="comment-action-btn"><i class="ri-thumb-up-line"></i> 点赞</button>
                        </div>
                        ${comment.replies.length > 0 ? '<div class="nested-comments"></div>' : ''}
                    </div>
                `;
                container.appendChild(item);

                if (comment.replies.length > 0) {
                    renderComments(comment.replies, item.querySelector('.nested-comments'));
                }
            });
        }
        renderComments(MOCK_COMMENTS, commentListEl);
    }

    // --- HUD Manual Toggle Logic ---
    const bottomHud = document.getElementById('bottomHud');
    const hudToggleBtn = document.getElementById('hudToggleBtn');

    if (bottomHud && hudToggleBtn) {
        hudToggleBtn.addEventListener('click', () => {
            bottomHud.classList.toggle('hud-collapsed');
        });
    }

    // Auto-expand on input focus
    const commentInput = document.getElementById('commentInput');
    if (commentInput) {
        commentInput.addEventListener('focus', () => {
            bottomHud.classList.remove('hud-collapsed');
        });
    }

    // Auto-resize Title Textarea
    const taskTitle = document.getElementById('taskTitle');
    if (taskTitle) {
        const autoResize = () => {
            taskTitle.style.height = 'auto';
            taskTitle.style.height = taskTitle.scrollHeight + 'px';
        };

        taskTitle.addEventListener('input', autoResize);
        // Initial resize
        autoResize();
    }
}

// --- Custom Status Modal Logic ---
const statusPill = document.getElementById('statusPill');
const statusModal = document.getElementById('statusModal');
const closeStatusModal = document.getElementById('closeStatusModal');
const statusText = document.getElementById('statusText');

if (statusPill) {
    statusPill.addEventListener('click', () => {
        statusModal.style.display = 'flex';
    });
}

if (closeStatusModal) {
    closeStatusModal.addEventListener('click', () => {
        statusModal.style.display = 'none';
    });
}

// Close on overlay click
if (statusModal) {
    statusModal.addEventListener('click', (e) => {
        if (e.target === statusModal) {
            statusModal.style.display = 'none';
        }
    });
}

window.updateStatus = function (status) {
    statusText.innerText = status === 'To Do' ? '待办' : (status === 'In Progress' ? '进行中' : '已完成');
    statusModal.style.display = 'none';
    showToast(`状态已更新为: ${statusText.innerText}`);
};

// --- Custom Date Modal Logic ---
const datePill = document.getElementById('datePill');
const dateModal = document.getElementById('dateModal');
const closeDateModal = document.getElementById('closeDateModal');
const dateText = document.getElementById('dateText');

if (datePill) {
    datePill.addEventListener('click', () => {
        dateModal.style.display = 'flex';
    });
}

if (closeDateModal) {
    closeDateModal.addEventListener('click', () => {
        dateModal.style.display = 'none';
    });
}

// Close on overlay click
if (dateModal) {
    dateModal.addEventListener('click', (e) => {
        if (e.target === dateModal) {
            dateModal.style.display = 'none';
        }
    });
}

window.updateDate = function (date) {
    dateText.innerText = date;
    dateModal.style.display = 'none';
    showToast(`截止日期已更新为: ${date}`);
};

// Bottom Action Bar Logic (Legacy - can be cleaned up or reused for HUD actions)
const jumpBtn = document.getElementById('jumpBtn');
const moreBtn = document.getElementById('moreBtn');
const openNotionBtn = document.getElementById('openNotionBtn');
const moreActionsModal = document.getElementById('moreActionsModal');
const closeMoreModal = document.getElementById('closeMoreModal');

if (jumpBtn) {
    jumpBtn.addEventListener('click', () => showToast('已跳转到 Telegram 原始消息'));
}

if (openNotionBtn) {
    openNotionBtn.addEventListener('click', () => showToast('正在唤起 Notion App...'));
}

if (moreBtn) {
    moreBtn.addEventListener('click', () => {
        moreActionsModal.style.display = 'flex';
    });
}

if (closeMoreModal) {
    closeMoreModal.addEventListener('click', () => {
        moreActionsModal.style.display = 'none';
    });
}

// Close on overlay click
if (moreActionsModal) {
    moreActionsModal.addEventListener('click', (e) => {
        if (e.target === moreActionsModal) {
            moreActionsModal.style.display = 'none';
        }
    });
}

// Handle More Actions
document.querySelectorAll('.action-item[data-action="delete"]').forEach(btn => {
    btn.addEventListener('click', () => {
        if (confirm('确定要删除这个任务吗？此操作无法撤销。')) {
            showToast('任务已删除');
            setTimeout(() => navigateTo('index.html'), 1000);
        }
        moreActionsModal.style.display = 'none';
    });
});

