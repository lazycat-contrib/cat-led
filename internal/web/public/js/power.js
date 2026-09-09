(() => {
    const $ = id => document.getElementById(id);
    const dialog = $('power-dialog');
    const form = $('power-form');
    let state = null;
    let busy = false;
    let loaded = false;
    let available = false;
    let isAdmin = false;
    let refreshVersion = 0;
    let loadError = '';
    let mutating = false;
    let closeTimer;
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const labels = {disabled:'计划已停用', preparing:'正在准备计划', active:'计划已启用', waiting_wake:'关机指令已发送，等待开机', shutdown_sent:'关机指令已发送', completed:'单次计划已结束', missed:'已跳过错过的时间', error:'计划已停止，请检查后重新保存'};
    const hasShutdown = spec => spec.shutdown_enabled !== false;
    const hasWake = spec => spec.wake_enabled !== false;
    const offEnabled = () => $('power-off-enabled').checked;
    const onEnabled = () => $('power-on-enabled').checked;
    const hasTime = value => value && !value.startsWith('0001-');
    const localValue = value => {
        const date = new Date(value);
        return new Date(date.getTime() - date.getTimezoneOffset()*60000).toISOString().slice(0,16);
    };
    function setBusy(value) {
        busy = value;
        form.setAttribute('aria-busy', String(value));
        $('power-fields').disabled = value || !loaded;
        const any = offEnabled() || onEnabled();
        const canDisable = state?.creator && (state.phase !== 'disabled' || state.error);
        $('power-save').disabled = value || !loaded || (onEnabled() && !available) || (!any && !canDisable);
        $('power-cancel-plan').disabled = value;
        $('power-close').disabled = value && mutating;
        $('power-save').textContent = value ? '正在保存…' : (!any && canDisable ? '停用计划' : '保存设置');
    }
    function showError(message) {
        $('power-form-error').textContent = message;
        $('power-form-error').hidden = !message;
        if (message && dialog.open) $('power-form-error').focus();
    }
    function updateOptions() {
        const off = offEnabled(), on = onEnabled(), any = off || on;
        $('power-options').hidden = !any;
        $('power-disabled-hint').hidden = any;
        $('power-off-field').hidden = !off;
        $('power-on-field').hidden = !on;
        $('power-off').disabled = !off;
        $('power-on').disabled = !on;
        $('power-off').required = off;
        $('power-on').required = on;
        $('power-timezone').required = any;
        $('power-timezone').disabled = !any;
        $('power-mode').disabled = !any;
        $('power-weekdays').disabled = !any;
        $('power-off-enabled').setAttribute('aria-expanded', String(off));
        $('power-on-enabled').setAttribute('aria-expanded', String(on));
        $('power-wake-note').hidden = !on;
        const weekly = $('power-mode').value === 'weekly';
        $('power-weekdays').hidden = !weekly;
        $('power-zone-hint').hidden = weekly;
        $('power-days-label').textContent = off ? '在哪些天关机' : '在哪些天开机';
        $('power-time-hint').textContent = off && on
            ? (weekly && $('power-on').value <= $('power-off').value ? '开机时间为次日。' : '开机至少比关机晚五分钟。')
            : '至少提前两分钟。';
        setBusy(busy);
    }
    function changeMode() {
        const weekly = $('power-mode').value === 'weekly';
        $('power-off').type = weekly ? 'time' : 'datetime-local';
        $('power-on').type = weekly ? 'time' : 'datetime-local';
        $('power-off').value = weekly ? '23:00' : localValue(Date.now()+3600000);
        $('power-on').value = weekly ? '07:00' : localValue(Date.now()+9*3600000);
        updateOptions();
    }
    function formatDate(value, timezone) {
        return new Intl.DateTimeFormat('zh-CN',{timeZone:timezone,month:'long',day:'numeric',weekday:'short',hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(value));
    }
    function renderState(next, fill = false) {
        state = next;
        const active = ['active','waiting_wake','shutdown_sent'].includes(next.phase);
        $('power-phase').textContent = next.creator ? labels[next.phase] || '状态未知' : '尚未设置计划';
        $('power-plan-button').dataset.active = String(active);
        $('power-plan-button').title = active ? '定时开关机 · 已启用' : '定时开关机';
        const spec = next.spec || {};
        const timezone = spec.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
        $('power-next').hidden = !active;
        $('power-next-off-row').hidden = !hasShutdown(spec) || !hasTime(next.shutdown_at);
        $('power-next-on-row').hidden = !hasWake(spec) || !hasTime(next.wake_at);
        if (hasTime(next.shutdown_at)) $('power-next-off').textContent = formatDate(next.shutdown_at,timezone);
        if (hasTime(next.wake_at)) $('power-next-on').textContent = formatDate(next.wake_at,timezone);
        $('power-cancel-plan').hidden = !next.creator || (next.phase === 'disabled' && !next.error);
        if (fill) {
            $('power-mode').value = spec.mode || 'once';
            changeMode();
            $('power-timezone').value = timezone;
            const weekly = spec.mode === 'weekly';
            if (weekly ? spec.shutdown_time : hasTime(spec.shutdown_at)) $('power-off').value = weekly ? spec.shutdown_time : localValue(spec.shutdown_at);
            if (weekly ? spec.wake_time : hasTime(spec.wake_at)) $('power-on').value = weekly ? spec.wake_time : localValue(spec.wake_at);
            document.querySelectorAll('#power-weekdays input').forEach(input => { input.checked = (spec.weekdays || []).includes(Number(input.value)); });
            $('power-off-enabled').checked = active && hasShutdown(spec) && (weekly || next.phase === 'active');
            $('power-on-enabled').checked = active && hasWake(spec);
            updateOptions();
        }
        renderTasks();
    }
    function renderTasks() {
        const list = $('schedules-list');
        list.querySelectorAll('[data-power-task]').forEach(node => node.remove());
        if (isAdmin && loadError) {
            const message = document.createElement('p');
            message.dataset.powerTask = 'error';
            message.className = 'power-task-error';
            message.textContent = '电源任务暂时无法读取，请从顶部按钮重试。';
            list.append(message);
        }
        if (isAdmin && state?.creator) {
            const spec = state.spec || {};
            for (const kind of ['off','on']) {
                const wake = kind === 'on';
                if (!(wake ? hasWake(spec) : hasShutdown(spec))) continue;
                const target = wake ? state.wake_at : state.shutdown_at;
                const weekly = spec.mode === 'weekly';
                const enabled = state.phase === 'active' || (state.phase === 'waiting_wake' && wake);
                const row = document.createElement('div');
                row.className = 'schedule-item power-task';
                row.dataset.powerTask = kind;
                row.innerHTML = `<div class="schedule-header"><div class="schedule-title"><h3 class="schedule-name"></h3></div><div class="schedule-actions"><button type="button" class="edit-btn"><i class="ri-edit-line" aria-hidden="true"></i></button></div></div><div class="schedule-meta"><div class="schedule-time"><i class="ri-time-line" aria-hidden="true"></i><span></span></div><div class="schedule-operation"><i class="${wake ? 'ri-alarm-line' : 'ri-timer-2-line'}" aria-hidden="true"></i><span>${wake ? 'RTC 开机' : '定时关机'}</span></div><div class="schedule-repeat"><i class="ri-repeat-line" aria-hidden="true"></i><span></span></div></div><p class="power-task-state"></p>`;
                row.querySelector('.schedule-name').textContent = '电源计划';
                const timeText = weekly ? (wake ? spec.wake_time : spec.shutdown_time) : hasTime(target) ? formatDate(target,spec.timezone) : '时间未设置';
                row.querySelector('.schedule-time span').textContent = timeText;
                const days = ['日','一','二','三','四','五','六'];
                row.querySelector('.schedule-repeat span').textContent = weekly ? `每周${(spec.weekdays || []).map(day=>days[day]).join('、')} · ${spec.timezone}${wake && hasShutdown(spec) && spec.wake_time <= spec.shutdown_time ? '（次日开机）' : ''}` : `仅一次 · ${spec.timezone}`;
                row.querySelector('.power-task-state').textContent = enabled && hasTime(target) ? `已启用 · 下次 ${formatDate(target,spec.timezone)}` : (!wake && ['waiting_wake','shutdown_sent'].includes(state.phase)) ? '关机指令已发送' : labels[state.phase] || '状态未知';
                row.querySelector('.power-task-state').hidden = enabled;
                if (state.error) { const error = document.createElement('p'); error.className='power-task-error'; error.textContent=state.error; row.append(error); }
                const edit = row.querySelector('button');
                edit.setAttribute('aria-label',wake ? '编辑定时开机' : '编辑定时关机');
                edit.addEventListener('click',()=>openDialog(kind));
                list.append(row);
            }
        }
        const hasRows = list.querySelector('.schedule-item, [data-power-task]');
        if (hasRows) list.querySelector('.empty-state')?.remove();
        else if (!list.querySelector('.empty-state')) {
            const empty = document.createElement('div'); empty.className='empty-state'; empty.textContent='暂无定时任务，点击右上角添加'; list.append(empty);
        }
    }
    async function request(method, body) {
        const response = await fetch('/api/power-plan',{method,headers:{'Content-Type':'application/json','X-Cat-Led-Request':'1'},body:body ? JSON.stringify(body) : undefined,redirect:'error'});
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || '操作失败，请稍后重试');
        return data;
    }
    async function refresh(fill = false) {
        const version = ++refreshVersion;
        try {
            const result = await request('GET');
            if (version !== refreshVersion || !isAdmin) return;
            loaded = true;
            available = result.available;
            loadError = '';
            $('power-device-error').textContent = result.device_error ? `RTC 开机暂不可用：${result.device_error}` : '';
            $('power-device-error').hidden = !result.device_error;
            renderState(result.state,fill);
            if (fill && result.state.error) showError(result.state.error);
        } catch(error) {
            if (version !== refreshVersion || !isAdmin) return;
            loadError = error.message;
            renderTasks();
            if (fill) showError(error.message);
        }
    }
    function revealDialog() {
        clearTimeout(closeTimer);
        dialog.inert = false;
        if (!dialog.open) {
            dialog.dataset.paper = 'closed';
            dialog.showModal();
            // Establish the clipped starting style before retargeting the transition.
            dialog.getBoundingClientRect();
        }
        dialog.dataset.paper = 'open';
    }
    function finishClose() {
        clearTimeout(closeTimer);
        if (dialog.dataset.paper !== 'closed') return;
        dialog.close();
        dialog.inert = false;
    }
    function hideDialog() {
        if (mutating || !dialog.open) return;
        ++refreshVersion;
        setBusy(false);
        dialog.dataset.paper = 'closed';
        dialog.inert = true;
        if (document.documentElement.dataset.input === 'keyboard') { finishClose(); return; }
        closeTimer = setTimeout(finishClose, reduceMotion.matches ? 180 : 240);
    }
    dialog.addEventListener('transitionend',event=>{
        if (event.target===dialog && event.propertyName === (reduceMotion.matches ? 'opacity' : 'clip-path')) finishClose();
    });
    async function openDialog(kind) {
        if (!isAdmin || mutating) return;
        showError('');
        if (state) renderState(state,true);
        loaded = false;
        $('power-phase').textContent = '正在读取计划…';
        revealDialog();
        $('power-close').focus({preventScroll:true});
        setBusy(true);
        const openingRequest = refreshVersion + 1;
        await refresh(true);
        if (openingRequest !== refreshVersion || dialog.dataset.paper === 'closed') return;
        setBusy(false);
        if (loaded && kind && dialog.open) $(kind === 'off' ? 'power-off-enabled' : 'power-on-enabled').focus();
    }
    document.addEventListener('schedules-rendered',renderTasks);
    document.addEventListener('lazycat-role', async event => {
        isAdmin = event.detail === true;
        $('power-plan-button').hidden = !isAdmin;
        if (!isAdmin) { ++refreshVersion; state=null; loadError=''; clearTimeout(closeTimer); if(dialog.open)dialog.close(); renderTasks(); return; }
        await refresh();
    });
    setInterval(async()=>{
        if (!isAdmin || busy || dialog.open || document.hidden) return;
        await refresh();
    },60000);
    $('power-plan-button').addEventListener('click',()=>openDialog());
    $('power-close').addEventListener('click',hideDialog);
    dialog.addEventListener('cancel',event=>{event.preventDefault();hideDialog();});
    dialog.addEventListener('close',()=>{if(isAdmin)$('power-plan-button').focus({preventScroll:true});});
    $('power-mode').addEventListener('change',changeMode);
    for (const id of ['power-off-enabled','power-on-enabled']) $(id).addEventListener('change',()=>{ updateOptions(); });
    for (const id of ['power-off','power-on']) $(id).addEventListener('input',updateOptions);
    form.addEventListener('submit',async event=>{
        event.preventDefault();
        if(busy || !isAdmin)return;
        showError('');
        const off=offEnabled(),on=onEnabled(),weekly=$('power-mode').value==='weekly';
        const weekdays=[...document.querySelectorAll('#power-weekdays input:checked')].map(input=>Number(input.value));
        if((off||on) && weekly && !weekdays.length){showError('请至少选择一个执行日期。');return;}
        const spec={mode:weekly?'weekly':'once',timezone:$('power-timezone').value.trim(),weekdays,shutdown_enabled:off,wake_enabled:on};
        if(weekly){if(off)spec.shutdown_time=$('power-off').value;if(on)spec.wake_time=$('power-on').value;}
        else {
            if(off)spec.shutdown_at=new Date($('power-off').value).toISOString();
            if(on)spec.wake_at=new Date($('power-on').value).toISOString();
            if([spec.shutdown_at,spec.wake_at].filter(Boolean).some(value=>new Date(value).getTime()<Date.now()+120000)){showError('执行时间至少需在两分钟之后。');return;}
            if(off&&on&&new Date(spec.wake_at)-new Date(spec.shutdown_at)<300000){showError('开机至少比关机晚五分钟。');return;}
        }
        ++refreshVersion;mutating=true;setBusy(true);
        try { renderState(await request('PUT',spec),true);}
        catch(error){showError(error.message);try{await refresh(false);}catch{}}
        finally{mutating=false;setBusy(false);}
    });
    $('power-cancel-plan').addEventListener('click',async()=>{
        if(busy)return;
        showError('');++refreshVersion;mutating=true;setBusy(true);
        try{renderState(await request('DELETE'),true);}
        catch(error){showError(error.message);try{await refresh(false);}catch{}}
        finally{mutating=false;setBusy(false);}
    });
})();
