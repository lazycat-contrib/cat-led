(() => {
    const button = document.getElementById('about-button');
    const dialog = document.getElementById('about-dialog');
    const cat = document.getElementById('about-cat');
    const message = document.getElementById('about-cat-message');
    const uptime = document.getElementById('about-uptime');
    let uptimeRequest;
    let refreshTimer;

    function renderUptime(seconds) {
        const minutes = Math.floor(seconds / 60);
        const days = Math.floor(minutes / 1440);
        const hours = Math.floor(minutes / 60) % 24;
        if (days) I18n.text(uptime, '{0}天{1}小时{2}分', {0: days, 1: hours, 2: minutes % 60});
        else if (hours) I18n.text(uptime, '{0}小时{1}分', {0: hours, 1: minutes % 60});
        else if (minutes) I18n.text(uptime, '{0}分', {0: minutes});
        else I18n.text(uptime, '不足1分钟');
    }

    async function loadUptime() {
        clearTimeout(refreshTimer);
        uptimeRequest?.abort();
        const request = new AbortController();
        uptimeRequest = request;
        const timeout = setTimeout(() => request.abort(), 5000);
        try {
            const response = await fetch('/api/system/uptime', {signal: request.signal, cache: 'no-store'});
            if (!response.ok) throw new Error('Uptime unavailable');
            const data = await response.json();
            if (!Number.isSafeInteger(data.uptime_seconds) || data.uptime_seconds < 0) throw new Error('Invalid uptime');
            if (uptimeRequest !== request || !dialog.open) return;
            uptime.dataset.state = 'ready';
            renderUptime(data.uptime_seconds);
        } catch {
            if (uptimeRequest !== request || !dialog.open) return;
            uptime.dataset.state = 'unavailable';
            I18n.text(uptime, '暂时无法读取');
        } finally {
            clearTimeout(timeout);
            if (uptimeRequest === request && dialog.open) refreshTimer = setTimeout(loadUptime, 60000);
        }
    }

    button.addEventListener('click', event => {
        cat.setAttribute('aria-pressed', 'false');
        I18n.text(message, '点点小猫，一起玩毛线球。');
        dialog.dataset.motion = event.detail ? 'pointer' : 'keyboard';
        dialog.showModal();
        uptime.dataset.state = 'loading';
        I18n.text(uptime, '加载中…');
        loadUptime();
    });
    document.getElementById('about-close').addEventListener('click', event => {
        if (!event.detail) dialog.dataset.motion = 'keyboard';
        dialog.close();
    });
    dialog.addEventListener('cancel', () => { dialog.dataset.motion = 'keyboard'; });
    dialog.addEventListener('click', event => {
        if (event.target !== dialog) return;
        const rect = dialog.getBoundingClientRect();
        if (event.clientX < rect.left || event.clientX > rect.right ||
            event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
    });
    dialog.addEventListener('close', () => {
        clearTimeout(refreshTimer);
        uptimeRequest?.abort();
        uptimeRequest = null;
        button.focus({preventScroll:true});
    });

    cat.addEventListener('click', () => {
        const playing = cat.getAttribute('aria-pressed') !== 'true';
        cat.setAttribute('aria-pressed', String(playing));
        I18n.text(message, playing ? '喵，再点一下，陪你静静待着。' : '点点小猫，一起玩毛线球。');
    });
})();
