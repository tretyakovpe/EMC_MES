// ========== МОДУЛЬ ОТГРУЗОК ==========

const ShipmentsModule = {
    currentShipment: null,
    shipmentItems: [],
    shipments: [],

    async render(container) {
        container.innerHTML = `
            <div class="logistics-container">
                <div class="shipments-layout">
                    <!-- Левая панель: список отгрузок -->
                    <div class="shipments-list-panel">
                        <div class="panel-header">
                            <h3>🚛 Отгрузки</h3>
                            <button id="create-shipment-btn" class="btn-primary">➕ Новая отгрузка</button>
                            <button id="clipboard-shipment-btn" class="btn-secondary">📋 Из буфера</button>
                        </div>
                        <div class="shipments-grid" id="shipments-list">
                            <div class="loading">Загрузка...</div>
                        </div>
                    </div>
                    
                    <!-- Правая панель: создание отгрузки -->
<div class="shipment-editor-panel" id="shipment-editor" style="display: none;">
    <div class="panel-header">
        <h3>📦 Сборка отгрузки</h3>
        <button id="close-editor-btn" class="btn-close">✕</button>
    </div>
    <div class="shipment-number-input">
        <label>Номер накладной:</label>
        <input type="text" id="shipment-number" placeholder="Введите номер накладной" class="form-input">
    </div>
    <div class="shipment-number-input">
        <label>Дата отгрузки:</label>
        <input type="date" id="shipment-date" class="form-input">
    </div>
    <div class="shipment-items-list" id="shipment-items-list">
        <div class="empty-items">Добавьте материалы</div>
    </div>
    <div class="shipment-actions">
        <button id="add-material-btn" class="btn-secondary">➕ Добавить материал</button>
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

        await this.loadShipments();
        this.attachEvents();
    },

    async loadShipments() {
        try {
            // Запрашиваем только незавершённые отгрузки (done = false)
            const shipments = await API.getShipments({ done: false });
            this.shipments = shipments;
            this.renderShipmentsList();
        } catch (error) {
            console.error('Ошибка загрузки отгрузок:', error);
            const container = document.getElementById('shipments-list');
            if (container) {
                container.innerHTML = '<div class="empty-state">❌ Ошибка загрузки отгрузок</div>';
            }
        }
    },

    renderShipmentsList() {
        const container = document.getElementById('shipments-list');
        if (!container) return;

        if (this.shipments.length === 0) {
            container.innerHTML = '<div class="empty-state">Нет отгрузок за сегодня</div>';
            return;
        }

        container.innerHTML = this.shipments.map(s => `
        <div class="shipment-card ${s.done ? 'done' : (s.completed ? 'completed' : 'active')}" 
             data-id="${s.shipmentId}"
             onclick="ShipmentsModule.viewShipment(${s.shipmentId})">
            <div class="shipment-header">
                <span class="shipment-number">№${s.number || s.shipmentId}</span>
                <span class="shipment-status">${this.getStatusText(s)}</span>
            </div>
            <div class="shipment-date">${this.formatDate(s.date)}</div>
            <div class="shipment-progress">
                <div class="progress-bar"><div class="progress-fill" style="width: ${s.progress}%"></div></div>
                <span>${s.progress}%</span>
            </div>
                    <button class="edit-shipment-btn" onclick="event.stopPropagation(); ShipmentsModule.editShipment(${s.shipmentId})">✏️</button>                    
            <div class="shipment-actions-row">
                ${!s.done ? `
                    <button class="delete-shipment-btn" onclick="event.stopPropagation(); ShipmentsModule.deleteShipment(${s.shipmentId})" title="Удалить">🗑</button>
                ` : ''}
            </div>
        </div>
    `).join('');
    },

    getStatusText(shipment) {
        if (shipment.done) return '🚚 Отгружена';
        if (shipment.completed) return '📦 Готова';
        return '🔄 В работе';
    },

    formatDate(dateStr) {
        const parts = dateStr.split('-');
        if (parts.length === 3) {
            return `${parts[2]}.${parts[1]}.${parts[0]}`;
        }
        return dateStr;
    },

    attachEvents() {
        const createBtn = document.getElementById('create-shipment-btn');
        if (createBtn) {
            createBtn.onclick = () => this.startNewShipment();
        }

        const closeBtn = document.getElementById('close-editor-btn');
        if (closeBtn) {
            closeBtn.onclick = () => this.closeEditor();
        }

        const addMaterialBtn = document.getElementById('add-material-btn');
        if (addMaterialBtn) {
            addMaterialBtn.onclick = () => this.showMaterialsPicker();
        }

        const saveBtn = document.getElementById('save-shipment-btn');
        if (saveBtn) {
            saveBtn.onclick = () => this.saveShipment();
        }

        const closePickerBtn = document.querySelector('.close-picker-btn');
        if (closePickerBtn) {
            closePickerBtn.onclick = () => this.closePicker();
        }
        const clipboardBtn = document.getElementById('clipboard-shipment-btn');
        if (clipboardBtn) {
            clipboardBtn.onclick = () => this.showClipboardModal();
        }

    },

    startNewShipment() {
        this.shipmentItems = [];
        this.currentShipmentNumber = null;
        document.getElementById('shipment-number').value = '';

        // Устанавливаем завтрашнюю дату
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        document.getElementById('shipment-date').value = tomorrow.toISOString().slice(0, 10);

        this.renderShipmentItems();
        this.showMaterialsPicker();
    },

    async showMaterialsPicker() {
        try {
            // Получаем доступные коробки (со статусом "Произведена")
            const boxes = await API.getBoxes({ status: 'Произведена', limit: 0 });

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
            const materialsList = document.getElementById('materials-list');
            const picker = document.getElementById('materials-picker');

            if (materials.length === 0) {
                materialsList.innerHTML = '<div class="empty-state">Нет доступных коробок для отгрузки</div>';
            } else {
                materialsList.innerHTML = materials.map(m => `
                    <div class="material-item" data-material-id="${m.materialId}" data-material-code="${m.materialCode}" data-available="${m.availableBoxes}">
                        <div class="material-info">
                            <strong>${this.escapeHtml(m.materialCode)}</strong>
                            <span>📦 ${m.availableBoxes} коробок</span>
                        </div>
                        <button class="add-material-btn">➕ Добавить</button>
                    </div>
                `).join('');
            }

            picker.style.display = 'flex';

            materialsList.querySelectorAll('.add-material-btn').forEach(btn => {
                btn.onclick = (e) => {
                    const item = btn.closest('.material-item');
                    const materialId = parseInt(item.dataset.materialId);
                    const materialCode = item.dataset.materialCode;
                    const maxBoxes = parseInt(item.dataset.available);

                    const count = prompt(`Сколько коробок ${materialCode} добавить? (доступно: ${maxBoxes})`, '1');
                    if (count && parseInt(count) > 0) {
                        this.addToShipment(materialId, materialCode, parseInt(count));
                        // Закрываем пикер после добавления
                        this.closePicker();
                    }
                };
            });
        } catch (error) {
            console.error('Ошибка загрузки материалов:', error);
            alert('Ошибка загрузки доступных материалов');
        }
    },

    addToShipment(materialId, materialCode, boxesCount) {
        const existing = this.shipmentItems.find(i => i.materialId === materialId);
        if (existing) {
            existing.boxes += boxesCount;
        } else {
            this.shipmentItems.push({
                materialId: materialId,
                materialCode: materialCode,
                boxes: boxesCount
            });
        }
        this.renderShipmentItems();
    },

    renderShipmentItems() {
        const container = document.getElementById('shipment-items-list');
        const editor = document.getElementById('shipment-editor');

        if (this.shipmentItems.length === 0) {
            container.innerHTML = '<div class="empty-items">Добавьте материалы</div>';
            editor.style.display = 'none';
            return;
        }

        editor.style.display = 'flex';

        // Рендерим список материалов
        let itemsHtml = this.shipmentItems.map((item, idx) => `
            <div class="shipment-item-card">
                <div class="item-info">
                    <strong>${this.escapeHtml(item.materialCode)}</strong>
                    <span>📦 ${item.boxes} кор.</span>
                </div>
                <div class="item-actions">
                    <button class="item-delete" onclick="ShipmentsModule.removeShipmentItem(${idx})">🗑</button>
                </div>
            </div>
        `).join('');

        // Добавляем кнопку "Добавить материал" в конце списка
        itemsHtml += `
            <div class="add-material-card">
                <button class="add-material-btn-full">➕ Добавить материал</button>
            </div>
        `;

        container.innerHTML = itemsHtml;

        // Привязываем обработчик для кнопки добавления
        const addBtn = container.querySelector('.add-material-btn-full');
        if (addBtn) {
            addBtn.onclick = () => this.showMaterialsPicker();
        }
    },

    removeShipmentItem(index) {
        this.shipmentItems.splice(index, 1);
        this.renderShipmentItems();
        // Если материалов не осталось, скрываем редактор
        if (this.shipmentItems.length === 0) {
            document.getElementById('shipment-editor').style.display = 'none';
        }
    },

    async saveShipment() {
        if (this.shipmentItems.length === 0) {
            alert('Добавьте хотя бы один материал');
            return;
        }

        const shipmentNumber = document.getElementById('shipment-number')?.value.trim();
        if (!shipmentNumber) {
            alert('Введите номер накладной');
            return;
        }

        const shipmentDate = document.getElementById('shipment-date')?.value;
        if (!shipmentDate) {
            alert('Выберите дату отгрузки');
            return;
        }

        const details = this.shipmentItems.map(item => ({
            materialId: item.materialId,
            boxes: item.boxes,
            amount: item.boxes * 50
        }));

        try {
            let result;
            if (this.editingShipmentId) {
                // Обновляем существующую отгрузку
                result = await API.updateShipment(this.editingShipmentId, {
                    number: parseInt(shipmentNumber),
                    date: shipmentDate,
                    details: details
                });
                alert(`Отгрузка №${shipmentNumber} обновлена`);
            } else {
                // Создаём новую отгрузку
                result = await API.createShipment({
                    number: parseInt(shipmentNumber),
                    date: shipmentDate,
                    details: details
                });
                alert(`Отгрузка №${shipmentNumber} создана`);
            }

            this.shipmentItems = [];
            this.editingShipmentId = null;
            this.renderShipmentItems();
            await this.loadShipments();
            this.closeEditor();

            // Восстанавливаем текст кнопки
            const saveBtn = document.getElementById('save-shipment-btn');
            if (saveBtn) {
                saveBtn.textContent = '💾 Сохранить отгрузку';
            }

        } catch (error) {
            console.error('Ошибка сохранения отгрузки:', error);
            alert('Ошибка сохранения отгрузки: ' + error.message);
        }
    },

    async viewShipment(shipmentId) {
        try {
            const shipment = await API.getShipmentById(shipmentId);
            let detailsText = '';
            if (shipment.details && shipment.details.length) {
                detailsText = shipment.details.map(d =>
                    `${d.materialCode}: ${d.scannedBoxes}/${d.boxes} кор.`
                ).join('\n');
            }
            alert(`Отгрузка №${shipment.number || shipment.shipmentId}\nДата: ${this.formatDate(shipment.date)}\nСтатус: ${this.getStatusText(shipment)}\nПрогресс: ${shipment.progress}%\n\nПозиции:\n${detailsText || 'нет'}`);
        } catch (error) {
            console.error('Ошибка:', error);
            alert('Ошибка загрузки деталей отгрузки');
        }
    },
    async editShipment(shipmentId) {
        try {
            // 1. Получаем данные отгрузки
            const shipment = await API.getShipmentById(shipmentId);
            if (!shipment) {
                alert('Отгрузка не найдена');
                return;
            }

            // 2. Проверяем, что отгрузка не завершена
            if (shipment.done) {
                alert('Нельзя редактировать завершённую отгрузку');
                return;
            }

            // 3. Показываем редактор
            const editor = document.getElementById('shipment-editor');
            if (!editor) {
                console.error('Editor not found');
                return;
            }

            // 4. Заполняем поля
            const numberInput = document.getElementById('shipment-number');
            const dateInput = document.getElementById('shipment-date');

            if (numberInput) {
                numberInput.value = shipment.number || '';
            }
            if (dateInput) {
                dateInput.value = shipment.date || new Date().toISOString().slice(0, 10);
            }

            // 5. Преобразуем детали в формат shipmentItems
            this.shipmentItems = shipment.details.map(d => ({
                materialId: d.materialId,
                materialCode: d.materialCode,
                boxes: d.boxes,
                scannedBoxes: d.scannedBoxes || 0
            }));

            // 6. Сохраняем ID отгрузки для обновления
            this.editingShipmentId = shipmentId;

            // 7. Отображаем список материалов
            this.renderShipmentItems();
            editor.style.display = 'flex';

            // 8. Меняем текст кнопки "Сохранить"
            const saveBtn = document.getElementById('save-shipment-btn');
            if (saveBtn) {
                saveBtn.textContent = '💾 Обновить отгрузку';
            }

            // 9. Показываем кнопку "Отмена", если её нет
            let cancelBtn = document.getElementById('cancel-edit-btn');
            if (!cancelBtn) {
                const actionsDiv = document.querySelector('.shipment-actions');
                if (actionsDiv) {
                    const newCancelBtn = document.createElement('button');
                    newCancelBtn.id = 'cancel-edit-btn';
                    newCancelBtn.className = 'btn-secondary';
                    newCancelBtn.textContent = '✕ Отменить';
                    newCancelBtn.onclick = () => this.closeEditor();
                    actionsDiv.prepend(newCancelBtn);
                }
            }

        } catch (error) {
            console.error('Ошибка загрузки отгрузки для редактирования:', error);
            alert('Не удалось загрузить отгрузку для редактирования');
        }
    },
    async deleteShipment(shipmentId) {
        if (!confirm('Удалить отгрузку?')) return;
        try {
            await API.deleteShipment(shipmentId);
            await this.loadShipments();
        } catch (error) {
            console.error('Ошибка удаления:', error);
            alert('Ошибка удаления: ' + error.message);
        }
    },

    closeEditor() {
        this.shipmentItems = [];
        this.editingShipmentId = null;
        this.renderShipmentItems();

        const editor = document.getElementById('shipment-editor');
        const numberInput = document.getElementById('shipment-number');
        const dateInput = document.getElementById('shipment-date');

        if (editor) {
            editor.style.display = 'none';
        }
        if (numberInput) {
            numberInput.value = '';
        }
        if (dateInput) {
            dateInput.value = '';
        }

        // Восстанавливаем текст кнопки "Сохранить"
        const saveBtn = document.getElementById('save-shipment-btn');
        if (saveBtn) {
            saveBtn.textContent = '💾 Сохранить отгрузку';
        }

        // Удаляем кнопку "Отмена"
        const cancelBtn = document.getElementById('cancel-edit-btn');
        if (cancelBtn) {
            cancelBtn.remove();
        }
    },

    closePicker() {
        document.getElementById('materials-picker').style.display = 'none';
    },

    escapeHtml(str) {
        if (!str) return '';
        return String(str).replace(/[&<>]/g, function (m) {
            if (m === '&') return '&amp;';
            if (m === '<') return '&lt;';
            if (m === '>') return '&gt;';
            return m;
        });
    },

    showClipboardModal() {
        // Создаём модальное окно
        const modal = document.createElement('div');
        modal.className = 'clipboard-modal';
        modal.innerHTML = `
        <div class="clipboard-modal-content">
            <div class="clipboard-modal-header">
                <h3>📋 Создание отгрузки из буфера обмена</h3>
                <button class="close-modal-btn">✕</button>
            </div>
            <div class="clipboard-modal-body">
                <div class="clipboard-step">
                    <label>Шаг 1: Вставьте данные (Ctrl+V)</label>
                    <textarea id="clipboard-text" rows="8" placeholder='Вставьте сюда строки из накладной 1С...'></textarea>
                    <button id="parse-clipboard-btn" class="btn-primary">🔍 Распознать и проверить</button>
                </div>
                <div class="clipboard-step" id="parse-result-section" style="display: none;">
                    <label>Шаг 2: Результат распознавания</label>
                    <div id="parse-result-table"></div>
                    <div class="shipment-number-input" style="margin-top: 16px;">
                        <label>Номер накладной:</label>
                        <input type="text" id="modal-shipment-number" placeholder="Введите номер накладной" class="form-input">
                    </div>
                    <button id="create-from-clipboard-btn" class="btn-success">✅ Оформить отгрузку</button>
                </div>
            </div>
        </div>
    `;
        document.body.appendChild(modal);

        // Закрытие модального окна
        modal.querySelector('.close-modal-btn').onclick = () => modal.remove();
        modal.onclick = (e) => { if (e.target === modal) modal.remove(); };

        // Парсинг буфера
        document.getElementById('parse-clipboard-btn').onclick = async () => {
            const text = document.getElementById('clipboard-text').value;
            if (!text.trim()) {
                alert('Вставьте данные');
                return;
            }

            try {
                const response = await fetch('/api/shipments/parse-clipboard', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ text: text })
                });
                const result = await response.json();

                if (result.errors && result.errors.length) {
                    alert('Ошибки:\n' + result.errors.join('\n'));
                }

                if (result.warnings && result.warnings.length) {
                    console.log('Предупреждения:', result.warnings);
                    // Можно показать неблокирующим уведомлением
                    const warningDiv = document.createElement('div');
                    warningDiv.style.cssText = 'background: #ffcc00; color: #333; padding: 8px; margin: 10px 0; border-radius: 4px;';
                    warningDiv.innerHTML = '⚠️ ' + result.warnings.join('<br>⚠️ ');
                    document.getElementById('parse-result-section').prepend(warningDiv);
                }
                this.renderParseResult(result);
                document.getElementById('parse-result-section').style.display = 'block';
            } catch (error) {
                console.error('Ошибка парсинга:', error);
                alert('Ошибка при распознавании данных');
            }
        };

        // Создание отгрузки из распознанных данных
        document.getElementById('create-from-clipboard-btn').onclick = async () => {
            const shipmentNumber = document.getElementById('modal-shipment-number').value.trim();
            if (!shipmentNumber) {
                alert('Введите номер накладной');
                return;
            }

            // Собираем только валидные строки
            const validRows = this.parsedRows.filter(row => row.valid);
            if (validRows.length === 0) {
                alert('Нет валидных позиций для создания отгрузки');
                return;
            }

            const details = validRows.map(row => ({
                materialId: row.materialId,
                boxes: row.boxes,
                amount: row.amount
            }));

            try {
                const result = await API.createShipment({
                    number: parseInt(shipmentNumber),
                    date: new Date().toISOString().slice(0, 10),
                    details: details
                });
                alert(`Отгрузка №${shipmentNumber} создана`);
                modal.remove();
                await this.loadShipments();
            } catch (error) {
                console.error('Ошибка создания отгрузки:', error);
                alert('Ошибка создания отгрузки: ' + error.message);
            }
        };
    },

    renderParseResult(result) {
        this.parsedRows = result.rows;
        const container = document.getElementById('parse-result-table');

        let html = '<table class="parse-result-table">';
        html += '<thead><th> Артикул </th><th> Материал </th><th> Штук </th><th> HU </th><th> Статус </th></thead><tbody>';

        for (const row of result.rows) {
            const status = row.valid ? '✅' : `❌ ${row.error || 'Не найден'}`;
            html += `<tr class="${row.valid ? 'valid-row' : 'invalid-row'}">
                    <td>${this.escapeHtml(row.customerCode)} </td>
                    <td>${this.escapeHtml(row.materialCode || '—')} </td>
                    <td>${row.amount} </td>
                    <td>${row.boxes} </td>
                    <td>${status}</td>
                 </tr>`;
        }

        html += '</tbody></table>';
        container.innerHTML = html;
    }
};

// Глобальные функции для onclick
window.ShipmentsModule = ShipmentsModule;