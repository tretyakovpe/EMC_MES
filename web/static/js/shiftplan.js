// Получаем ID линии из URL
function getLineId() {
    const path = window.location.pathname;
    const parts = path.split('/');
    return parts[parts.length - 1];
}

function renderShiftPlan(data) {
    const container = document.getElementById('shiftplan-content');
    
    if (!data.materials || data.materials.length === 0) {
        container.innerHTML = '<div class="empty">📭 Нет планов на текущую смену</div>';
        return;
    }
    
    container.innerHTML = data.materials.map(material => {
        const planned = material.plannedAmount;
        const actual = material.actualAmount;
        const progress = planned > 0 ? (actual / planned * 100) : 0;
        
        // Проверяем статус: "Незапланирована" → жёлтый фон
        const isUnplanned = material.status === 'Незапланирована';
        const sectionClass = isUnplanned ? 'material-section unplanned' : 'material-section';
        
        return `
            <div class="${sectionClass}">
                <div class="material-title">${escapeHtml(material.materialCode)}</div>
                <div class="material-stats">
                    <span class="planned">📋 План: <span>${planned}</span> шт.</span>
                    <span class="actual">✅ Факт: ${actual} шт.</span>
                    <span class="percent">${Math.round(progress)}%</span>
                </div>
                <div class="progress-bar-container">
                    <div class="progress-bar-fill" style="width: ${progress}%"></div>
                </div>
            </div>
        `;
    }).join('');
}

function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${day}.${month}.${year} ${hours}:${minutes}`;
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str).replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}

let ws = null;

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
    
    ws.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            
            // Обновляем сменное задание при изменении плана
            if (message.type === 'plan_updated') {
                console.log('Plan updated, refreshing shiftplan...');
                loadShiftPlan();
            }
        } catch (error) {
            console.error('WebSocket parse error:', error);
        }
    };
    
    ws.onclose = () => {
        console.log('WebSocket disconnected, reconnecting in 5s...');
        setTimeout(initWebSocket, 5000);
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

// В loadShiftPlan добавить обновление времени:
async function loadShiftPlan() {
    const lineId = getLineId();
    document.getElementById('line-name').textContent = lineId;
    document.getElementById('date-info').innerHTML = `${formatDate(new Date())} <span style="font-size:11px; opacity:0.7;">(обновлено)</span>`;
    
    try {
        const response = await fetch(`/api/shiftplan/${lineId}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        renderShiftPlan(data);
    } catch (error) {
        console.error('Ошибка:', error);
        document.getElementById('shiftplan-content').innerHTML = 
            '<div class="error">❌ Ошибка загрузки данных</div>';
    }
}

// Запуск WebSocket при загрузке страницы
window.addEventListener('DOMContentLoaded', () => {
    loadShiftPlan();
    initWebSocket();
});