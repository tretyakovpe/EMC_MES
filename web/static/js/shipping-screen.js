let countdown = 60;
let updateInterval = null;
let countdownInterval = null;

async function loadData() {
    try {
        const response = await fetch('/api/shipping-screen');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        renderTable(data);
        document.getElementById('updateTime').textContent = new Date().toLocaleTimeString('ru-RU');
        countdown = 60;
    } catch (error) {
        console.error('Ошибка загрузки:', error);
        document.getElementById('tableBody').innerHTML = '<tr><td colspan="7" class="empty-row">❌ Ошибка загрузки данных</td></tr>';
    }
}

function renderTable(data) {
    const tbody = document.getElementById('tableBody');
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty-row">📭 Нет планируемых отгрузок</td></tr>';
        return;
    }
    tbody.innerHTML = data.map(row => {
        const today = row.today || 0;
        const tomorrow = row.tomorrow || 0;
        const dayAfter = row.dayAfter || 0;
        return `
            <tr>
                <td>${escapeHtml(row.customerCode)}</td>
                <td>${escapeHtml(row.materialCode)}</td>
                <td>${row.boxAmount}</td>
                <td>${escapeHtml(row.destination)}</td>
                <td class="today">${today}</td>
                <td class="tomorrow">${tomorrow}</td>
                <td class="dayafter">${dayAfter}</td>
            </tr>
        `;
    }).join('');
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

function updateClock() {
    document.getElementById('currentTime').textContent = new Date().toLocaleTimeString('ru-RU');
}

function startAutoUpdate() {
    loadData();
    updateClock();

    // Обновление данных каждые 60 секунд
    updateInterval = setInterval(loadData, 60000);

    // Таймер обратного отсчёта
    countdownInterval = setInterval(() => {
        countdown--;
        if (countdown <= 0) countdown = 60;
        document.getElementById('countdown').textContent = countdown;
    }, 1000);

    // Обновление часов каждую секунду
    setInterval(updateClock, 1000);
}

// Запуск при загрузке страницы
window.addEventListener('DOMContentLoaded', startAutoUpdate);