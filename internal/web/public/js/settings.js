(() => {
    const $ = id => document.getElementById(id);
    const dialog = $('settings-dialog');
    let preference = null;
    let scheduleVisibility = null;
    let busy = false;
    let loading = false;
    let scheduleSample = null;
    let powerSample = null;
    let refreshing = false;
    let refreshAgain = false;
    let scheduleVersion = 0;
    const operations = {on:'开灯', off:'关灯', shutdown:'关机', reboot:'重启', wake:'RTC 开机'};

    function message(key) { I18n.text($('settings-status'), key); }
    function sample(events, serverTime) {
        return {events, serverTime:Date.parse(serverTime), receivedAt:performance.now()};
    }
    function renderReminders() {
        const container = $('upcoming-reminders');
        const events = preference?.reminders_enabled
            ? [...Reminders.dueEvents(scheduleSample, preference.reminder_minutes, performance.now()),
               ...Reminders.dueEvents(powerSample, preference.reminder_minutes, performance.now())]
                .sort((a,b) => Date.parse(a.at) - Date.parse(b.at)) : [];
        const lines = events.filter(event => operations[event.operation]).map(event =>
            I18n.t('{0} 分钟后 · {1}{2}', {'0':event.minutes, '1':I18n.t(operations[event.operation]), '2':event.name ? ' · '+event.name : ''}));
        const content = lines.join('\n');
        if (container.dataset.content !== content || container.dataset.language !== I18n.language) {
            container.replaceChildren();
            if (lines.length) {
                const title = document.createElement('strong');
                title.textContent = I18n.t('即将执行'); container.append(title);
                for (const line of lines) { const p=document.createElement('p'); p.textContent=line; container.append(p); }
            }
            container.dataset.content = content;
            container.dataset.language = I18n.language;
        }
        container.hidden = !lines.length;
    }
    function apply(value) {
        preference = value;
        renderScheduleVisibility();
        renderReminders();
    }
    function renderScheduleVisibility() {
        document.querySelector('.schedule-disclosure').hidden = preference?.show_schedules !== false;
        const expanded = preference?.show_schedules !== false || scheduleVisibility === true;
        $('schedule-list-section').hidden = !expanded;
        const toggle = $('schedule-list-toggle');
        toggle.setAttribute('aria-expanded', String(expanded));
        const label = expanded ? '收起定时任务' : '展开定时任务';
        toggle.dataset.i18nAriaLabel = label;
        toggle.dataset.i18nTitle = label;
        I18n.render(toggle);
    }
    $('schedule-list-toggle').addEventListener('click', () => {
        scheduleVisibility = $('schedule-list-section').hidden;
        renderScheduleVisibility();
    });
    function fill() {
        if (!preference) return;
        $('settings-show-schedules').checked = preference.show_schedules;
        $('settings-reminders').checked = preference.reminders_enabled;
        $('settings-minutes').value = preference.reminder_minutes;
        $('settings-minutes').disabled = !preference.reminders_enabled;
    }
    async function load(fillForm = false) {
        if (loading) return;
        loading = true;
        $('settings-fields').disabled = true;
        $('settings-save').disabled = true;
        message('加载中…');
        try {
            const response = await fetch('/api/user/preference', {cache:'no-store'});
            if (!response.ok) throw new Error();
            apply(await response.json());
            if (fillForm || dialog.open) fill();
            message('');
        } catch { message('无法读取设置，请关闭后重试。'); }
        finally {
            loading = false;
            $('settings-fields').disabled = !preference;
            $('settings-save').disabled = !preference;
        }
    }
    async function refresh() {
        if (document.hidden || !preference?.reminders_enabled) return;
        if (refreshing) { refreshAgain = true; return; }
        refreshing = true;
        const version = scheduleVersion;
        try {
            const response = await fetch('/api/upcoming-events', {cache:'no-store', signal:AbortSignal.timeout(10000)});
            if (!response.ok) throw new Error();
            const data = await response.json();
            if (version === scheduleVersion) scheduleSample = sample(data.events, data.server_time);
        } catch { scheduleSample = null; }
        finally {
            refreshing = false; renderReminders();
            if (refreshAgain) { refreshAgain = false; refresh(); }
        }
    }
    $('settings-button').addEventListener('click', event => {
        dialog.dataset.motion = event.detail ? 'pointer' : 'keyboard';
        dialog.showModal();
        $('settings-close').focus();
        load(true);
    });
    $('settings-close').addEventListener('click', event => { if (!busy) { if (!event.detail) dialog.dataset.motion = 'keyboard'; dialog.close(); } });
    dialog.addEventListener('cancel', event => { if (busy) event.preventDefault(); else dialog.dataset.motion = 'keyboard'; });
    dialog.addEventListener('close', () => $('settings-button').focus());
    $('settings-reminders').addEventListener('change', () => {
        $('settings-minutes').disabled = !$('settings-reminders').checked;
    });
    $('settings-form').addEventListener('submit', async event => {
        event.preventDefault();
        if (busy || loading || !preference) return;
        const next = {
            show_schedules:$('settings-show-schedules').checked,
            reminders_enabled:$('settings-reminders').checked,
            reminder_minutes:$('settings-reminders').checked ? Number($('settings-minutes').value) : preference.reminder_minutes
        };
        busy = true;
        $('settings-fields').disabled = true; $('settings-save').disabled = true; $('settings-close').disabled = true;
        message('正在保存…');
        try {
            const response = await fetch('/api/user/preference', {
                method:'PUT', headers:{'Content-Type':'application/json','X-Cat-Led-Request':'1'}, body:JSON.stringify(next)
            });
            if (!response.ok) throw new Error();
            const saved = await response.json();
            scheduleVisibility = null;
            apply(saved); fill(); message('设置已保存'); refresh();
        } catch { message('保存失败，请重试'); }
        finally {
            busy = false;
            $('settings-fields').disabled = false; $('settings-save').disabled = false; $('settings-close').disabled = false;
        }
    });
    document.addEventListener('power-status', event => {
        powerSample = event.detail ? sample(Reminders.powerEvents(event.detail.state), event.detail.server_time) : null;
        renderReminders();
    });
    document.addEventListener('schedules-rendered', () => { ++scheduleVersion; scheduleSample = null; renderReminders(); refresh(); });
    document.addEventListener('languagechange', renderReminders);
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden) { renderReminders(); refresh(); if (!dialog.open && !busy) load(); }
    });
    document.addEventListener('DOMContentLoaded', async () => { await load(); refresh(); });
    setInterval(() => { if (!document.hidden) renderReminders(); }, 1000);
    setInterval(refresh, 15000);
})();
