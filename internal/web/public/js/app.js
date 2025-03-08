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
            // 如果没有设备信息，隐藏整个区域
            deviceInfoSection.style.display = 'none';
        }
    } catch (error) {
        console.error('更新用户信息显示错误:', error);
    }
}

// 获取LED状态
async function fetchLedStatus() {
    try {
        const response = await fetch('/api/led-status');
        if (!response.ok) {
            throw new Error('获取LED状态失败');
        }
        
        const data = await response.json();
        
        // 添加调试日志
        console.log('LED状态API响应:', data);
        
        if (typeof data.status !== 'boolean') {
            throw new Error('无效的LED状态数据');
        }
        
        updateLedStatus(data.status);
    } catch (error) {
        console.error('获取LED状态错误:', error);
        handleLedStatusError('获取状态失败');
    }
}

// 处理LED状态错误
function handleLedStatusError(errorMsg) {
    // 日志记录错误
    console.error('LED状态错误:', errorMsg);
    
    // 更新UI以显示错误状态
    $ledStatus.textContent = '状态未知';
    $ledStatus.classList.add('error');
    
    // 禁用开关
    $ledToggle.disabled = true;
    $ledToggle.checked = false;
    
    // 显示错误通知
    showNotification('无法获取LED状态', 'error');
    
    // 尝试在更长的延迟后重新获取
    setTimeout(() => {
        // 停止当前的刷新间隔（如果有）
        stopStatusRefresh();
        
        // 尝试重新获取
        fetchLedStatus()
            .then(() => {
                // 如果成功，恢复正常的刷新间隔
                restartStatusRefreshWithInterval(2000);
            })
            .catch(() => {
                // 如果仍然失败，使用更长的刷新间隔
                restartStatusRefreshWithInterval(5000);
            });
    }, 2000);
}

// 用指定间隔重启状态刷新
function restartStatusRefreshWithInterval(interval) {
    // 停止现有的刷新
    stopStatusRefresh();
    
    // 启动新的刷新计时器
    statusRefreshInterval = setInterval(() => {
        fetchLedStatus().catch(error => {
            console.error('状态刷新错误:', error);
        });
    }, interval);
}

// 开始状态刷新
function startStatusRefresh() {
    // 如果已经有一个刷新间隔，先停止它
    stopStatusRefresh();
    
    // 设置新的刷新间隔
    statusRefreshInterval = setInterval(() => {
        fetchLedStatus().catch(error => {
            console.error('状态刷新错误:', error);
        });
    }, 2000);
    
    // 添加页面可见性变化的处理程序
    document.addEventListener('visibilitychange', handleVisibilityChange);
}

// 停止状态刷新
function stopStatusRefresh() {
    if (statusRefreshInterval) {
        clearInterval(statusRefreshInterval);
        statusRefreshInterval = null;
    }
}

// 更新LED状态UI
function updateLedStatus(status) {
    currentLedStatus = status;
    $ledToggle.checked = status;
    $ledStatus.textContent = status ? '已开启' : '已关闭';
    $ledStatus.classList.remove('error');
    $ledToggle.disabled = false;
}

// 切换LED状态
async function toggleLedStatus() {
    const newStatus = !currentLedStatus;
    
    try {
        // 更新UI以显示加载状态
        $ledToggle.disabled = true;
        $ledStatus.textContent = '更新中...';
        
        // 构建请求URL
        const url = `/ledcontrol?turn=${newStatus ? 'on' : 'off'}`;
        
        // 发送请求
        const response = await fetch(url);
        
        if (!response.ok) {
            throw new Error('切换LED状态失败');
        }
        
        // 更新UI
        updateLedStatus(newStatus);
        
        // 显示通知
        showNotification(`灯已${newStatus ? '开启' : '关闭'}`, 'success');
    } catch (error) {
        console.error('切换LED状态错误:', error);
        
        // 恢复原状态
        updateLedStatus(currentLedStatus);
        
        // 显示错误通知
        showNotification('操作失败', 'error');
    }
}

// 获取定时任务
async function fetchSchedules() {
    try {
        const response = await fetch('/api/schedules');
        if (!response.ok) {
            throw new Error('获取定时任务失败');
        }
        
        schedules = await response.json();
        console.log('获取的定时任务:', schedules);
        
        // 渲染任务列表
        renderSchedulesList();
    } catch (error) {
        console.error('获取定时任务错误:', error);
        showNotification('获取定时任务失败', 'error');
    }
}

// 渲染定时任务列表
function renderSchedulesList() {
    // 清空列表内容
    $schedulesList.innerHTML = '';
    
    // 如果没有任务，显示空状态
    if (!schedules || schedules.length === 0) {
        $schedulesList.innerHTML = `
            <div class="empty-state">
                <i class="ri-time-line"></i>
                <p>暂无定时任务，点击右上角添加</p>
            </div>
        `;
        return;
    }
    
    // 渲染每个任务
    schedules.forEach(schedule => {
        // 格式化时间
        const hour = schedule.hour || 0;
        const minute = schedule.minute || 0;
        const timeFormatted = `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
        
        // 确定操作类型图标和文本
        const operationIcon = schedule.operation === 'on' ? 'ri-lightbulb-line' : 'ri-lightbulb-flash-line';
        const operationText = schedule.operation === 'on' ? '开灯' : '关灯';
        
        // 创建任务元素
        const scheduleElement = document.createElement('div');
        scheduleElement.className = 'schedule-item';
        
        // 添加启用/禁用状态类
        if (!schedule.enabled) {
            scheduleElement.classList.add('disabled');
        }
        
        scheduleElement.innerHTML = `
            <div class="schedule-header">
                <h3 class="schedule-name">${schedule.name}</h3>
                <div class="schedule-actions">
                    <button class="edit-btn" data-id="${schedule.id}" aria-label="编辑任务">
                        <i class="ri-edit-line"></i>
                    </button>
                    <button class="delete-btn" data-id="${schedule.id}" aria-label="删除任务">
                        <i class="ri-delete-bin-line"></i>
                    </button>
                </div>
            </div>
            <div class="schedule-time">
                <i class="ri-time-line"></i>
                <span>${timeFormatted}</span>
            </div>
            <div class="schedule-operation">
                <i class="${operationIcon}"></i>
                <span>${operationText}</span>
            </div>
            <div class="schedule-repeat">
                <i class="ri-repeat-line"></i>
                <span>${renderWeekdays(schedule.repeatDays)}</span>
            </div>
            <div class="schedule-creator" title="创建者">
                <i class="ri-user-line"></i>
                <span>${schedule.creatorId || '未知'}</span>
            </div>
            <div class="schedule-toggle">
                <div class="toggle-switch small">
                    <input type="checkbox" id="toggle-${schedule.id}" ${schedule.enabled ? 'checked' : ''}>
                    <label for="toggle-${schedule.id}" class="toggle-label"></label>
                </div>
            </div>
        `;
        
        // 添加到列表
        $schedulesList.appendChild(scheduleElement);
        
        // 添加事件监听器
        const toggleInput = scheduleElement.querySelector(`#toggle-${schedule.id}`);
        toggleInput.addEventListener('change', () => toggleSchedule(schedule.id));
        
        // 编辑按钮
        const editBtn = scheduleElement.querySelector('.edit-btn');
        editBtn.addEventListener('click', () => openEditScheduleModal(schedule.id));
        
        // 删除按钮
        const deleteBtn = scheduleElement.querySelector('.delete-btn');
        deleteBtn.addEventListener('click', () => deleteSchedule(schedule.id));
    });
}

// 渲染星期几
function renderWeekdays(days) {
    if (!days || days.length === 0) return '无重复';
    
    const weekdayNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    
    // 如果包含所有日期
    if (days.length === 7) return '每天';
    
    // 如果是工作日
    if (days.length === 5 && days.includes(1) && days.includes(2) && days.includes(3) && days.includes(4) && days.includes(5)) {
        return '工作日';
    }
    
    return days.map(day => weekdayNames[day]).join(', ');
}

// 打开添加任务模态框
function openAddScheduleModal() {
    // 重置表单
    $scheduleForm.reset();
    $modalTitle.textContent = '添加定时任务';
    currentEditingScheduleId = null;
    
    // 重置选择的星期几
    $daySelects.forEach(el => el.classList.remove('selected'));
    
    // 设置默认时间 (格式: HH:MM)
    document.getElementById('start-time').value = '22:00';
    
    // 设置默认操作为开灯
    document.getElementById('operation').value = 'on';
    
    // 显示模态框
    $scheduleModal.classList.add('show');
}

// 打开编辑任务模态框
function openEditScheduleModal(scheduleId) {
    const schedule = schedules.find(s => s.id === scheduleId);
    if (!schedule) return;
    
    // 设置当前正在编辑的任务ID
    currentEditingScheduleId = scheduleId;
    $modalTitle.textContent = '编辑定时任务';
    
    // 填充表单数据
    document.getElementById('schedule-name').value = schedule.name;
    
    // 设置时间 - 使用小时和分钟
    const hour = schedule.hour || 0;
    const minute = schedule.minute || 0;
    document.getElementById('start-time').value = `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
    
    // 设置操作
    document.getElementById('operation').value = schedule.operation || 'on';
    
    // 设置允许他人编辑
    document.getElementById('allow-edit').checked = schedule.allowEdit;
    
    // 设置启用状态
    document.getElementById('schedule-enabled').checked = schedule.enabled;
    
    // 设置重复的星期几
    $daySelects.forEach(el => {
        const day = parseInt(el.dataset.day);
        if (schedule.repeatDays && schedule.repeatDays.includes(day)) {
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
    const timeString = document.getElementById('start-time').value;
    const operation = document.getElementById('operation').value;
    const allowEdit = document.getElementById('allow-edit').checked;
    const enabled = document.getElementById('schedule-enabled').checked;
    
    // 获取选中的星期
    const repeatDays = [];
    $daySelects.forEach(el => {
        if (el.classList.contains('selected')) {
            repeatDays.push(parseInt(el.dataset.day));
        }
    });
    
    // 直接从时间字符串中提取小时和分钟
    const [hours, minutes] = timeString.split(':').map(Number);
    
    // 构建任务数据 - 直接使用小时和分钟数
    const scheduleData = {
        name,
        hour: hours,
        minute: minutes,
        repeatDays,
        allowEdit,
        enabled,
        operation
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
            const errorData = await response.json();
            throw new Error(errorData.error || (currentEditingScheduleId ? '更新任务失败' : '创建任务失败'));
        }
        
        // 关闭模态框
        closeModal();
        
        // 重新获取任务列表
        await fetchSchedules();
        
        showNotification(currentEditingScheduleId ? '任务已更新' : '任务已创建', 'success');
    } catch (error) {
        console.error('保存定时任务错误:', error);
        showNotification(`保存任务失败: ${error.message}`, 'error');
    }
}

// 切换任务启用状态
async function toggleSchedule(scheduleId) {
    const schedule = schedules.find(s => s.id === scheduleId);
    if (!schedule) return;
    
    // 创建更新数据
    const updatedSchedule = {
        name: schedule.name,
        hour: schedule.hour || 0,
        minute: schedule.minute || 0,
        repeatDays: schedule.repeatDays || [],
        allowEdit: schedule.allowEdit,
        enabled: !schedule.enabled,
        operation: schedule.operation
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
            const errorData = await response.json();
            throw new Error(errorData.error || '更新任务状态失败');
        }
        
        // 重新获取任务列表而不是直接更新本地数据
        await fetchSchedules();
        
        showNotification(`任务已${!schedule.enabled ? '启用' : '禁用'}`, 'success');
    } catch (error) {
        console.error('切换任务状态错误:', error);
        showNotification(`更新任务状态失败: ${error.message}`, 'error');
        
        // 恢复UI状态（因为操作失败）
        const toggleInput = document.querySelector(`#toggle-${scheduleId}`);
        if (toggleInput) {
            toggleInput.checked = schedule.enabled;
        }
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
            const errorData = await response.json();
            throw new Error(errorData.error || '删除任务失败');
        }
        
        // 重新获取任务列表
        await fetchSchedules();
        
        showNotification('任务已删除', 'success');
    } catch (error) {
        console.error('删除任务错误:', error);
        showNotification(`删除任务失败: ${error.message}`, 'error');
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
    // LED开关
    $ledToggle.addEventListener('change', toggleLedStatus);
    
    // 添加任务按钮
    $addScheduleBtn.addEventListener('click', openAddScheduleModal);
    
    // 关闭模态框按钮
    $closeModalBtn.addEventListener('click', closeModal);
    $cancelScheduleBtn.addEventListener('click', closeModal);
    
    // 保存任务
    $scheduleForm.addEventListener('submit', saveSchedule);
    
    // 重复日期选择
    $daySelects.forEach(el => {
        el.addEventListener('click', () => {
            el.classList.toggle('selected');
        });
    });
    
    // 模态框外部点击关闭
    $scheduleModal.addEventListener('click', (e) => {
        if (e.target === $scheduleModal) {
            closeModal();
        }
    });
    
    // 添加键盘Escape键关闭模态框
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && $scheduleModal.classList.contains('show')) {
            closeModal();
        }
    });
    
    // 监听页面可见性变化
    document.addEventListener('visibilitychange', handleVisibilityChange);
}

// 处理页面可见性变化
function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
        // 页面变为可见时，立即获取最新状态并重启定时刷新
        fetchLedStatus().catch(console.error);
        fetchSchedules().catch(console.error);
        
        // 如果刷新计时器不存在，重新启动它
        if (!statusRefreshInterval) {
            startStatusRefresh();
        }
    } else {
        // 页面不可见时，停止刷新以节省资源
        stopStatusRefresh();
    }
} 