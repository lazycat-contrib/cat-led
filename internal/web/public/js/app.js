// 全局变量
let currentUserInfo = null;
let allUserInfos = [];
let currentLedStatus = false;
let schedules = [];
let currentEditingScheduleId = null;
let statusRefreshInterval = null; // 新增：用于存储状态刷新的定时器ID

// DOM元素
const $ledToggle = document.getElementById('led-toggle');
const $ledStatus = document.getElementById('led-status');
const $schedulesList = document.getElementById('schedules-list');
const $addScheduleBtn = document.getElementById('add-schedule-btn');
const $scheduleModal = document.getElementById('schedule-modal');
const $modalTitle = document.getElementById('modal-title');
const $scheduleForm = document.getElementById('schedule-form');
const $closeModalBtn = document.getElementById('close-modal-btn');
const $cancelScheduleBtn = document.getElementById('cancel-schedule-btn');
const $daySelects = document.querySelectorAll('.day-select');

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    initApp();
});

// 应用初始化
async function initApp() {
    // 获取用户信息
    await fetchUserInfo();
    
    // 获取LED状态
    await fetchLedStatus();
    
    // 设置2秒定时刷新设备状态
    startStatusRefresh();
    
    // 获取定时任务
    await fetchSchedules();
    
    // 初始化事件监听器
    initEventListeners();
}

// 获取用户信息
async function fetchUserInfo() {
    try {
        const response = await fetch('/userinfo');
        if (!response.ok) {
            throw new Error('获取用户信息失败');
        }
        
        const data = await response.json();
        
        // 添加日志，帮助调试
        console.log('用户信息API响应:', data);
        
        // 确保数据格式正确
        if (!data || typeof data !== 'object') {
            throw new Error('用户信息格式不正确');
        }
        
        currentUserInfo = data.CurrentUserInfo || {};
        detailInfo = data.Detail || {};
        
        // 更新用户信息显示
        updateUserInfoDisplay();
    } catch (error) {
        console.error('获取用户信息错误:', error);
        showNotification('获取用户信息失败', 'error');
    }
}

// 更新用户信息显示
function updateUserInfoDisplay() {
    console.log('更新用户信息显示', { currentUserInfo, detailInfo });
    
    // 即使currentUserInfo不完整，也尝试显示尽可能多的信息
    
    // 处理用户头像
    const userAvatarElem = document.querySelector('.user-avatar');
    if (detailInfo && detailInfo.avatar) {
        // 如果有头像，替换默认图标为图片
        userAvatarElem.innerHTML = `<img src="${detailInfo.avatar}" alt="用户头像">`;
    } else {
        // 没有头像时使用默认图标
        userAvatarElem.innerHTML = `<i class="ri-user-3-line"></i>`;
    }
    
    // 构建用户名显示文本，同时显示nickname和uid（如果存在）
    let userNameText = '您好！ ';
    
    // 检查nickname是否存在
    if (detailInfo && detailInfo.nickname) {
        userNameText += detailInfo.nickname;
        // 如果uid也存在，添加括号中的uid
        if (detailInfo.uid) {
            userNameText += ` (${detailInfo.uid})`;
        }
    } else if (detailInfo && detailInfo.uid) {
        // 只有uid存在
        userNameText += detailInfo.uid;
    } else {
        // 都不存在
        userNameText += '没名字的小懒猫';
    }
    
    // 安全地更新DOM
    try {
        // 更新用户名
        document.querySelector('.user-name').textContent = userNameText;
        
        // 处理角色信息 - 只在有值时显示
        const userRoleElem = document.querySelector('.user-role');
        if (detailInfo && detailInfo.role) {
            // 根据role值显示对应角色名称
            let roleName = '普通用户';
            if (detailInfo.role === 1) {
                roleName = '管理员';
            } else if (detailInfo.role === 2) {
                roleName = '超级管理员';
            }
            userRoleElem.textContent = `角色: ${roleName}`;
            userRoleElem.style.display = ''; // 显示
        } else {
            userRoleElem.style.display = 'none'; // 隐藏
        }
        
        // 处理设备信息 - 只在有值时显示
        const deviceInfoSection = document.querySelector('.device-info');
        const deviceIdElem = document.querySelector('.device-id');
        const deviceVersionElem = document.querySelector('.device-version');
        
        // 检查是否有任何设备信息
        if (currentUserInfo && (currentUserInfo.DeviceID || currentUserInfo.DeviceVersion)) {
            // 只有在有值时显示相应信息
            if (currentUserInfo.DeviceID) {
                deviceIdElem.textContent = `设备ID: ${currentUserInfo.DeviceID}`;
                deviceIdElem.style.display = '';
            } else {
                deviceIdElem.style.display = 'none';
            }
            
            if (currentUserInfo.DeviceVersion) {
                deviceVersionElem.textContent = `版本: ${currentUserInfo.DeviceVersion}`;
                deviceVersionElem.style.display = '';
            } else {
                deviceVersionElem.style.display = 'none';
            }
            
            // 如果至少有一个值，显示设备信息区域
            deviceInfoSection.style.display = '';
        } else {
            // 没有任何设备信息时，隐藏整个区域
            deviceInfoSection.style.display = 'none';
        }
    } catch (error) {
        console.error('更新用户信息DOM时出错:', error);
    }
}

// 获取LED状态
async function fetchLedStatus() {
    try {
        const response = await fetch('/api/led-status');
        if (response.ok) {
            const data = await response.json();
            updateLedStatus(data.status);
            
            // 连续失败计数器重置
            if (window.ledStatusErrorCount && window.ledStatusErrorCount > 0) {
                window.ledStatusErrorCount = 0;
                console.log('设备状态连接已恢复');
            }
        } else {
            handleLedStatusError('API返回错误: ' + response.status);
        }
    } catch (error) {
        handleLedStatusError(error.message);
    }
}

// 处理LED状态获取错误
function handleLedStatusError(errorMsg) {
    // 使用全局变量跟踪连续失败次数
    if (!window.ledStatusErrorCount) {
        window.ledStatusErrorCount = 0;
    }
    
    window.ledStatusErrorCount++;
    
    console.log(`设备状态刷新失败 (${window.ledStatusErrorCount}): ${errorMsg}`);
    
    // 如果连续失败5次以上，显示通知并减慢刷新频率
    if (window.ledStatusErrorCount === 5) {
        showNotification('设备状态连接异常，正在尝试重新连接...', 'warning');
        // 调整为5秒刷新一次
        restartStatusRefreshWithInterval(5000);
    }
    
    // 如果连续失败15次以上，考虑暂停刷新或进一步降低频率
    if (window.ledStatusErrorCount === 15) {
        showNotification('设备状态连接失败，请检查网络连接', 'error');
        // 调整为10秒刷新一次
        restartStatusRefreshWithInterval(10000);
    }
}

// 使用指定的时间间隔重启状态刷新
function restartStatusRefreshWithInterval(interval) {
    if (statusRefreshInterval) {
        clearInterval(statusRefreshInterval);
    }
    
    statusRefreshInterval = setInterval(fetchLedStatus, interval);
    console.log(`设备状态刷新已调整为${interval/1000}秒刷新一次`);
}

// 开始定时刷新设备状态
function startStatusRefresh() {
    // 如果已经有定时器在运行，先清除它
    if (statusRefreshInterval) {
        clearInterval(statusRefreshInterval);
    }
    
    // 重置错误计数器
    window.ledStatusErrorCount = 0;
    
    // 设置新的定时器，每2秒刷新一次设备状态
    statusRefreshInterval = setInterval(fetchLedStatus, 2000);
    console.log('设备状态刷新已启动，每2秒刷新一次');
}

// 停止定时刷新设备状态
function stopStatusRefresh() {
    if (statusRefreshInterval) {
        clearInterval(statusRefreshInterval);
        statusRefreshInterval = null;
        console.log('设备状态刷新已停止');
    }
}

// 更新LED状态显示
function updateLedStatus(status) {
    currentLedStatus = status;
    
    // 更新复选框状态
    $ledToggle.checked = status;
    
    // 更新文本状态
    $ledStatus.textContent = status ? '开启' : '关闭';
}

// 切换LED状态
async function toggleLedStatus() {
    try {
        const response = await fetch('/ledcontrol');
        if (!response.ok) {
            throw new Error('切换LED状态失败');
        }
        
        // 更新状态 (由于我们没有获取状态的接口，所以切换后直接反转当前已知状态)
        updateLedStatus(!currentLedStatus);
        
        showNotification(`LED已${currentLedStatus ? '开启' : '关闭'}`, 'success');
    } catch (error) {
        console.error('切换LED状态错误:', error);
        showNotification('切换LED状态失败', 'error');

        // 发生错误时恢复复选框状态
        $ledToggle.checked = currentLedStatus;
    }
}

// 获取定时任务列表
async function fetchSchedules() {
    try {
        const response = await fetch('/api/schedules');
        if (!response.ok) {
            throw new Error('获取定时任务列表失败');
        }
        
        schedules = await response.json();
        renderSchedulesList();
    } catch (error) {
        console.error('获取定时任务列表错误:', error);
        showNotification('获取定时任务列表失败', 'error');
    }
}

// 渲染定时任务列表
function renderSchedulesList() {
    if (!schedules || schedules.length === 0) {
        $schedulesList.innerHTML = `
            <div class="empty-state">
                <i class="ri-time-line"></i>
                <p>暂无定时任务，点击右上角添加</p>
            </div>
        `;
        return;
    }
    
    let html = '';
    
    schedules.forEach(schedule => {
        const startTime = new Date(schedule.startTime).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
        const endTime = new Date(schedule.endTime).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
        const creatorName = findUserNameById(schedule.creatorId);
        const isCurrentUserCreator = schedule.creatorId === currentUserInfo.UserId;
        const canEdit = isCurrentUserCreator || schedule.allowEdit;
        
        html += `
            <div class="schedule-item" data-id="${schedule.id}">
                <div class="schedule-header">
                    <div class="schedule-name">${schedule.name}</div>
                    <div class="schedule-toggle">
                        <div class="toggle-switch">
                            <input type="checkbox" id="toggle-${schedule.id}" ${schedule.enabled ? 'checked' : ''} ${canEdit ? '' : 'disabled'}>
                            <label for="toggle-${schedule.id}" class="toggle-label" onclick="toggleSchedule('${schedule.id}')"></label>
                        </div>
                    </div>
                </div>
                <div class="schedule-times">
                    <span>${startTime}</span>
                    <i class="ri-arrow-right-line"></i>
                    <span>${endTime}</span>
                </div>
                <div class="schedule-dates">
                    ${renderWeekdays(schedule.repeatDays)}
                </div>
                <div class="schedule-creator">
                    <i class="ri-user-line"></i>
                    <span>${creatorName}</span>
                    ${schedule.allowEdit ? '<span class="public-badge">公开</span>' : ''}
                </div>
                ${canEdit ? `
                <div class="schedule-actions">
                    <button class="schedule-action-btn edit" onclick="openEditScheduleModal('${schedule.id}')">
                        编辑
                    </button>
                    ${isCurrentUserCreator ? `
                    <button class="schedule-action-btn delete" onclick="deleteSchedule('${schedule.id}')">
                        删除
                    </button>
                    ` : ''}
                </div>
                ` : ''}
            </div>
        `;
    });
    
    $schedulesList.innerHTML = html;
}

// 渲染星期显示
function renderWeekdays(days) {
    const weekdays = ['日', '一', '二', '三', '四', '五', '六'];
    let html = '';
    
    weekdays.forEach((day, index) => {
        const isActive = days.includes(index);
        html += `<div class="schedule-day ${isActive ? 'active' : ''}">${day}</div>`;
    });
    
    return html;
}

// 打开添加定时任务模态框
function openAddScheduleModal() {
    currentEditingScheduleId = null;
    $modalTitle.textContent = '添加定时任务';
    $scheduleForm.reset();
    
    // 重置星期选择
    $daySelects.forEach(el => el.classList.remove('selected'));
    
    // 显示模态框
    $scheduleModal.classList.add('show');
}

// 打开编辑定时任务模态框
function openEditScheduleModal(scheduleId) {
    const schedule = schedules.find(s => s.id === scheduleId);
    if (!schedule) return;
    
    currentEditingScheduleId = scheduleId;
    $modalTitle.textContent = '编辑定时任务';
    
    // 填充表单
    document.getElementById('schedule-name').value = schedule.name;
    document.getElementById('start-time').value = formatTimeForInput(new Date(schedule.startTime));
    document.getElementById('end-time').value = formatTimeForInput(new Date(schedule.endTime));
    document.getElementById('allow-edit').checked = schedule.allowEdit;
    document.getElementById('schedule-enabled').checked = schedule.enabled;
    
    // 设置星期选择
    $daySelects.forEach(el => {
        const day = parseInt(el.dataset.day);
        if (schedule.repeatDays.includes(day)) {
            el.classList.add('selected');
        } else {
            el.classList.remove('selected');
        }
    });
    
    // 显示模态框
    $scheduleModal.classList.add('show');
}

// 关闭模态框
function closeModal() {
    $scheduleModal.classList.remove('show');
}

// 格式化时间为输入框格式 (HH:MM)
function formatTimeForInput(date) {
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
}

// 保存定时任务
async function saveSchedule(e) {
    e.preventDefault();
    
    // 获取表单数据
    const name = document.getElementById('schedule-name').value;
    const startTimeStr = document.getElementById('start-time').value;
    const endTimeStr = document.getElementById('end-time').value;
    const allowEdit = document.getElementById('allow-edit').checked;
    const enabled = document.getElementById('schedule-enabled').checked;
    
    // 获取选中的星期
    const repeatDays = [];
    $daySelects.forEach(el => {
        if (el.classList.contains('selected')) {
            repeatDays.push(parseInt(el.dataset.day));
        }
    });
    
    // 创建日期对象
    const now = new Date();
    const [startHours, startMinutes] = startTimeStr.split(':').map(Number);
    const [endHours, endMinutes] = endTimeStr.split(':').map(Number);
    
    const startTime = new Date(now);
    startTime.setHours(startHours, startMinutes, 0, 0);
    
    const endTime = new Date(now);
    endTime.setHours(endHours, endMinutes, 0, 0);
    
    // 构建任务数据
    const scheduleData = {
        name,
        startTime,
        endTime,
        repeatDays,
        allowEdit,
        enabled
    };
    
    try {
        let response;
        
        if (currentEditingScheduleId) {
            // 更新现有任务
            response = await fetch(`/api/schedules/${currentEditingScheduleId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(scheduleData)
            });
        } else {
            // 创建新任务
            response = await fetch('/api/schedules', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(scheduleData)
            });
        }
        
        if (!response.ok) {
            throw new Error(currentEditingScheduleId ? '更新任务失败' : '创建任务失败');
        }
        
        // 关闭模态框
        closeModal();
        
        // 重新获取任务列表
        await fetchSchedules();
        
        showNotification(currentEditingScheduleId ? '任务已更新' : '任务已创建', 'success');
    } catch (error) {
        console.error('保存定时任务错误:', error);
        showNotification('保存任务失败', 'error');
    }
}

// 切换任务启用状态
async function toggleSchedule(scheduleId) {
    const schedule = schedules.find(s => s.id === scheduleId);
    if (!schedule) return;
    
    const updatedSchedule = {
        ...schedule,
        enabled: !schedule.enabled
    };
    
    try {
        const response = await fetch(`/api/schedules/${scheduleId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(updatedSchedule)
        });
        
        if (!response.ok) {
            throw new Error('更新任务状态失败');
        }
        
        // 更新本地数据
        schedule.enabled = !schedule.enabled;
        
        // 重新渲染列表
        renderSchedulesList();
        
        showNotification(`任务已${schedule.enabled ? '启用' : '禁用'}`, 'success');
    } catch (error) {
        console.error('切换任务状态错误:', error);
        showNotification('更新任务状态失败', 'error');
    }
}

// 删除定时任务
async function deleteSchedule(scheduleId) {
    if (!confirm('确定要删除这个任务吗？')) return;
    
    try {
        const response = await fetch(`/api/schedules/${scheduleId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            throw new Error('删除任务失败');
        }
        
        // 从本地数据中移除
        schedules = schedules.filter(s => s.id !== scheduleId);
        
        // 重新渲染列表
        renderSchedulesList();
        
        showNotification('任务已删除', 'success');
    } catch (error) {
        console.error('删除任务错误:', error);
        showNotification('删除任务失败', 'error');
    }
}

// 显示通知
function showNotification(message, type = 'info') {
    // 创建toast元素
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    
    // 根据类型设置图标
    let icon = '';
    switch(type) {
        case 'success':
            icon = 'ri-check-line';
            break;
        case 'error':
            icon = 'ri-error-warning-line';
            break;
        default:
            icon = 'ri-information-line';
    }
    
    // 设置toast内容
    toast.innerHTML = `
        <i class="toast-icon ${icon}"></i>
        <div class="toast-message">${message}</div>
        <button class="toast-close" aria-label="关闭通知"><i class="ri-close-line"></i></button>
    `;
    
    // 获取toast容器
    const container = document.getElementById('toast-container');
    container.appendChild(toast);
    
    // 添加关闭事件
    const closeBtn = toast.querySelector('.toast-close');
    closeBtn.addEventListener('click', () => {
        closeToast(toast);
    });
    
    // 显示动画
    setTimeout(() => {
        toast.classList.add('show');
    }, 10);
    
    // 3秒后自动关闭
    setTimeout(() => {
        closeToast(toast);
    }, 3000);
}

// 关闭toast
function closeToast(toast) {
    // 添加隐藏动画
    toast.classList.add('hide');
    
    // 动画结束后移除元素
    setTimeout(() => {
        if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
        }
    }, 300);
}

// 初始化事件监听器
function initEventListeners() {
    // LED复选框切换
    $ledToggle.addEventListener('change', function(e) {
        // 阻止默认行为，我们手动控制复选框状态
        e.preventDefault();
        
        // 调用后端切换LED状态
        toggleLedStatus();
    });
    
    // 添加定时任务按钮
    $addScheduleBtn.addEventListener('click', openAddScheduleModal);
    
    // 关闭模态框按钮
    $closeModalBtn.addEventListener('click', closeModal);
    $cancelScheduleBtn.addEventListener('click', closeModal);
    
    // 表单提交
    $scheduleForm.addEventListener('submit', saveSchedule);
    
    // 星期选择
    $daySelects.forEach(el => {
        el.addEventListener('click', () => {
            el.classList.toggle('selected');
        });
    });
    
    // 点击模态框外部关闭
    $scheduleModal.addEventListener('click', (e) => {
        if (e.target === $scheduleModal) {
            closeModal();
        }
    });
    
    // 点击模态框内容区域阻止冒泡
    document.querySelector('.modal-content').addEventListener('click', (e) => {
        e.stopPropagation();
    });
    
    // 页面卸载时清理定时器
    window.addEventListener('beforeunload', () => {
        stopStatusRefresh();
    });
    
    // 页面可见性变化事件
    document.addEventListener('visibilitychange', handleVisibilityChange);
    
    // 给定时任务列表添加委托事件处理
    $schedulesList.addEventListener('click', handleScheduleActions);
}

// 处理页面可见性变化
function handleVisibilityChange() {
    if (document.hidden) {
        // 当页面不可见时，降低刷新频率为10秒一次，以节省资源
        console.log('页面不可见，调整设备状态刷新频率');
        restartStatusRefreshWithInterval(10000);
    } else {
        // 当页面重新可见时，恢复正常的刷新频率
        console.log('页面可见，恢复设备状态刷新频率');
        restartStatusRefreshWithInterval(2000);
        
        // 页面可见时立即刷新一次状态
        fetchLedStatus();
    }
} 