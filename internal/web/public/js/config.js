// 全局变量
let currentTheme = 'dark'; // 默认使用暗色主题

// DOM元素
const $themeToggle = document.getElementById('theme-toggle');
const $configForm = document.getElementById('serverchan-config-form');
const $serverchanEnabled = document.getElementById('serverchan-enabled');
const $emailEnabled = document.getElementById('email-enabled');
const $emailUrl = document.getElementById('email-url');
const $sendKey = document.getElementById('send-key');
const $onTemplate = document.getElementById('on-template');
const $offTemplate = document.getElementById('off-template');

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
    $emailEnabled.checked = config.emailEnabled || false;
    $emailUrl.value = config.emailURL || '';
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
        emailEnabled: $emailEnabled.checked,
        emailUrl: $emailUrl.value,
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
        
        showNotification('配置已保存', 'success');
    } catch (error) {
        console.error('保存配置错误:', error);
        showNotification(`保存配置失败: ${error.message}`, 'error');
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