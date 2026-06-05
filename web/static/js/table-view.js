// Парсим параметры URL
function getQueryParams() {
    const params = new URLSearchParams(window.location.search);
    return {
        view: params.get('view'),           // 'boxes-by-material'
        materialCode: params.get('material'),
        title: params.get('title')
    };
}

// Определяем настройки для разных типов таблиц
const tableConfigs = {
    'boxes-by-material': {
        title: 'Коробки по материалу',
        apiUrl: (params) => `/api/boxes?materialCode=${encodeURIComponent(params.materialCode)}&status=Произведена`,
        columns: [
            { key: 'huNumber', title: 'Номер бирки' },
            { key: 'productionDate', title: 'Дата производства' },
            { key: 'amount', title: 'Количество, шт' }
        ],
        mapRow: (item) => ({
            huNumber: item.huNumber || '—',
            productionDate: item.productionDate || formatDate(item.createdAt),
            amount: item.amount
        })
    }
};

// Форматирование даты
function formatDate(dateStr) {
    if (!dateStr) return '—';
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Загрузка данных
async function loadTableData() {
    const params = getQueryParams();
    const config = tableConfigs[params.view];
    
    if (!config) {
        document.getElementById('table-content').innerHTML = '<div class="empty">❌ Неизвестный тип таблицы</div>';
        return;
    }
    
    // Устанавливаем заголовок
    const title = params.title || config.title;
    document.getElementById('page-title').textContent = title;
    
    try {
        const url = config.apiUrl(params);
        const response = await fetch(url);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        
        renderTable(config, data);
    } catch (error) {
        console.error('Ошибка загрузки:', error);
        document.getElementById('table-content').innerHTML = '<div class="empty">❌ Ошибка загрузки данных</div>';
    }
}

// Отрисовка таблицы
function renderTable(config, data) {
    const container = document.getElementById('table-content');
    
    if (!data || data.length === 0) {
        container.innerHTML = '<div class="empty">📭 Нет данных</div>';
        return;
    }
    
    // Создаём таблицу
    let html = '<table class="data-table">';
    
    // Заголовки
    html += '<thead><tr>';
    for (const col of config.columns) {
        html += `<th>${col.title}</th>`;
    }
    html += '</tr></thead>';
    
    // Строки
    html += '<tbody>';
    data.forEach((item, index) => {
        const row = config.mapRow(item);
        const rowClass = index % 2 === 0 ? 'even' : '';
        html += `<tr class="${rowClass}">`;
        for (const col of config.columns) {
            let value = row[col.key];
            if (value === undefined || value === null) value = '—';
            html += `<td>${value}</td>`;
        }
        html += '</tr>';
    });
    html += '</tbody></table>';
    
    container.innerHTML = html;
}

// Запуск
window.addEventListener('DOMContentLoaded', loadTableData);