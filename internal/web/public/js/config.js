// 全局变量
let currentTheme = 'dark'; // 默认使用暗色主题

// DOM元素
const $themeToggle = document.getElementById('theme-toggle');
const $configForm = document.getElementById('serverchan-config-form');
const $serverchanEnabled = document.getElementById('serverchan-enabled');
const $sendKey = document.getElementById('send-key');
const $onTemplate = document.getElementById('on-template');
const $offTemplate = document.getElementById('off-template');

// ntfy DOM元素
const $ntfyForm = document.getElementById('ntfy-config-form');
const $ntfyEnabled = document.getElementById('ntfy-enabled');
const $ntfyServerUrl = document.getElementById('ntfy-server-url');
const $ntfyTopic = document.getElementById('ntfy-topic');
const $ntfyToken = document.getElementById('ntfy-token');
const $ntfyOnTemplate = document.getElementById('ntfy-on-template');
const $ntfyOffTemplate = document.getElementById('ntfy-off-template');
const $testNtfyBtn = document.getElementById('test-ntfy-btn');

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    initApp();
});

// 初始化应用
async function initApp() {
    // 初始化主题
    initTheme();
    
    // 获取配置
    await fetchServerChanConfig();
    await fetchNtfyConfig();
    
    // 初始化事件监听器
    initEventListeners();
}

// 获取Server酱配置
async function fetchServerChanConfig() {
    try {
        const response = await fetch('/api/serverchan/config');
        if (!response.ok) {
            throw new Error('获取Server酱配置失败');
        }

        const config = await response.json();
        
        // 更新表单显示
        updateConfigForm(config);
    } catch (error) {
        console.error('获取Server酱配置错误:', error);
        showNotification('获取配置失败', 'error');
    }
}

// 更新配置表单显示
function updateConfigForm(config) {
    $serverchanEnabled.checked = config.enabled || false;
    $sendKey.value = config.sendKey || '';
    $onTemplate.value = config.onTemplate || '{{.Name}} 任务执行成功，灯已开启';
    $offTemplate.value = config.offTemplate || '{{.Name}} 任务执行成功，灯已关闭';
}

// 保存Server酱配置
async function saveServerChanConfig(e) {
    e.preventDefault();
    
    // 获取表单数据
    const config = {
        enabled: $serverchanEnabled.checked,
        sendKey: $sendKey.value,
        onTemplate: $onTemplate.value,
        offTemplate: $offTemplate.value
    };
    
    try {
        const response = await fetch('/api/serverchan/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '保存配置失败');
        }
        
        showNotification('配置保存成功', 'success');
        
    } catch (error) {
        console.error('保存Server酱配置错误:', error);
        showNotification(`配置保存失败: ${error.message}`, 'error');
    }
}

// 获取ntfy配置
async function fetchNtfyConfig() {
    try {
        const response = await fetch('/api/ntfy/config');
        if (!response.ok) throw new Error('获取ntfy配置失败');
        const config = await response.json();
        $ntfyEnabled.checked = config.enabled || false;
        $ntfyServerUrl.value = config.server_url || 'https://ntfy.sh';
        $ntfyTopic.value = config.topic || '';
        $ntfyToken.value = config.token || '';
        $ntfyOnTemplate.value = config.on_template || '{{.Name}} 任务执行成功，灯已开启';
        $ntfyOffTemplate.value = config.off_template || '{{.Name}} 任务执行成功，灯已关闭';
    } catch (error) {
        console.error('获取ntfy配置错误:', error);
    }
}

// 保存ntfy配置
async function saveNtfyConfig(e) {
    e.preventDefault();
    const config = {
        enabled: $ntfyEnabled.checked,
        server_url: $ntfyServerUrl.value,
        topic: $ntfyTopic.value,
        token: $ntfyToken.value,
        on_template: $ntfyOnTemplate.value,
        off_template: $ntfyOffTemplate.value
    };
    try {
        const response = await fetch('/api/ntfy/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '保存配置失败');
        }
        showNotification('ntfy配置保存成功', 'success');
    } catch (error) {
        console.error('保存ntfy配置错误:', error);
        showNotification(`配置保存失败: ${error.message}`, 'error');
    }
}

// 测试ntfy连接
async function testNtfyConnection() {
    $testNtfyBtn.disabled = true;
    $testNtfyBtn.textContent = '测试中...';
    try {
        const response = await fetch('/api/ntfy/test', { method: 'POST' });
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || '测试失败');
        showNotification(data.message || '测试通知已发送', 'success');
    } catch (error) {
        showNotification(`测试失败: ${error.message}`, 'error');
    } finally {
        $testNtfyBtn.disabled = false;
        $testNtfyBtn.textContent = '测试连接';
    }
}

// 显示通知
function showNotification(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    // 设置图标
    let icon;
    switch (type) {
        case 'success':
            icon = 'ri-check-line';
            break;
        case 'error':
            icon = 'ri-error-warning-line';
            break;
        case 'warning':
            icon = 'ri-alert-line';
            break;
        default:
            icon = 'ri-information-line';
    }

    toast.innerHTML = `
        <div class="toast-icon">
            <i class="${icon}"></i>
        </div>
        <div class="toast-content">
            <p>${message}</p>
        </div>
        <button class="toast-close" aria-label="关闭">
            <i class="ri-close-line"></i>
        </button>
    `;

    // 添加到容器
    const container = document.getElementById('toast-container');
    container.appendChild(toast);

    // 关闭按钮事件
    const closeBtn = toast.querySelector('.toast-close');
    closeBtn.addEventListener('click', () => {
        closeToast(toast);
    });

    // 添加显示类，触发动画
    setTimeout(() => {
        toast.classList.add('show');
    }, 10);

    // 自动关闭
    setTimeout(() => {
        closeToast(toast);
    }, 3000);
}

// 关闭通知
function closeToast(toast) {
    // 如果已经关闭，不执行
    if (!toast.parentNode) return;

    // 触发关闭动画
    toast.classList.remove('show');

    // 动画结束后移除元素
    setTimeout(() => {
        if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
        }
    }, 300);
}

// 初始化事件监听器
function initEventListeners() {
    // 保存配置
    $configForm.addEventListener('submit', saveServerChanConfig);
    
    // 主题切换按钮
    $themeToggle.addEventListener('click', toggleTheme);
    
    // ntfy 配置
    $ntfyForm.addEventListener('submit', saveNtfyConfig);
    $testNtfyBtn.addEventListener('click', testNtfyConnection);
}

// 初始化主题
function initTheme() {
    // 检查本地存储中是否有保存的主题设置
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        currentTheme = savedTheme;
    }
    
    // 应用主题
    applyTheme(currentTheme);
    
    // 确保初始图标显示正确
    const darkIcon = document.getElementById('dark-icon');
    const lightIcon = document.getElementById('light-icon');
    
    if (currentTheme === 'dark') {
        darkIcon.style.display = 'block';
        lightIcon.style.display = 'none';
    } else {
        darkIcon.style.display = 'none';
        lightIcon.style.display = 'block';
    }
}

// 切换主题
function toggleTheme() {
    currentTheme = currentTheme === 'dark' ? 'light' : 'dark';
    applyTheme(currentTheme);
    localStorage.setItem('theme', currentTheme);
}

// 应用主题
function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    
    // 更新主题切换按钮的图标
    const darkIcon = document.getElementById('dark-icon');
    const lightIcon = document.getElementById('light-icon');
    
    if (theme === 'dark') {
        darkIcon.style.display = 'block';
        lightIcon.style.display = 'none';
    } else {
        darkIcon.style.display = 'none';
        lightIcon.style.display = 'block';
    }
}