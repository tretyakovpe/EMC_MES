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

function getStatusText(s) {
    if (s.done) return '✅ Отгружена';
    if (s.completed) return '📦 Готова';
    return '🔄 В работе';
}

// ========== ОТГРУЗКИ ==========

let currentShipment = null;
let shipmentItems = [];
let currentShipmentNumber = null;

async function renderShipmentsTab(container) {
    container.innerHTML = `
        <div class="logistics-container">
            <div class="shipments-layout">
                <!-- Левая панель: список отгрузок -->
                <div class="shipments-list-panel">
                    <div class="panel-header">
                        <h3>🚛 Отгрузки за сегодня</h3>
                        <button id="create-shipment-btn" class="btn-primary">➕ Новая отгрузка</button>
                    </div>
                    <div class="shipments-grid" id="shipments-list">
                        <div class="loading">Загрузка...</div>
                    </div>
                </div>
                
                <!-- Правая панель: создание/редактирование отгрузки -->
                <div class="shipment-editor-panel" id="shipment-editor" style="display: none;">
                    <div class="panel-header">
                        <h3>📦 Сборка отгрузки</h3>
                        <button id="close-editor-btn" class="btn-close">✕</button>
                    </div>
                    <div class="shipment-number-input">
                        <label>Номер накладной:</label>
                        <input type="text" id="shipment-number" placeholder="Введите номер накладной" class="form-input">
                    </div>
                    <div class="shipment-items-list" id="shipment-items-list">
                        <div class="empty-items">Добавьте материалы</div>
                    </div>
                    <div class="shipment-actions">
                        <button id="save-shipment-btn" class="btn-success">💾 Сохранить отгрузку</button>
                    </div>
                </div>
                
                <!-- Панель выбора материалов -->
                <div id="materials-picker" class="materials-picker" style="display: none;">
                    <div class="picker-content">
                        <div class="picker-header">
                            <h4>Выберите материал</h4>
                            <button class="close-picker-btn">✕</button>
                        </div>
                        <div class="materials-list" id="materials-list"></div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    await loadTodayShipments();
    
    document.getElementById('create-shipment-btn')?.addEventListener('click', () => {
        currentShipmentNumber = null;
        document.getElementById('shipment-number').value = '';
        showMaterialsPicker();
    });
    document.getElementById('close-editor-btn')?.addEventListener('click', closeShipmentEditor);
    document.getElementById('save-shipment-btn')?.addEventListener('click', saveShipment);
}

async function loadTodayShipments() {
    const today = new Date().toISOString().slice(0, 10);
    try {
        const shipments = await API.getShipments({ fromDate: today, toDate: today });
        const container = document.getElementById('shipments-list');
        
        if (!shipments.length) {
            container.innerHTML = '<div class="empty-state">Нет отгрузок за сегодня</div>';
            return;
        }
        
        container.innerHTML = shipments.map(s => `
            <div class="shipment-card ${s.done ? 'done' : (s.completed ? 'completed' : 'active')}" 
                 data-id="${s.shipmentId}"
                 onclick="viewShipmentDetails(${s.shipmentId})">
                <div class="shipment-header">
                    <span class="shipment-number">№${s.number || s.shipmentId}</span>
                    <span class="shipment-status">${getStatusText(s)}</span>
                </div>
                <div class="shipment-date">${s.date}</div>
                <div class="shipment-progress">
                    <div class="progress-bar"><div class="progress-fill" style="width: ${s.progress}%"></div></div>
                    <span>${s.progress}%</span>
                </div>
                ${!s.done ? `<button class="delete-shipment-btn" onclick="event.stopPropagation(); deleteShipment(${s.shipmentId})">🗑</button>` : ''}
            </div>
        `).join('');
    } catch (error) {
        console.error('Ошибка загрузки отгрузок:', error);
    }
}

async function viewShipmentDetails(shipmentId) {
    try {
        const shipment = await API.getShipmentById(shipmentId);
        
        // Формируем детали для отображения
        let detailsHtml = '';
        if (shipment.details && shipment.details.length) {
            detailsHtml = '<div class="details-list">';
            for (const d of shipment.details) {
                detailsHtml += `
                    <div class="detail-item">
                        <span class="detail-material">${d.materialCode}</span>
                        <span class="detail-boxes">📦 ${d.scannedBoxes}/${d.boxes} кор.</span>
                        <span class="detail-amount">📊 ${d.amount} шт.</span>
                    </div>
                `;
            }
            detailsHtml += '</div>';
        } else {
            detailsHtml = '<div class="no-details">Нет позиций</div>';
        }
        
        alert(`Отгрузка №${shipment.number || shipment.shipmentId}\nДата: ${shipment.date}\nСтатус: ${shipment.completed ? 'Готова к отгрузке' : (shipment.done ? 'Отгружена' : 'В работе')}\nПрогресс: ${shipment.progress}%\n\nПозиции:\n${shipment.details?.map(d => `${d.materialCode}: ${d.scannedBoxes}/${d.boxes} кор. (${d.amount} шт.)`).join('\n') || 'нет'}`);
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка загрузки деталей отгрузки');
    }
}

async function showMaterialsPicker() {
    // Получаем доступные коробки (со статусом "Произведена")
    const boxes = await API.getBoxes({ status: 'Произведена', limit: 100 });
    
    // Группируем по материалам
    const materialsMap = new Map();
    for (const box of boxes) {
        if (!materialsMap.has(box.materialCode)) {
            materialsMap.set(box.materialCode, {
                materialId: box.materialId,
                materialCode: box.materialCode,
                availableBoxes: 0,
                availableAmount: 0
            });
        }
        const m = materialsMap.get(box.materialCode);
        m.availableBoxes++;
        m.availableAmount += box.amount;
    }
    
    const materials = Array.from(materialsMap.values());
    
    const picker = document.getElementById('materials-picker');
    const materialsList = document.getElementById('materials-list');
    
    if (materials.length === 0) {
        materialsList.innerHTML = '<div class="empty-state">Нет доступных коробок для отгрузки</div>';
    } else {
        materialsList.innerHTML = materials.map(m => `
            <div class="material-item" data-material-id="${m.materialId}" data-material-code="${m.materialCode}" data-available="${m.availableBoxes}" data-amount="${m.availableAmount}">
                <div class="material-info">
                    <strong>${m.materialCode}</strong>
                    <span>📦 ${m.availableBoxes} коробок</span>
                    <span>📊 ${m.availableAmount} шт.</span>
                </div>
                <button class="add-material-btn">➕ Добавить</button>
            </div>
        `).join('');
    }
    
    picker.style.display = 'flex';
    
    materialsList.querySelectorAll('.add-material-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const item = btn.closest('.material-item');
            const materialId = parseInt(item.dataset.materialId);
            const materialCode = item.dataset.materialCode;
            const maxBoxes = parseInt(item.dataset.available);
            
            const count = prompt(`Сколько коробок ${materialCode} добавить? (доступно: ${maxBoxes})`, '1');
            if (count && parseInt(count) > 0) {
                addToShipment(materialId, materialCode, parseInt(count));
            }
        });
    });
    
    document.querySelector('.close-picker-btn').onclick = () => {
        picker.style.display = 'none';
    };
}

function addToShipment(materialId, materialCode, boxesCount) {
    const existing = shipmentItems.find(i => i.materialId === materialId);
    if (existing) {
        existing.boxes += boxesCount;
    } else {
        shipmentItems.push({
            materialId: materialId,
            materialCode: materialCode,
            boxes: boxesCount
        });
    }
    renderShipmentItems();
    
    document.getElementById('materials-picker').style.display = 'none';
}

function renderShipmentItems() {
    const container = document.getElementById('shipment-items-list');
    const editor = document.getElementById('shipment-editor');
    
    if (shipmentItems.length === 0) {
        container.innerHTML = '<div class="empty-items">Добавьте материалы</div>';
        editor.style.display = 'none';
        return;
    }
    
    editor.style.display = 'flex';
    
    container.innerHTML = shipmentItems.map((item, idx) => `
        <div class="shipment-item-card">
            <div class="item-info">
                <strong>${item.materialCode}</strong>
                <span>📦 ${item.boxes} кор.</span>
            </div>
            <div class="item-actions">
                <button class="item-delete" onclick="removeShipmentItem(${idx})">🗑</button>
            </div>
        </div>
    `).join('');
}

function removeShipmentItem(index) {
    shipmentItems.splice(index, 1);
    renderShipmentItems();
}

async function saveShipment() {
    if (shipmentItems.length === 0) {
        alert('Добавьте хотя бы один материал');
        return;
    }
    
    const shipmentNumber = document.getElementById('shipment-number')?.value.trim();
    if (!shipmentNumber) {
        alert('Введите номер накладной');
        return;
    }
    
    // Получаем количество деталей в коробке для каждого материала
    const details = [];
    for (const item of shipmentItems) {
        // Получаем количество деталей в коробке из материала
        const material = await API.getMaterialByCode(item.materialCode);
        const amountPerBox = material?.quantityInHU || 50;
        
        details.push({
            materialId: item.materialId,
            boxes: item.boxes,
            amount: item.boxes * amountPerBox
        });
    }
    
    try {
        const result = await API.createShipment({
            number: parseInt(shipmentNumber),
            date: new Date().toISOString().slice(0, 10),
            details: details
        });
        
        alert(`Отгрузка №${shipmentNumber} создана`);
        
        // Очищаем форму
        shipmentItems = [];
        currentShipmentNumber = null;
        renderShipmentItems();
        
        // Обновляем список отгрузок
        await loadTodayShipments();
        
    } catch (error) {
        console.error('Ошибка создания отгрузки:', error);
        alert('Ошибка создания отгрузки: ' + error.message);
    }
}

function closeShipmentEditor() {
    shipmentItems = [];
    currentShipmentNumber = null;
    renderShipmentItems();
}

async function deleteShipment(shipmentId) {
    if (!confirm('Удалить отгрузку?')) return;
    try {
        await API.deleteShipment(shipmentId);
        await loadTodayShipments();
    } catch (error) {
        console.error('Ошибка удаления:', error);
        alert('Ошибка удаления: ' + error.message);
    }
}

// Глобальные функции для onclick
window.viewShipmentDetails = viewShipmentDetails;
window.deleteShipment = deleteShipment;
window.removeShipmentItem = removeShipmentItem;

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
