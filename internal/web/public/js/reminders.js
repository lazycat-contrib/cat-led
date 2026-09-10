(() => {
    // Samples use a monotonic browser clock and a server timestamp, so changing
    // the client clock or timezone cannot move a reminder's execution window.
    function dueEvents(sample, minutes, tick) {
        if (!sample || tick - sample.receivedAt > 90000 || !Number.isFinite(sample.serverTime)) return [];
        const now = sample.serverTime + tick - sample.receivedAt;
        return sample.events.filter(event => {
            const remaining = Date.parse(event.at) - now;
            return remaining > 0 && remaining <= minutes * 60000;
        }).map(event => ({...event, minutes:Math.ceil((Date.parse(event.at) - now) / 60000)}));
    }
    function powerEvents(state) {
        if (!state || state.error) return [];
        const events = [];
        if (state.phase === 'active' && state.spec?.shutdown_enabled !== false)
            events.push({id:'power-off', operation:'shutdown', at:state.shutdown_at});
        if (['active','waiting_wake'].includes(state.phase) && state.spec?.wake_enabled !== false)
            events.push({id:'power-on', operation:'wake', at:state.wake_at});
        return events;
    }
    const api = {dueEvents, powerEvents};
    if (typeof module !== 'undefined') module.exports = api;
    else window.Reminders = Object.freeze(api);
})();
