// ========== УПРАВЛЕНИЕ МАТЕРИАЛАМИ ==========

let materialsData = [];
let editingId = null;

async function loadMaterials() {
    try {
        const response = await fetch('/api/materials');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        materialsData = await response.json();
        renderTable();
    } catch (error) {
        console.error('Ошибка загрузки материалов:', error);
        document.getElementById('materials-body').innerHTML = `
            <tr><td colspan="9" style="text-align:center; color:#cc0000; padding:20px;">
                ❌ Ошибка загрузки материалов
            </td></tr>
        `;
    }
}

function renderTable() {
    const tbody = document.getElementById('materials-body');
    if (materialsData.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" style="text-align:center; padding:20px; color:#888;">Нет материалов</td></tr>';
        return;
    }

    tbody.innerHTML = materialsData.map(m => `
        <tr>
            <td class="code-cell">${escapeHtml(m.materialCode)}</td>
            <td>${escapeHtml(m.description || '—')}</td>
            <td>${escapeHtml(m.customerCode || '—')}</td>
            <td>${escapeHtml(m.destination || '—')}</td>
            <td>${escapeHtml(m.hu || '—')}</td>
            <td>${m.netto}</td>
            <td>${m.brutto}</td>
            <td>${m.quantityInHU}</td>
            <td>
                <button class="btn-edit" onclick="editMaterial(${m.materialId})">✏️</button>
                <button class="btn-delete" onclick="deleteMaterial(${m.materialId})">🗑</button>
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
    const modal = document.getElementById('material-modal');
    document.getElementById('modal-title').textContent = title;
    document.getElementById('material-id').value = data?.materialId || '';
    document.getElementById('material-code').value = data?.materialCode || '';
    document.getElementById('material-desc').value = data?.description || '';
    document.getElementById('material-customer').value = data?.customerCode || '';
    document.getElementById('material-dest').value = data?.destination || '';
    document.getElementById('material-hu').value = data?.hu || '';
    document.getElementById('material-netto').value = data?.netto || '';
    document.getElementById('material-brutto').value = data?.brutto || '';
    document.getElementById('material-qty').value = data?.quantityInHU || '';
    modal.style.display = 'flex';
    editingId = data?.materialId || null;
}

function closeModal() {
    document.getElementById('material-modal').style.display = 'none';
    editingId = null;
}

async function saveMaterial() {
    console.log('saveMaterial() called');

    const id = document.getElementById('material-id').value;
    const materialCode = document.getElementById('material-code').value.trim();
    const description = document.getElementById('material-desc').value.trim();
    const customerCode = document.getElementById('material-customer').value.trim();
    const destination = document.getElementById('material-dest').value.trim();
    const hu = document.getElementById('material-hu').value.trim();
    const netto = parseInt(document.getElementById('material-netto').value) || 0;
    const brutto = parseInt(document.getElementById('material-brutto').value) || 0;
    const quantityInHU = parseInt(document.getElementById('material-qty').value) || 0;

    console.log('Data:', { id, materialCode, description, customerCode, destination, hu, netto, brutto, quantityInHU });

    if (!materialCode) {
        alert('⚠️ Код материала обязателен');
        document.getElementById('material-code').style.borderColor = '#cc0000';
        return;
    }
    if (!customerCode) {
        alert('⚠️ Артикул обязателен');
        document.getElementById('material-customer').style.borderColor = '#cc0000';
        return;
    }
    if (!hu) {
        alert('⚠️ HU обязателен');
        document.getElementById('material-hu').style.borderColor = '#cc0000';
        return;
    }

    try {
        const url = id ? `/api/materials/${id}` : '/api/materials';
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                materialCode,
                customerCode,
                destination: destination || null,
                hu,
                netto,
                brutto,
                quantityInHU,
                description
            })
        });

        if (response.ok) {
            closeModal();
            await loadMaterials();
            alert('Материал сохранён!');
        } else {
            const err = await response.json();
            alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Ошибка сохранения:', error);
        alert('Ошибка сохранения материала: ' + error.message);
    }
}

async function editMaterial(id) {
    try {
        const response = await fetch(`/api/materials/${id}`);
        if (!response.ok) throw new Error('Материал не найден');
        const data = await response.json();
        showModal('Редактирование материала', data);
    } catch (error) {
        console.error('Ошибка загрузки:', error);
        alert('Ошибка загрузки данных материала');
    }
}

async function deleteMaterial(id) {
    if (!confirm('Удалить материал?')) return;

    try {
        const response = await fetch(`/api/materials/${id}`, { method: 'DELETE' });
        if (response.ok) {
            await loadMaterials();
        } else {
            const err = await response.json();
            alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Ошибка удаления:', error);
        alert('Ошибка удаления материала');
    }
}

// Сброс подсветки полей при вводе
document.addEventListener('input', function (e) {
    if (e.target.classList.contains('form-control')) {
        e.target.style.borderColor = '#999';
    }
});

// ========== ИНИЦИАЛИЗАЦИЯ ==========

document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded');

    loadMaterials();

    const addBtn = document.getElementById('add-material-btn');
    if (addBtn) {
        addBtn.addEventListener('click', () => {
            showModal('Добавление материала');
        });
    }

    const cancelBtn = document.getElementById('modal-cancel');
    if (cancelBtn) {
        cancelBtn.addEventListener('click', closeModal);
    }

    const closeBtn = document.querySelector('.modal-close');
    if (closeBtn) {
        closeBtn.addEventListener('click', closeModal);
    }

    const saveBtn = document.getElementById('modal-save');
    if (saveBtn) {
        saveBtn.addEventListener('click', saveMaterial);
    }

    const modal = document.getElementById('material-modal');
    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === e.currentTarget) closeModal();
        });
    }

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeModal();
    });
});