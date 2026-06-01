// Глобальное состояние
let currentTab = 'plans';

// Инициализация
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
        await renderPlansTab(container);
    } else if (currentTab === 'shipments') {
        renderShipmentsTab(container);
    } else if (currentTab === 'stats') {
        renderStatsTab(container);
    }
}

// ========== ПЛАНЫ (Excel-like таблица) ==========

async function renderPlansTab(container) {
    // Загружаем данные
    const [plans, materials, lines] = await Promise.all([
        API.getPlans(),
        API.getMaterials(),
        API.getStatisticsLines()
    ]);

    // Группируем планы по датам и материалам
    const plansMap = new Map(); // key: "MaterialID_PlanDate" -> Plan
    plans.forEach(p => {
        const key = `${p.materialId}_${p.planDate}`;
        plansMap.set(key, p);
    });

    // Получаем уникальные даты из планов
    const dates = [...new Set(plans.map(p => p.planDate))].sort();
    
    // Фильтр по дате (выбор месяца)
    const currentMonth = new Date().toISOString().slice(0, 7);
    
    container.innerHTML = `
        <div class="logistics-container">
            <div class="plans-filter-bar">
                <div class="filter-group">
                    <label>📅 Месяц</label>
                    <input type="month" id="month-selector" value="${currentMonth}" class="filter-input">
                </div>
                <div class="filter-group">
                    <label>🔧 Линия</label>
                    <select id="line-filter" class="filter-input">
                        <option value="">Все линии</option>
                        ${lines.map(l => `<option value="${l}">${l}</option>`).join('')}
                    </select>
                </div>
                <button id="apply-plans-filter" class="btn-primary">Применить</button>
                <button id="save-plans" class="btn-success">💾 Сохранить все</button>
            </div>
            <div class="plans-table-wrapper">
                <table class="plans-table" id="plans-table">
                    <thead>
                        <tr>
                            <th class="sticky-col">Материал</th>
                            ${dates.map(d => `<th class="date-col">${formatDate(d)}</th>`).join('')}
                            <th class="total-col">Итого</th>
                        </tr>
                    </thead>
                    <tbody id="plans-table-body">
                        ${materials.map(m => renderPlanRow(m, dates, plansMap)).join('')}
                    </tbody>
                </table>
            </div>
        </div>
    `;

    // Привязываем обработчики
    document.getElementById('apply-plans-filter')?.addEventListener('click', () => {
        // TODO: фильтрация по линии
    });
    
    document.getElementById('save-plans')?.addEventListener('click', saveAllPlans);
    
    // Привязываем обработчики изменения ячеек
    attachCellChangeHandlers();
}

function renderPlanRow(material, dates, plansMap) {
    let total = 0;
    const cells = dates.map(date => {
        const key = `${material.materialId}_${date}`;
        const plan = plansMap.get(key);
        const amount = plan ? plan.plannedAmount : 0;
        total += amount;
        return `<td class="editable-cell" data-material-id="${material.materialId}" data-date="${date}">
                    <input type="number" class="plan-input" value="${amount}" step="10">
                </td>`;
    }).join('');
    
    return `
        <tr>
            <td class="material-cell" data-material-id="${material.materialId}">
                ${escapeHtml(material.materialCode)}
            </td>
            ${cells}
            <td class="total-cell">${total}</td>
        </tr>
    `;
}

function attachCellChangeHandlers() {
    document.querySelectorAll('.plan-input').forEach(input => {
        input.removeEventListener('change', onPlanChange);
        input.addEventListener('change', onPlanChange);
    });
}

function onPlanChange(e) {
    const td = e.target.closest('td');
    const materialId = td.dataset.materialId;
    const date = td.dataset.date;
    const newValue = parseInt(e.target.value) || 0;
    
    // Обновляем итого по строке
    const row = td.closest('tr');
    const totalCell = row.querySelector('.total-cell');
    let total = 0;
    row.querySelectorAll('.plan-input').forEach(inp => {
        total += parseInt(inp.value) || 0;
    });
    totalCell.textContent = total;
}

async function saveAllPlans() {
    const plansToSave = [];
    
    document.querySelectorAll('#plans-table-body tr').forEach(row => {
        const materialId = row.querySelector('.material-cell')?.dataset.materialId;
        const inputs = row.querySelectorAll('.plan-input');
        
        inputs.forEach(input => {
            const td = input.closest('td');
            const date = td.dataset.date;
            const amount = parseInt(input.value) || 0;
            
            if (amount > 0) {
                plansToSave.push({
                    planDate: date,
                    shift: null,
                    materialId: parseInt(materialId),
                    plannedAmount: amount
                });
            }
        });
    });
    
    // Сохраняем каждый план (в реальности лучше одним запросом)
    for (const plan of plansToSave) {
        try {
            await API.createPlan(plan);
        } catch (error) {
            console.error('Ошибка сохранения плана:', plan, error);
        }
    }
    
    alert('Планы сохранены');
    renderPlansTab(document.getElementById('tab-container'));
}

// ========== ОТГРУЗКИ ==========

function renderShipmentsTab(container) {
    container.innerHTML = `
        <div class="logistics-container">
            <div class="shipments-panel">
                <div class="shipments-header">
                    <h3>🚛 Управление отгрузками</h3>
                    <button id="create-shipment-btn" class="btn-primary">➕ Создать отгрузку</button>
                </div>
                <div class="shipments-list" id="shipments-list">
                    <div class="loading">Загрузка...</div>
                </div>
            </div>
        </div>
    `;
    
    loadShipments();
    
    document.getElementById('create-shipment-btn')?.addEventListener('click', () => {
        alert('Создание отгрузки (в разработке)');
    });
}

async function loadShipments() {
    try {
        const shipments = await API.getShipments();
        const container = document.getElementById('shipments-list');
        
        if (!shipments.length) {
            container.innerHTML = '<div class="empty-state">Нет отгрузок</div>';
            return;
        }
        
        container.innerHTML = `
            <table class="shipments-table">
                <thead>
                    <tr>
                        <th>№</th>
                        <th>Дата</th>
                        <th>Статус</th>
                        <th>Прогресс</th>
                        <th>Действия</th>
                    </tr>
                </thead>
                <tbody>
                    ${shipments.map(s => `
                        <tr>
                            <td>${s.number || s.shipmentId}</td>
                            <td>${s.date}</td>
                            <td>${s.done ? '✅ Отгружена' : (s.completed ? '📦 Готова' : '🔄 В работе')}</td>
                            <td><div class="progress-bar"><div class="progress-fill" style="width: ${s.progress}%"></div></div></td>
                            <td><button class="btn-sm" onclick="viewShipment(${s.shipmentId})">Детали</button></td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        console.error('Ошибка загрузки отгрузок:', error);
    }
}

// ========== СТАТИСТИКА ==========

function renderStatsTab(container) {
    container.innerHTML = `
        <div class="logistics-container">
            <div class="stats-panel">
                <h3>📊 Логистическая статистика</h3>
                <div class="stats-placeholder">
                    <p>Статистика по отгрузкам и планам</p>
                    <p><em>В разработке...</em></p>
                </div>
            </div>
        </div>
    `;
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========

function formatDate(dateStr) {
    const parts = dateStr.split('-');
    return `${parts[2]}.${parts[1]}`;
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

// Экспортируем для кнопок
window.viewShipment = async (id) => {
    alert(`Просмотр отгрузки ${id} (в разработке)`);
};