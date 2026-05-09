// 全局变量
let currentUserInfo = null;
let allUserInfos = [];
let currentLedStatus = false;
let schedules = [];
let currentEditingScheduleId = null;
let statusRefreshInterval = null; // 新增：用于存储状态刷新的定时器ID
let currentTheme = 'dark'; // 默认使用暗色主题
let currentBulbStyle = 'classic'; // 默认灯泡样式
let liquidAnimationId = null; // 液态光效动画ID
let analogClockInterval = null; // 点阵时钟定时器

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
const $notifyViaLzc = document.getElementById('notify-via-lzc');
const $testLzcNotifyBtn = document.getElementById('test-lzc-notify-btn');
const $themeToggle = document.getElementById('theme-toggle'); // 主题切换按钮
const $logoutBtn = document.getElementById('logout-btn'); // 登出按钮
const $bulbStyleToggle = document.getElementById('bulb-style-toggle'); // 灯泡样式切换按钮
const $bulbStyleModal = document.getElementById('bulb-style-modal'); // 灯泡样式模态框
const $closeBulbStyleModalBtn = document.getElementById('close-bulb-style-modal-btn'); // 关闭灯泡样式模态框按钮

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    initApp();
});

// 初始化应用
async function initApp() {
    // 初始化主题
    initTheme();

    // 获取用户信息
    await fetchUserInfo();

    // 获取用户偏好设置（包括LED样式）
    await fetchUserPreference();

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
    console.log('更新用户信息显示', {currentUserInfo, detailInfo});

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
    const classicToggle = document.querySelector('.bulb-classic #led-toggle');
    if (classicToggle) {
        classicToggle.disabled = true;
        classicToggle.checked = false;
    }

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

    // 更新经典灯泡的开关
    const classicToggle = document.querySelector('.bulb-classic #led-toggle');
    if (classicToggle) {
        classicToggle.checked = status;
        classicToggle.disabled = false; // 确保开关可用
    }

    // 更新熔岩灯的状态
    const lavaContainer = document.querySelector('.bulb-lava');
    if (lavaContainer) {
        if (status) {
            lavaContainer.classList.add('lamp-on');
        } else {
            lavaContainer.classList.remove('lamp-on');
        }
    }

    // 更新老式台灯的状态
    const vintageContainer = document.querySelector('.bulb-vintage');
    if (vintageContainer) {
        if (status) {
            vintageContainer.classList.add('lamp-on');
        } else {
            vintageContainer.classList.remove('lamp-on');
        }
    }

    // 更新液态光效的状态
    const liquidContainer = document.querySelector('.bulb-liquid');
    if (liquidContainer) {
        if (status) {
            liquidContainer.classList.add('lamp-on');
            startLiquidAnimation();
        } else {
            liquidContainer.classList.remove('lamp-on');
            stopLiquidAnimation();
        }
    }

    // 更新灯泡开关的状态
    const lightbulbToggle = document.querySelector('#lightbulb-toggle');
    if (lightbulbToggle) {
        lightbulbToggle.checked = status;
    }

    // 更新Single LED的状态
    const singleLedBulb = document.querySelector('.single-led-bulb');
    if (singleLedBulb) {
        if (status) {
            singleLedBulb.classList.add('light_up');
        } else {
            singleLedBulb.classList.remove('light_up');
        }
    }

    // 更新Neon Switch的状态
    const neonSwitchInput = document.querySelector('.neon-switch__input');
    if (neonSwitchInput) {
        neonSwitchInput.checked = status;
    }

    // 更新Fox Day-Night的状态（注意：反转逻辑）
    const foxToggle = document.getElementById('fox-toggle');
    if (foxToggle) {
        foxToggle.checked = !status; // LED开启=白天(unchecked), LED关闭=夜晚(checked)
    }

    $ledStatus.textContent = status ? '已开启' : '已关闭';
    $ledStatus.classList.remove('error');
}

// 切换LED状态
async function toggleLedStatus() {
    const newStatus = !currentLedStatus;
    const classicToggle = document.querySelector('.bulb-classic #led-toggle');

    try {
        // 更新UI以显示加载状态
        if (classicToggle) {
            classicToggle.disabled = true;
        }
        $ledStatus.textContent = '更新中…';

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
    } finally {
        // 无论成功或失败，都重新启用开关
        if (classicToggle) {
            classicToggle.disabled = false;
        }
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


        let operationIcon;
        let operationText;
        switch (schedule.operation) {
            case 'on':
                operationIcon = 'ri-lightbulb-flash-line';
                operationText = '开灯';
                break; // 不要忘记 break!
            case 'off': // 需要显式处理 'off'
                operationIcon = 'ri-lightbulb-line';
                operationText = '关灯';
                break;
            case 'shutdown':
                operationIcon = 'ri-shut-down-fill';
                operationText = '关机';
                break;
            case 'reboot':
                operationIcon = 'ri-restart-fill';
                operationText = '重启';
                break;
            default:
                // 可选：处理未知的 operation 值
                operationIcon = 'ri-question-mark'; // 示例：未知操作图标
                operationText = '未知';
                //console.warn(`Unknown schedule operation: ${schedule.operation}`);
        }


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
            <div class="schedule-serverchan ${schedule.notifyViaServerChan ? 'enabled' : ''}" title="Server酱通知">
                <i class="ri-notification-line"></i>
                <span>Server酱${schedule.notifyViaServerChan ? '已启用' : '未启用'}</span>
            </div>
            <div class="schedule-notify-lzc ${schedule.notifyViaLzc ? 'enabled' : ''}" title="懒猫内置通知">
                <i class="ri-notification-badge-line"></i>
                <span>懒猫内置${schedule.notifyViaLzc ? '已启用' : '未启用'}</span>
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
    $daySelects.forEach(el => {
        el.classList.remove('selected');
        el.setAttribute('aria-checked', 'false');
    });

    // 设置默认时间 (格式: HH:MM)
    document.getElementById('start-time').value = '22:00';

    // 设置默认操作为开灯
    document.getElementById('operation').value = 'on';
    updateLzcNotifyTestButton();

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
    
    // 设置Server酱通知选项
    document.getElementById('notify-via-server-chan').checked = schedule.notifyViaServerChan || false;
    
    // 设置懒猫内置通知选项
    document.getElementById('notify-via-lzc').checked = schedule.notifyViaLzc || false;
    updateLzcNotifyTestButton();

    // 设置重复的星期几
    $daySelects.forEach(el => {
        const day = parseInt(el.dataset.day);
        if (schedule.repeatDays && schedule.repeatDays.includes(day)) {
            el.classList.add('selected');
            el.setAttribute('aria-checked', 'true');
        } else {
            el.classList.remove('selected');
            el.setAttribute('aria-checked', 'false');
        }
    });

    // 显示模态框
    $scheduleModal.classList.add('show');
}

function updateLzcNotifyTestButton() {
    if (!$notifyViaLzc || !$testLzcNotifyBtn) return;
    $testLzcNotifyBtn.hidden = !$notifyViaLzc.checked;
}

async function testLzcNotification() {
    if (!$testLzcNotifyBtn) return;

    const previousHTML = $testLzcNotifyBtn.innerHTML;
    $testLzcNotifyBtn.disabled = true;
    $testLzcNotifyBtn.innerHTML = '<i class="ri-loader-4-line"></i>';

    try {
        const response = await fetch('/api/lzc-notification/test', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });

        const data = await response.json().catch(() => ({}));
        if (!response.ok) {
            throw new Error(data.error || '发送测试通知失败');
        }

        showNotification(data.message || '测试通知已发送', 'success');
    } catch (error) {
        console.error('发送懒猫内置测试通知错误:', error);
        showNotification(`测试通知失败: ${error.message}`, 'error');
    } finally {
        $testLzcNotifyBtn.disabled = false;
        $testLzcNotifyBtn.innerHTML = previousHTML;
    }
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
    const notifyViaServerChan = document.getElementById('notify-via-server-chan').checked;
    const notifyViaLzc = document.getElementById('notify-via-lzc').checked;

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
        notifyViaServerChan,
        notifyViaLzc,
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
        notifyViaServerChan: schedule.notifyViaServerChan || false,
        notifyViaLzc: schedule.notifyViaLzc || false,
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
    // LED开关 - 监听经典灯泡的开关
    const classicToggle = document.querySelector('.bulb-classic #led-toggle');
    if (classicToggle) {
        classicToggle.addEventListener('change', toggleLedStatus);
    }

    // 添加任务按钮
    $addScheduleBtn.addEventListener('click', openAddScheduleModal);

    // 关闭模态框按钮
    $closeModalBtn.addEventListener('click', closeModal);
    $cancelScheduleBtn.addEventListener('click', closeModal);

    // 保存任务
    $scheduleForm.addEventListener('submit', saveSchedule);

    if ($notifyViaLzc) {
        $notifyViaLzc.addEventListener('change', updateLzcNotifyTestButton);
    }

    if ($testLzcNotifyBtn) {
        $testLzcNotifyBtn.addEventListener('click', testLzcNotification);
    }

    // 重复日期选择
    $daySelects.forEach(el => {
        el.addEventListener('click', () => {
            toggleDaySelect(el);
        });

        // Keyboard support for day selection
        el.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleDaySelect(el);
            }
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
        if (e.key === 'Escape') {
            if ($scheduleModal.classList.contains('show')) {
                closeModal();
            }
            if ($bulbStyleModal && $bulbStyleModal.classList.contains('show')) {
                closeBulbStyleModal();
            }
        }
    });

    // 监听页面可见性变化
    document.addEventListener('visibilitychange', handleVisibilityChange);
    
    // 主题切换按钮
    $themeToggle.addEventListener('click', toggleTheme);
    
    // 登出按钮
    if ($logoutBtn) {
        $logoutBtn.addEventListener('click', handleLogout);
    }

    // 灯泡样式切换按钮
    if ($bulbStyleToggle) {
        $bulbStyleToggle.addEventListener('click', openBulbStyleModal);
    }

    // 关闭灯泡样式模态框
    if ($closeBulbStyleModalBtn) {
        $closeBulbStyleModalBtn.addEventListener('click', closeBulbStyleModal);
    }

    // 灯泡样式模态框外部点击关闭
    if ($bulbStyleModal) {
        $bulbStyleModal.addEventListener('click', (e) => {
            if (e.target === $bulbStyleModal) {
                closeBulbStyleModal();
            }
        });
    }

    // 灯泡样式卡片选择
    const bulbStyleCards = document.querySelectorAll('.bulb-style-card');
    bulbStyleCards.forEach(card => {
        card.addEventListener('click', () => {
            const style = card.dataset.style;
            updateUserPreference(style);

            // 更新选中状态
            bulbStyleCards.forEach(c => c.classList.remove('active'));
            card.classList.add('active');

            // 延迟关闭模态框，让用户看到选中效果
            setTimeout(() => {
                closeBulbStyleModal();
            }, 300);
        });
    });

    // 灯泡样式选择器（移除旧的内联选择器代码）
    const bulbStyleOptions = document.querySelectorAll('.bulb-style-option');
    if (bulbStyleOptions.length > 0) {
        // 如果存在旧的内联选择器，保留事件监听（向后兼容）
        bulbStyleOptions.forEach(option => {
            option.addEventListener('click', () => {
                const style = option.dataset.style;
                updateUserPreference(style);

                // 更新选中状态
                bulbStyleOptions.forEach(opt => opt.classList.remove('active'));
                option.classList.add('active');
            });
        });
    }

    // 老式台灯拉绳点击事件
    const vintagePullCord = document.querySelector('.lamp__pull-cord');
    if (vintagePullCord) {
        vintagePullCord.addEventListener('click', (e) => {
            e.preventDefault();
            toggleLedStatus();
        });
        vintagePullCord.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // 液态光效点击事件
    const liquidCanvas = document.getElementById('liquid-canvas');
    const liquidContainer = document.querySelector('.bulb-liquid');
    if (liquidContainer) {
        liquidContainer.addEventListener('click', () => {
            toggleLedStatus();
        });
        liquidContainer.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // 老式台灯容器点击事件（整个容器可点击）
    const vintageContainer = document.querySelector('.bulb-vintage');
    if (vintageContainer) {
        vintageContainer.addEventListener('click', (e) => {
            // 如果点击的是拉绳，让拉绳的事件处理
            if (e.target.closest('.lamp__pull-cord')) {
                return;
            }
            toggleLedStatus();
        });
        vintageContainer.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // 熔岩灯点击事件
    const lavaContainer = document.querySelector('.bulb-lava');
    if (lavaContainer) {
        lavaContainer.addEventListener('click', () => {
            toggleLedStatus();
        });
        lavaContainer.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // 灯泡开关点击事件
    const lightbulbToggle = document.querySelector('#lightbulb-toggle');
    if (lightbulbToggle) {
        lightbulbToggle.addEventListener('change', toggleLedStatus);
    }

    // 点阵时钟点击事件
    const analogContainer = document.querySelector('.bulb-analog');
    if (analogContainer) {
        analogContainer.addEventListener('click', () => {
            toggleLedStatus();
        });
        analogContainer.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // Single LED 点击事件
    const singleLedContainer = document.querySelector('.bulb-single-led');
    const singleLedButton = document.querySelector('.single-led-button');
    const singleLedBulb = document.querySelector('.single-led-bulb');

    if (singleLedButton) {
        singleLedButton.addEventListener('click', (e) => {
            e.stopPropagation();
            toggleLedStatus();
        });
    }

    if (singleLedBulb) {
        singleLedBulb.addEventListener('click', () => {
            toggleLedStatus();
        });
    }

    if (singleLedContainer) {
        singleLedContainer.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleLedStatus();
            }
        });
    }

    // Neon Switch 点击事件
    const neonSwitchInput = document.querySelector('.neon-switch__input');
    if (neonSwitchInput) {
        neonSwitchInput.addEventListener('change', toggleLedStatus);
    }

    // Fox Day-Night 点击事件
    const foxToggle = document.getElementById('fox-toggle');
    const foxSwitch = document.querySelector('.fox-switch');

    if (foxToggle) {
        foxToggle.addEventListener('change', toggleLedStatus);
    }

    // 让fox-switch也可以点击切换
    if (foxSwitch && foxToggle) {
        foxSwitch.addEventListener('click', (e) => {
            e.stopPropagation();
            foxToggle.checked = !foxToggle.checked;
            toggleLedStatus();
        });
    }
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

// 处理登出
function handleLogout() {
    if (confirm('确定要退出登录吗?')) {
        // 显示加载状态
        if ($logoutBtn) {
            $logoutBtn.innerHTML = '<i class="ri-loader-4-line" style="animation: spin 1s linear infinite;"></i><span>退出中…</span>';
            $logoutBtn.disabled = true;
        }
        
        // 跳转到登出页面
        window.location.href = '/logout';
    }
}

// 切换周几选择
function toggleDaySelect(el) {
    const isSelected = el.classList.contains('selected');
    el.classList.toggle('selected');
    el.setAttribute('aria-checked', !isSelected);
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

// 获取用户偏好设置
async function fetchUserPreference() {
    try {
        const response = await fetch('/api/user/preference');
        if (!response.ok) {
            // 如果获取失败，使用默认样式
            console.log('获取用户偏好失败，使用默认样式');
            applyBulbStyle('classic');
            return;
        }

        const data = await response.json();
        console.log('用户偏好:', data);

        if (data.bulb_style) {
            currentBulbStyle = data.bulb_style;
            applyBulbStyle(currentBulbStyle);
        }
    } catch (error) {
        console.error('获取用户偏好错误:', error);
        // 使用默认样式
        applyBulbStyle('classic');
    }
}

// 应用灯泡样式
function applyBulbStyle(style) {
    const containers = document.querySelectorAll('.bulb-container');

    // 隐藏所有灯泡
    containers.forEach(container => {
        container.classList.remove('active');
    });

    // 显示选中的灯泡
    const targetContainer = document.querySelector(`.bulb-${style}`);
    if (targetContainer) {
        targetContainer.classList.add('active');
    }

    currentBulbStyle = style;

    // 更新样式选择器的激活状态（内联选择器，如果存在）
    const bulbStyleOptions = document.querySelectorAll('.bulb-style-option');
    bulbStyleOptions.forEach(option => {
        if (option.dataset.style === style) {
            option.classList.add('active');
        } else {
            option.classList.remove('active');
        }
    });

    // 更新模态框中卡片的激活状态
    const bulbStyleCards = document.querySelectorAll('.bulb-style-card');
    bulbStyleCards.forEach(card => {
        if (card.dataset.style === style) {
            card.classList.add('active');
        } else {
            card.classList.remove('active');
        }
    });

    // 同步两个开关的状态
    syncBulbToggles();

    // 启动或停止点阵时钟
    if (style === 'analog') {
        startAnalogClock();
    } else {
        stopAnalogClock();
    }

    // 如果是液态光效，重新初始化canvas
    if (style === 'liquid') {
        setTimeout(() => {
            resizeLiquidCanvas();
            if (currentLedStatus) {
                startLiquidAnimation();
            }
        }, 100);
    }

    console.log(`已应用灯泡样式: ${style}`);
}

// 同步两个灯泡的开关状态
function syncBulbToggles() {
    const classicToggle = document.querySelector('.bulb-classic #led-toggle');
    const lavaContainer = document.querySelector('.bulb-lava');
    const vintageContainer = document.querySelector('.bulb-vintage');
    const liquidContainer = document.querySelector('.bulb-liquid');
    const singleLedBulb = document.querySelector('.single-led-bulb');
    const neonSwitchInput = document.querySelector('.neon-switch__input');
    const foxToggle = document.getElementById('fox-toggle');

    if (!classicToggle) return;

    // 根据经典灯泡的状态同步其他灯的显示
    if (classicToggle.checked) {
        if (lavaContainer) lavaContainer.classList.add('lamp-on');
        if (vintageContainer) vintageContainer.classList.add('lamp-on');
        if (liquidContainer) {
            liquidContainer.classList.add('lamp-on');
            startLiquidAnimation();
        }
        if (singleLedBulb) singleLedBulb.classList.add('light_up');
        if (neonSwitchInput) neonSwitchInput.checked = true;
        if (foxToggle) foxToggle.checked = false; // LED开启 = 白天（unchecked）
    } else {
        if (lavaContainer) lavaContainer.classList.remove('lamp-on');
        if (vintageContainer) vintageContainer.classList.remove('lamp-on');
        if (liquidContainer) {
            liquidContainer.classList.remove('lamp-on');
            stopLiquidAnimation();
        }
        if (singleLedBulb) singleLedBulb.classList.remove('light_up');
        if (neonSwitchInput) neonSwitchInput.checked = false;
        if (foxToggle) foxToggle.checked = true; // LED关闭 = 夜晚（checked）
    }
}

// 更新用户偏好设置
async function updateUserPreference(bulbStyle) {
    try {
        const response = await fetch('/api/user/preference', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                bulb_style: bulbStyle
            })
        });

        if (!response.ok) {
            throw new Error('更新用户偏好失败');
        }

        const data = await response.json();
        console.log('用户偏好已更新:', data);

        // 应用新样式
        applyBulbStyle(bulbStyle);
        showNotification('灯泡样式已更新', 'success');
    } catch (error) {
        console.error('更新用户偏好错误:', error);
        showNotification('更新灯泡样式失败', 'error');
    }
} 
// ===================================
// 液态光效动画
// ===================================

let liquidCanvas, liquidCtx, liquidW, liquidH;
let liquidArr = [];
let liquidCnt = 0;
let isLiquidRunning = false;
let resizeTimeout = null;

function initLiquidCanvas() {
    liquidCanvas = document.getElementById('liquid-canvas');
    if (!liquidCanvas) return;

    liquidCtx = liquidCanvas.getContext('2d');
    resizeLiquidCanvas();

    // Debounced resize handler
    window.addEventListener('resize', () => {
        if (resizeTimeout) clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(resizeLiquidCanvas, 150);
    });
}

function resizeLiquidCanvas() {
    if (!liquidCanvas) return;

    const container = liquidCanvas.parentElement;
    if (!container) return;

    // 如果容器是隐藏的，使用父级section的尺寸
    if (container.offsetParent === null) {
        const section = container.closest('.led-status-section');
        if (section) {
            liquidW = liquidCanvas.width = section.clientWidth;
            liquidH = liquidCanvas.height = Math.max(section.clientHeight, 400);
        }
    } else {
        liquidW = liquidCanvas.width = container.clientWidth || 400;
        liquidH = liquidCanvas.height = container.clientHeight || 400;
    }
}

function startLiquidAnimation() {
    if (isLiquidRunning) return;

    // Check for reduced motion preference
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
        return;
    }

    isLiquidRunning = true;
    liquidCnt = 0;
    liquidArr = [];
    animateLiquid();
}

function stopLiquidAnimation() {
    isLiquidRunning = false;
    if (liquidAnimationId) {
        cancelAnimationFrame(liquidAnimationId);
        liquidAnimationId = null;
    }
    if (liquidCtx) {
        liquidCtx.clearRect(0, 0, liquidW, liquidH);
    }
    liquidArr = [];
}

function animateLiquid() {
    if (!isLiquidRunning) return;

    // Stop animation if canvas is not visible (performance)
    if (liquidCanvas && liquidCanvas.offsetParent === null) {
        stopLiquidAnimation();
        return;
    }

    liquidCnt++;
    if (liquidCnt % 6 === 0) drawLiquid();

    liquidAnimationId = requestAnimationFrame(animateLiquid);
}

function drawLiquid() {
    if (!liquidCtx) return;
    
    const _w = liquidW * 0.5;
    const _h = liquidH * 0.5;
    
    const splot = {
        x: rng(_w - 300, _w + 300),
        y: rng(_h - 300, _h + 300),
        r: rng(20, 60),
        spX: rng(-1, 1),
        spY: rng(-1, 1)
    };

    liquidArr.push(splot);
    
    while (liquidArr.length > 80) {
        liquidArr.shift();
    }
    
    liquidCtx.clearRect(0, 0, liquidW, liquidH);

    for (let i = 0; i < liquidArr.length; i++) {
        const splot = liquidArr[i];
        
        liquidCtx.fillStyle = rndCol();
        liquidCtx.beginPath();
        liquidCtx.arc(splot.x, splot.y, splot.r, 0, Math.PI * 2, true);
        liquidCtx.shadowBlur = 80;
        liquidCtx.shadowOffsetX = 2;
        liquidCtx.shadowOffsetY = 2;
        liquidCtx.shadowColor = rndCol();
        liquidCtx.globalCompositeOperation = 'lighter';
        liquidCtx.fill();

        splot.x = splot.x + splot.spX;
        splot.y = splot.y + splot.spY;
        splot.r = splot.r * 0.96;
    }
}

function rndCol() {
    const r = Math.floor(Math.random() * 180);
    const g = Math.floor(Math.random() * 60);
    const b = Math.floor(Math.random() * 100);
    return `rgb(${r}, ${g}, ${b})`;
}

function rng(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

// 在页面加载时初始化liquid canvas
document.addEventListener('DOMContentLoaded', () => {
    setTimeout(() => {
        initLiquidCanvas();
    }, 500);
});

// ===================================
// 点阵时钟功能
// ===================================

function setAnalogTime() {
    const analogClock = document.querySelector('.analog-clock');
    if (!analogClock) return;

    const now = new Date();
    const hours = now.getHours();
    const minutes = now.getMinutes();
    const seconds = now.getSeconds();

    // 转换为两位数字符串
    const h = hours.toString().padStart(2, '0');
    const m = minutes.toString().padStart(2, '0');
    const s = seconds.toString().padStart(2, '0');

    // 设置data-time属性为HHMMSS格式
    analogClock.setAttribute('data-time', h + m + s);
}

function startAnalogClock() {
    if (analogClockInterval) return;

    // 立即设置一次时间
    setAnalogTime();

    // 每秒更新一次
    analogClockInterval = setInterval(setAnalogTime, 1000);
}

function stopAnalogClock() {
    if (analogClockInterval) {
        clearInterval(analogClockInterval);
        analogClockInterval = null;
    }
}

// ===================================
// 灯泡样式模态框
// ===================================

function openBulbStyleModal() {
    if ($bulbStyleModal) {
        $bulbStyleModal.classList.add('show');

        // 更新模态框中卡片的选中状态
        const bulbStyleCards = document.querySelectorAll('.bulb-style-card');
        bulbStyleCards.forEach(card => {
            if (card.dataset.style === currentBulbStyle) {
                card.classList.add('active');
            } else {
                card.classList.remove('active');
            }
        });
    }
}

function closeBulbStyleModal() {
    if ($bulbStyleModal) {
        $bulbStyleModal.classList.remove('show');
    }
}
