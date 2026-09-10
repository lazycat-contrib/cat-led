(() => {
    const button = document.getElementById('about-button');
    const dialog = document.getElementById('about-dialog');
    const cat = document.getElementById('about-cat');
    const message = document.getElementById('about-cat-message');

    button.addEventListener('click', event => {
        dialog.dataset.motion = event.detail ? 'pointer' : 'keyboard';
        dialog.showModal();
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
    dialog.addEventListener('close', () => button.focus({preventScroll:true}));

    cat.addEventListener('click', event => {
        cat.dataset.motion = event.detail ? 'pointer' : 'keyboard';
        const awake = cat.getAttribute('aria-pressed') !== 'true';
        cat.setAttribute('aria-pressed', String(awake));
        I18n.text(message, awake ? '喵，把今晚的小星星送给你。' : '嘘，小猫正在打盹。');
    });
})();
