// ========== УПРАВЛЕНИЕ СКЛАДАМИ ==========

let warehousesData = [];
let editingId = null;

async function loadWarehouses() {
    try {
        const response = await fetch('/api/warehouses');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        warehousesData = await response.json();
        renderTable();
    } catch (error) {
        console.error('Ошибка загрузки складов:', error);
        document.getElementById('warehouses-body').innerHTML = `
            <tr><td colspan="6" style="text-align:center; color:#cc0000; padding:20px;">
                ❌ Ошибка загрузки складов
            </td></tr>
        `;
    }
}

function renderTable() {
    const tbody = document.getElementById('warehouses-body');
    if (warehousesData.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center; padding:20px; color:#888;">Нет складов</td></tr>';
        return;
    }

    tbody.innerHTML = warehousesData.map(w => `
        <tr>
            <td>${w.warehouseId}</td>
            <td><strong>${escapeHtml(w.code)}</strong></td>
            <td>${escapeHtml(w.name)}</td>
            <td>${escapeHtml(w.description || '—')}</td>
            <td>
                <span class="${w.isActive ? 'status-active' : 'status-inactive'}">
                    ${w.isActive ? '✅ Активен' : '❌ Неактивен'}
                </span>
            </td>
            <td>
                <button class="btn-edit" onclick="editWarehouse(${w.warehouseId})">✏️</button>
                <button class="btn-delete" onclick="deleteWarehouse(${w.warehouseId})">🗑</button>
            </td>
        </tr>
    `).join('');
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

function showModal(title, data = null) {
    const modal = document.getElementById('warehouse-modal');
    document.getElementById('modal-title').textContent = title;
    document.getElementById('warehouse-id').value = data?.warehouseId || '';
    document.getElementById('warehouse-code').value = data?.code || '';
    document.getElementById('warehouse-name').value = data?.name || '';
    document.getElementById('warehouse-desc').value = data?.description || '';
    document.getElementById('warehouse-active').checked = data?.isActive !== undefined ? data.isActive : true;
    modal.style.display = 'flex';
    editingId = data?.warehouseId || null;
}

function closeModal() {
    document.getElementById('warehouse-modal').style.display = 'none';
    editingId = null;
}

async function saveWarehouse() {
    console.log('saveWarehouse() called'); // Отладка

    const id = document.getElementById('warehouse-id').value;
    const code = document.getElementById('warehouse-code').value.trim();
    const name = document.getElementById('warehouse-name').value.trim();
    const description = document.getElementById('warehouse-desc').value.trim();
    const isActive = document.getElementById('warehouse-active').checked;

    console.log('Data:', { id, code, name, description, isActive }); // Отладка

    if (!code || !name) {
        alert('Код и название обязательны');
        return;
    }

    try {
        const url = id ? `/api/warehouses/${id}` : '/api/warehouses';
        const method = id ? 'PUT' : 'POST';

        console.log('Sending:', { method, url }); // Отладка

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, name, description: description || null, isActive })
        });

        console.log('Response status:', response.status); // Отладка

        if (response.ok) {
            const result = await response.json();
            console.log('Result:', result); // Отладка
            closeModal();
            await loadWarehouses();
            alert('Склад сохранён!');
        } else {
            const err = await response.json();
            alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Ошибка сохранения:', error);
        alert('Ошибка сохранения склада: ' + error.message);
    }
}

async function editWarehouse(id) {
    try {
        const response = await fetch(`/api/warehouses/${id}`);
        if (!response.ok) throw new Error('Склад не найден');
        const data = await response.json();
        showModal('Редактирование склада', data);
    } catch (error) {
        console.error('Ошибка загрузки:', error);
        alert('Ошибка загрузки данных склада');
    }
}

async function deleteWarehouse(id) {
    if (!confirm('Удалить склад?')) return;

    try {
        const response = await fetch(`/api/warehouses/${id}`, { method: 'DELETE' });
        if (response.ok) {
            await loadWarehouses();
        } else {
            const err = await response.json();
            alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Ошибка удаления:', error);
        alert('Ошибка удаления склада');
    }
}

// ========== ИНИЦИАЛИЗАЦИЯ ==========

document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded'); // Отладка

    loadWarehouses();

    // Кнопка "Добавить склад"
    const addBtn = document.getElementById('add-warehouse-btn');
    if (addBtn) {
        addBtn.addEventListener('click', () => {
            console.log('Add button clicked'); // Отладка
            showModal('Добавление склада');
        });
    } else {
        console.error('Add button not found');
    }

    // Кнопка "Отмена"
    const cancelBtn = document.getElementById('modal-cancel');
    if (cancelBtn) {
        cancelBtn.addEventListener('click', closeModal);
    }

    // Кнопка "✕" в заголовке
    const closeBtn = document.querySelector('.modal-close');
    if (closeBtn) {
        closeBtn.addEventListener('click', closeModal);
    }

    // Кнопка "Сохранить" — ПРОВЕРЬТЕ, ЧТО ОНА ЕСТЬ
    const saveBtn = document.getElementById('modal-save');
    if (saveBtn) {
        console.log('Save button found'); // Отладка
        saveBtn.addEventListener('click', saveWarehouse);
    } else {
        console.error('Save button not found!');
    }

    // Закрытие по клику на фон
    const modal = document.getElementById('warehouse-modal');
    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === e.currentTarget) closeModal();
        });
    }

    // Закрытие по Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeModal();
    });
});