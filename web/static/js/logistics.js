// ========== РОУТЕР ЛОГИСТИКИ ==========

let currentTab = 'shipments';

window.addEventListener('DOMContentLoaded', () => {
    renderTab();
    setupTabHandlers();
});

function setupTabHandlers() {
    document.querySelectorAll('[data-tab]').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('[data-tab]').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            currentTab = tab.dataset.tab;
            renderTab();
        });
    });
}

async function renderTab() {
    const container = document.getElementById('tab-container');
    if (!container) return;

    console.log('renderTab called, currentTab:', currentTab);  // ← ОТЛАДКА

    if (currentTab === 'plans') {
        await PlansModule.render(container);
    } else if (currentTab === 'warehouse') {
        await WarehouseModule.render(container);
    } else if (currentTab === 'shipments') {
        await ShipmentsModule.render(container);
    } else if (currentTab === 'transfers') {
        await TransfersModule.render(container);
    } else if (currentTab === 'stats') {
        container.innerHTML = '<div class="logistics-container"><div class="stats-panel"><h3>📊 Статистика</h3><p>В разработке...</p></div></div>';
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str).replace(/[&<>]/g, function (m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}