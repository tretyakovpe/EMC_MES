// ========== РОУТЕР ЛОГИСТИКИ ==========

let currentTab = 'plans';

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

    if (currentTab === 'plans') {
        await PlansModule.render(container);
    } else if (currentTab === 'shipments') {
        // TODO: ShipmentsModule.render(container);
        container.innerHTML = '<div class="logistics-container"><div class="stats-panel"><h3>🚛 Отгрузки</h3><p>В разработке...</p></div></div>';
    } else if (currentTab === 'stats') {
        container.innerHTML = '<div class="logistics-container"><div class="stats-panel"><h3>📊 Статистика</h3><p>В разработке...</p></div></div>';
    }
}

// Вспомогательная функция (нужна для PlansModule)
function escapeHtml(str) {
    if (!str) return '';
    return String(str).replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}