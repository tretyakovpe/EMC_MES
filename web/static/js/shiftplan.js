// Получаем ID линии из URL
function getLineId() {
    const path = window.location.pathname;
    const parts = path.split('/');
    return parts[parts.length - 1];
}

// Загрузка данных
async function loadShiftPlan() {
    const lineId = getLineId();
    document.getElementById('line-name').textContent = lineId;
    document.getElementById('date-info').textContent = formatDate(new Date());
    
    try {
        // Получаем план на текущую смену
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

function renderShiftPlan(data) {
    const container = document.getElementById('shiftplan-content');
    
    if (!data.materials || data.materials.length === 0) {
        container.innerHTML = '<div class="empty">📭 Нет планов на текущую смену</div>';
        return;
    }
    
    container.innerHTML = data.materials.map(material => {
        const totalBoxes = material.plannedBoxes;
        const completedBoxes = material.completedBoxes;
        const progress = totalBoxes > 0 ? (completedBoxes / totalBoxes * 100) : 0;
        
        // Генерируем коробки
        const boxesHtml = [];
        for (let i = 1; i <= totalBoxes; i++) {
            const isCompleted = i <= completedBoxes;
            boxesHtml.push(`
                <div class="box-card ${isCompleted ? 'completed' : 'planned'}">
                    <div class="box-number">${i}</div>
                    <div class="box-status">${isCompleted ? '✓' : '📋'}</div>
                </div>
            `);
        }
        
        return `
            <div class="material-section">
                <div class="material-title">${escapeHtml(material.materialCode)}</div>
                <div class="boxes-container">
                    ${boxesHtml.join('')}
                </div>
                <div class="material-progress">
                    <div class="progress-text">${completedBoxes} / ${totalBoxes} коробок</div>
                    <div class="progress-bar-container">
                        <div class="progress-bar-fill" style="width: ${progress}%"></div>
                    </div>
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

// Запуск
window.addEventListener('DOMContentLoaded', loadShiftPlan);