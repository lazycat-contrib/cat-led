(() => {
    const $ = id => document.getElementById(id);
    const dialog = $('power-dialog');
    const form = $('power-form');
    let state = null;
    let busy = false;
    let available = false;
    const labels = {disabled:'尚未启用计划', preparing:'正在准备闹钟', active:'计划已启用', waiting_wake:'关机指令已发出，等待开机时间', completed:'单次计划已结束', missed:'已跳过错过的关机时间', error:'计划已停止，请检查后重新保存'};
    const localValue = date => {
        const d = new Date(date);
        return new Date(d.getTime() - d.getTimezoneOffset()*60000).toISOString().slice(0,16);
    };
    function setBusy(value) {
        busy = value;
        form.setAttribute('aria-busy', String(value));
        $('power-fields').disabled = value || !available;
        $('power-save').disabled = value || !available;
        $('power-cancel-plan').disabled = value;
        $('power-close').disabled = value;
        $('power-save').textContent = value ? '正在处理…' : '保存并启用';
    }
    function showError(message) {
        $('power-form-error').textContent = message;
        $('power-form-error').hidden = !message;
        if (message) $('power-form-error').focus();
    }
    function changeMode() {
        const weekly = $('power-mode').value === 'weekly';
        $('power-weekdays').hidden = !weekly;
        $('power-off').type = weekly ? 'time' : 'datetime-local';
        $('power-on').type = weekly ? 'time' : 'datetime-local';
        $('power-off').value = weekly ? '23:00' : localValue(Date.now()+3600000);
        $('power-on').value = weekly ? '07:00' : localValue(Date.now()+9*3600000);
        updateHint();
    }
    function updateHint() {
        const weekly = $('power-mode').value === 'weekly';
        $('power-time-hint').textContent = weekly
            ? ($('power-on').value <= $('power-off').value ? '开机在关机的次日；星期选择对应关机日。' : '关机与开机在同一天，至少间隔五分钟。')
            : '日期使用浏览器时区；关机至少提前两分钟，开机至少晚五分钟。';
    }
    function renderState(next, fill) {
        state = next;
        const label = labels[next.phase] || '状态未知';
        $('power-phase').textContent = label;
        $('power-entry-status').textContent = label;
        const hasTime = ['active','waiting_wake'].includes(next.phase);
        $('power-next').hidden = !hasTime;
        const timezone = next.spec?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
        if (hasTime) {
            const format = value => new Intl.DateTimeFormat('zh-CN',{timeZone:timezone,month:'long',day:'numeric',weekday:'short',hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(value));
            $('power-next-off').textContent = format(next.shutdown_at);
            $('power-next-on').textContent = format(next.wake_at);
        }
        $('power-cancel-plan').hidden = !next.creator || next.phase === 'completed' || (next.phase === 'disabled' && !next.error);
        if (fill && next.spec?.mode) {
            $('power-mode').value = next.spec.mode;
            changeMode();
            $('power-timezone').value = timezone;
            const weekly = next.spec.mode === 'weekly';
            $('power-off').value = weekly ? next.spec.shutdown_time : localValue(next.spec.shutdown_at);
            $('power-on').value = weekly ? next.spec.wake_time : localValue(next.spec.wake_at);
            document.querySelectorAll('#power-weekdays input').forEach(input => { input.checked = (next.spec.weekdays || []).includes(Number(input.value)); });
        }
        updateHint();
        if (next.error) showError(next.error);
    }
    async function request(method, body) {
        const response = await fetch('/api/power-plan', {method, headers:{'Content-Type':'application/json','X-Cat-Led-Request':'1'}, body:body ? JSON.stringify(body) : undefined, redirect:'error'});
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || '操作失败，请稍后重试');
        return data;
    }
    async function refresh(fill = false) {
        const result = await request('GET');
        available = result.available;
        $('power-device-error').textContent = result.device_error ? `设备暂不可用：${result.device_error}` : '';
        $('power-device-error').hidden = !result.device_error;
        renderState(result.state, fill);
    }
    document.addEventListener('lazycat-role', event => {
        $('power-plan-button').hidden = !event.detail;
        if (!event.detail && dialog.open) dialog.close();
    });
    $('power-plan-button').addEventListener('click', async () => {
        showError('');
        $('power-confirm').checked = false;
        available = false;
        $('power-phase').textContent = '正在读取设备…';
        $('power-timezone').value = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai';
        changeMode();
        dialog.showModal();
        $('power-close').focus({preventScroll:true});
        setBusy(true);
        try { await refresh(true); } catch(error) { showError(error.message); }
        finally { setBusy(false); }
    });
    $('power-close').addEventListener('click', () => { if (!busy) dialog.close(); });
    dialog.addEventListener('cancel', event => { if (busy) event.preventDefault(); });
    dialog.addEventListener('close', () => $('power-plan-button').focus({preventScroll:true}));
    $('power-mode').addEventListener('change', changeMode);
    $('power-on').addEventListener('input', updateHint);
    $('power-off').addEventListener('input', updateHint);
    form.addEventListener('submit', async event => {
        event.preventDefault();
        if (busy) return;
        showError('');
        const weekly = $('power-mode').value === 'weekly';
        const weekdays = [...document.querySelectorAll('#power-weekdays input:checked')].map(input => Number(input.value));
        if (weekly && !weekdays.length) { showError('请至少选择一个关机日。'); return; }
        const spec = {mode:weekly ? 'weekly':'once', timezone:$('power-timezone').value.trim(), weekdays};
        if (weekly) { spec.shutdown_time = $('power-off').value; spec.wake_time = $('power-on').value; }
        else {
            spec.shutdown_at = new Date($('power-off').value).toISOString();
            spec.wake_at = new Date($('power-on').value).toISOString();
            if (new Date(spec.shutdown_at).getTime() < Date.now()+120000) { showError('关机时间至少需在两分钟之后。'); return; }
            if (new Date(spec.wake_at)-new Date(spec.shutdown_at) < 300000) { showError('开机时间需比关机时间至少晚五分钟。'); return; }
        }
        setBusy(true);
        try { renderState(await request('PUT',spec),false); $('power-confirm').checked = false; }
        catch(error) { showError(error.message); try { await refresh(false); } catch {} }
        finally { setBusy(false); }
    });
    $('power-cancel-plan').addEventListener('click', async () => {
        if (busy) return;
        showError(''); setBusy(true);
        try { renderState(await request('DELETE'),false); }
        catch(error) { showError(error.message); }
        finally { setBusy(false); }
    });
})();
