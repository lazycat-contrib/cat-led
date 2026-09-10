(() => {
    const messages = window.CAT_LED_MESSAGES;
    const storageKey = 'cat-led-language';
    const normalize = value => value?.toLowerCase().startsWith('zh') ? 'zh' : 'en';
    let language = normalize(navigator.language);
    try {
        const saved = localStorage.getItem(storageKey);
        if (saved === 'zh' || saved === 'en') language = saved;
    } catch { /* Language switching also works when browser storage is blocked. */ }

    function t(source, params = {}) {
        const pattern = language === 'en' && Object.hasOwn(messages, source) ? messages[source] : source;
        return String(pattern).replace(/\{(\d+)\}/g, (match, key) => Object.hasOwn(params, key) ? String(params[key]) : match);
    }
    function error(value, preserveDetail = false, depth = 0) {
        if (!value) return '';
        const source = String(value);
        if (depth > 4) return source;
        if (language === 'en' && Object.hasOwn(messages,source)) return messages[source];
        if (language === 'zh') {
            const exact = Object.entries(messages).find(([,translated])=>translated===source);
            if (exact) return exact[0];
        }
        // Recover the key and trailing detail of a previously formatted message.
        // Keep unknown detail verbatim; it may be a user value or an OS error.
        for (const [key, translated] of Object.entries(messages)) {
            if (!key.endsWith('{0}') || !translated.endsWith('{0}')) continue;
            for (const pattern of [key,translated]) {
                const prefix = pattern.slice(0,-3);
                if (/(?:：|: )$/.test(prefix) && source.startsWith(prefix)) {
                    return t(key,{'0':error(source.slice(prefix.length),true,depth+1)});
                }
            }
        }
        if (language === 'zh') {
            for (const [key, value] of Object.entries(messages)) {
                if (source === value) return key;
                if (source.startsWith(value+': ')) return key+': '+error(source.slice(value.length+2),true,depth+1);
            }
            return source;
        }
        if (Object.hasOwn(messages, source)) return messages[source];
        const clients = source.match(/^测试通知已发送到(\d+)个客户端(，部分客户端失败)?$/);
        if (clients) return `Test notification sent to ${clients[1]} ${clients[1] === '1' ? 'client' : 'clients'}${clients[2] ? '; some clients failed.' : '.'}`;
        // Translate recognized API prefixes while preserving technical details.
        for (const key of Object.keys(messages).sort((a,b)=>b.length-a.length)) {
            for (const separator of [': ', '：']) {
                if (source.startsWith(key+separator)) return messages[key]+': '+error(source.slice(key.length+separator.length),true,depth+1);
            }
        }
        return !preserveDetail && /[\u4e00-\u9fff]/.test(source) ? t('操作失败，请稍后重试') : source;
    }
    function render(root = document) {
        const nodes = [];
        if (root.nodeType === 1) nodes.push(root);
        if (root.querySelectorAll) nodes.push(...root.querySelectorAll('[data-i18n], [data-i18n-message], [data-i18n-title], [data-i18n-aria-label], [data-i18n-placeholder], [data-language-toggle]'));
        for (const node of nodes) {
            const key = node.getAttribute('data-i18n');
            let params = {};
            try { params = JSON.parse(node.getAttribute('data-i18n-params') || '{}'); } catch {}
            if (key !== null && node.textContent !== t(key,params)) node.textContent = t(key,params);
            const message = node.getAttribute('data-i18n-message');
            if (message !== null && node.textContent !== error(message)) node.textContent = error(message);
            for (const attribute of ['title','aria-label','placeholder']) {
                const source = node.getAttribute('data-i18n-'+attribute);
                if (source !== null) node.setAttribute(attribute,t(source));
            }
            if (node.hasAttribute('data-language-toggle')) {
                const label = node.querySelector('[data-language-label]');
                if (label) label.textContent = language === 'zh' ? 'EN' : '中文';
                node.title = node.ariaLabel = language === 'zh' ? 'Switch to English' : '切换为中文';
                node.lang = language === 'zh' ? 'en' : 'zh-CN';
            }
        }
    }
    function text(node, source, params = {}) {
        node.setAttribute('data-i18n',source);
        node.setAttribute('data-i18n-params',JSON.stringify(params));
        render(node);
    }
    function setLanguage(next, persist = true) {
        if (next !== 'zh' && next !== 'en') return;
        language = next;
        document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
        if (persist) { try { localStorage.setItem(storageKey,language); } catch {} }
        render();
        document.dispatchEvent(new CustomEvent('languagechange',{detail:language}));
    }
    window.I18n = Object.freeze({t,error,render,text,setLanguage,get language(){return language;},locale:()=>language==='zh'?'zh-CN':'en-US'});
    document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
    document.addEventListener('DOMContentLoaded',()=>{
        render();
        document.addEventListener('click',event=>{
            if (event.target.closest('[data-language-toggle]')) setLanguage(language==='zh'?'en':'zh');
        });
        // Only explicit UI message markers are translated. User-created text,
        // form values and notification templates are never scanned or rewritten.
        new MutationObserver(records=>{
            for (const record of records) for (const node of record.addedNodes) if (node.nodeType===1) render(node);
        }).observe(document.body,{childList:true,subtree:true});
    });
    window.addEventListener('storage',event=>{
        if (event.key===storageKey && (event.newValue==='zh'||event.newValue==='en')) setLanguage(event.newValue,false);
    });
})();
