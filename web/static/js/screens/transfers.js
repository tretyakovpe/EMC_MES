console.log('TransfersModule: файл загружен');

const TransfersModule = {
    transferOrders: [],
    warehouses: [],
    materials: [],
    editingOrderId: null,
    transferItems: [],
    selectedFromWarehouse: null,
    selectedToWarehouse: null,
    selectedPlannedDate: null,

    async render(container) {
        console.log('TransfersModule.render() START');
        this.container = container;

        // Загружаем данные
        await this.loadWarehouses();
        await this.loadMaterials();
        await this.loadTransferOrders();

        container.innerHTML = `
            <div class="transfers-container">
                <div class="transfers-layout">
                    <!-- Левая панель: список заказов -->
                    <div class="transfers-list-panel">
                        <div class="panel-header">
                            <h3>📦 Перемещения</h3>
                            <button id="create-transfer-btn" class="btn-primary">➕ Создать заказ</button>
                        </div>
                        <div class="transfers-grid" id="transfers-list">
                            ${this.renderTransferOrdersList()}
                        </div>
                    </div>
                    
                    <!-- Правая панель: создание/редактирование заказа -->
                    <div class="transfer-editor-panel" id="transfer-editor" style="display: none;">
                        <div class="panel-header">
                            <h3>📦 Создание заказа на перемещение</h3>
                            <button id="close-editor-btn" class="btn-close">✕</button>
                        </div>
                        
                        <div class="transfer-route-input">
                            <div class="form-group">
                                <label>Откуда:</label>
                                <select id="transfer-from" class="form-input">
                                    <option value="">Выберите склад</option>
                                    ${this.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                </select>
                            </div>
                            <div class="form-group">
                                <label>Куда:</label>
                                <select id="transfer-to" class="form-input">
                                    <option value="">Выберите склад</option>
                                    ${this.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                </select>
                            </div>
                        </div>

                        <div class="transfer-date-input">
                            <div class="form-group">
                                <label>📅 Плановая дата:</label>
                                <input type="date" id="transfer-planned-date" class="form-input">
                            </div>
                        </div>

                        <div class="transfer-items-list" id="transfer-items-list">
                            <div class="empty-items">Добавьте материалы</div>
                        </div>
                        
                        <div class="transfer-actions">
                            <button id="add-material-btn" class="btn-secondary">➕ Добавить материал</button>
                            <button id="save-transfer-btn" class="btn-success">💾 Сохранить заказ</button>
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

        console.log('HTML rendered');
        this.attachEvents();
        this.initWebSocket();
        console.log('TransfersModule.render() END');
    },

    renderTransferOrdersList() {
        console.log('renderTransferOrdersList() called, orders:', this.transferOrders.length);
        if (this.transferOrders.length === 0) {
            return '<div class="empty-state">Нет заказов на перемещение</div>';
        }

        const getStatus = (order) => {
            if (order.completed) return '✅ Завершён';
            return '🔄 В работе';
        };

        const getStatusColor = (order) => {
            if (order.completed) return '#28a745';
            return '#ffc107';
        };

        return this.transferOrders.map(o => `
            <div class="transfer-card ${o.completed ? 'completed' : 'active'}" 
                 data-id="${o.transferOrderId}"
                 onclick="TransfersModule.viewTransferOrder(${o.transferOrderId})">
                <div class="transfer-header">
                    <span class="transfer-number">№${o.number}</span>
                    <span class="transfer-status" style="background: ${getStatusColor(o)}">
                        ${getStatus(o)}
                    </span>
                </div>
                <div class="transfer-body">
                    <div class="transfer-route">
                        <span class="warehouse-from">🏢 ${o.fromWarehouseCode}</span>
                        <span class="transfer-arrow">➜</span>
                        <span class="warehouse-to">🏢 ${o.toWarehouseCode}</span>
                    </div>
                    <div class="transfer-materials">
                        ${o.details.map(d =>
            `<div class="transfer-material-item">
                                <strong>${this.escapeHtml(d.materialCode)}</strong>
                                <span>${d.quantity} шт.</span>
                            </div>`
        ).join('')}
                    </div>
                    <div class="transfer-date">
                        <span>📅 ${o.plannedDate || o.date}</span>
                    </div>
                </div>
                <div class="transfer-actions-row">
                    ${!o.completed ? `
                        <button class="edit-transfer-btn" onclick="event.stopPropagation(); TransfersModule.editTransferOrder(${o.transferOrderId})">✏️</button>
                        <button class="delete-transfer-btn" onclick="event.stopPropagation(); TransfersModule.deleteTransferOrder(${o.transferOrderId})">🗑</button>
                        <button class="complete-transfer-btn" onclick="event.stopPropagation(); TransfersModule.completeTransferOrder(${o.transferOrderId})">✅ Завершить</button>
                    ` : ''}
                </div>
            </div>
        `).join('');
    },

    async loadWarehouses() {
        try {
            const response = await fetch('/api/warehouses?active=true');
            if (response.ok) {
                this.warehouses = await response.json();
                console.log('Загружено складов:', this.warehouses.length);
            }
        } catch (error) {
            console.error('Ошибка загрузки складов:', error);
        }
    },

    async loadMaterials() {
        try {
            const response = await fetch('/api/materials');
            if (response.ok) {
                this.materials = await response.json();
                console.log('Загружено материалов:', this.materials.length);
            }
        } catch (error) {
            console.error('Ошибка загрузки материалов:', error);
        }
    },

    async loadTransferOrders() {
        try {
            const response = await fetch('/api/transfer-orders?completed=false');
            if (response.ok) {
                this.transferOrders = await response.json();
                console.log('Загружено заказов:', this.transferOrders.length);
            }
        } catch (error) {
            console.error('Ошибка загрузки заказов:', error);
        }
    },

    attachEvents() {
        document.getElementById('create-transfer-btn')?.addEventListener('click', () => {
            this.startNewTransferOrder();
        });

        document.getElementById('close-editor-btn')?.addEventListener('click', () => {
            this.closeEditor();
        });

        document.getElementById('add-material-btn')?.addEventListener('click', () => {
            this.showMaterialsPicker();
        });

        document.getElementById('save-transfer-btn')?.addEventListener('click', () => {
            this.saveTransferOrder();
        });

        document.querySelector('.close-picker-btn')?.addEventListener('click', () => {
            this.closePicker();
        });

        document.getElementById('transfer-from')?.addEventListener('change', (e) => {
            this.selectedFromWarehouse = parseInt(e.target.value);
        });

        document.getElementById('transfer-to')?.addEventListener('change', (e) => {
            this.selectedToWarehouse = parseInt(e.target.value);
        });

        document.getElementById('transfer-planned-date')?.addEventListener('change', (e) => {
            this.selectedPlannedDate = e.target.value;
        });
    },

    startNewTransferOrder() {
        this.transferItems = [];
        this.editingOrderId = null;

        const fromSelect = document.getElementById('transfer-from');
        const toSelect = document.getElementById('transfer-to');
        const dateInput = document.getElementById('transfer-planned-date');

        if (fromSelect) fromSelect.value = '';
        if (toSelect) toSelect.value = '';

        // Устанавливаем завтрашнюю дату по умолчанию
        if (dateInput) {
            const tomorrow = new Date();
            tomorrow.setDate(tomorrow.getDate() + 1);
            dateInput.value = tomorrow.toISOString().slice(0, 10);
            this.selectedPlannedDate = dateInput.value;
        }

        this.selectedFromWarehouse = null;
        this.selectedToWarehouse = null;

        this.renderTransferItems();
        document.getElementById('transfer-editor').style.display = 'flex';

        const saveBtn = document.getElementById('save-transfer-btn');
        if (saveBtn) {
            saveBtn.textContent = '💾 Сохранить заказ';
        }
    },

    showMaterialsPicker() {
        const materialsList = document.getElementById('materials-list');
        const picker = document.getElementById('materials-picker');

        if (this.materials.length === 0) {
            materialsList.innerHTML = '<div class="empty-state">Нет доступных материалов</div>';
        } else {
            materialsList.innerHTML = this.materials.map(m => `
                <div class="material-item" data-material-id="${m.materialId}" data-material-code="${m.materialCode}">
                    <div class="material-info">
                        <strong>${this.escapeHtml(m.materialCode)}</strong>
                        <span>${this.escapeHtml(m.description || '')}</span>
                        <span style="font-size: 12px; color: #6c757d;">${this.escapeHtml(m.customerCode || '')}</span>
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

                const count = prompt(`Сколько ${materialCode} переместить?`, '1');
                if (count && parseInt(count) > 0) {
                    this.addToTransfer(materialId, materialCode, parseInt(count));
                    this.closePicker();
                }
            };
        });
    },

    addToTransfer(materialId, materialCode, quantity) {
        const existing = this.transferItems.find(i => i.materialId === materialId);
        if (existing) {
            existing.quantity += quantity;
        } else {
            this.transferItems.push({
                materialId: materialId,
                materialCode: materialCode,
                quantity: quantity
            });
        }
        this.renderTransferItems();
    },

    renderTransferItems() {
        const container = document.getElementById('transfer-items-list');
        if (!container) {
            console.error('transfer-items-list not found');
            return;
        }

        if (this.transferItems.length === 0) {
            container.innerHTML = '<div class="empty-items">Добавьте материалы</div>';
            return;
        }

        let itemsHtml = this.transferItems.map((item, idx) => `
            <div class="transfer-item-card">
                <div class="item-info">
                    <strong>${this.escapeHtml(item.materialCode)}</strong>
                    <span>📦 ${item.quantity} шт.</span>
                </div>
                <div class="item-actions">
                    <button class="item-delete" onclick="TransfersModule.removeTransferItem(${idx})">🗑</button>
                </div>
            </div>
        `).join('');

        itemsHtml += `
            <div class="add-material-card">
                <button class="add-material-btn-full">➕ Добавить материал</button>
            </div>
        `;

        container.innerHTML = itemsHtml;

        const addBtn = container.querySelector('.add-material-btn-full');
        if (addBtn) {
            addBtn.onclick = () => this.showMaterialsPicker();
        }
    },

    removeTransferItem(index) {
        this.transferItems.splice(index, 1);
        this.renderTransferItems();
    },

    async saveTransferOrder() {
        if (this.transferItems.length === 0) {
            alert('Добавьте хотя бы один материал');
            return;
        }

        if (!this.selectedFromWarehouse) {
            alert('Выберите склад "Откуда"');
            return;
        }

        if (!this.selectedToWarehouse) {
            alert('Выберите склад "Куда"');
            return;
        }

        if (this.selectedFromWarehouse === this.selectedToWarehouse) {
            alert('Склады должны быть разными');
            return;
        }

        if (!this.selectedPlannedDate) {
            alert('Выберите плановую дату');
            return;
        }

        const details = this.transferItems.map(item => ({
            materialId: item.materialId,
            quantity: item.quantity
        }));

        try {
            let result;
            if (this.editingOrderId) {
                // Получаем существующий заказ для номера
                const existingOrder = this.transferOrders.find(o => o.transferOrderId === this.editingOrderId);

                result = await fetch(`/api/transfer-orders/${this.editingOrderId}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        number: existingOrder.number,
                        fromWarehouseId: this.selectedFromWarehouse,
                        toWarehouseId: this.selectedToWarehouse,
                        plannedDate: this.selectedPlannedDate,
                        details: details
                    })
                });

                if (result.ok) {
                    alert(`Заказ №${existingOrder.number} обновлён`);
                }
            } else {
                result = await fetch('/api/transfer-orders', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        fromWarehouseId: this.selectedFromWarehouse,
                        toWarehouseId: this.selectedToWarehouse,
                        plannedDate: this.selectedPlannedDate,
                        details: details
                    })
                });

                if (result.ok) {
                    const data = await result.json();
                    alert(`Заказ №${data.number} создан`);
                }
            }

            if (!result.ok) {
                const err = await result.json();
                throw new Error(err.message || 'Неизвестная ошибка');
            }

            this.transferItems = [];
            this.editingOrderId = null;
            this.selectedFromWarehouse = null;
            this.selectedToWarehouse = null;
            this.selectedPlannedDate = null;

            this.renderTransferItems();
            await this.loadTransferOrders();

            const list = document.getElementById('transfers-list');
            if (list) {
                list.innerHTML = this.renderTransferOrdersList();
            }

            this.closeEditor();

            const saveBtn = document.getElementById('save-transfer-btn');
            if (saveBtn) {
                saveBtn.textContent = '💾 Сохранить заказ';
            }

        } catch (error) {
            console.error('Ошибка сохранения заказа:', error);
            alert('Ошибка сохранения заказа: ' + error.message);
        }
    },

    async viewTransferOrder(orderId) {
        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`);
            if (!response.ok) throw new Error('Заказ не найден');
            const order = await response.json();

            let detailsText = order.details.map(d =>
                `${d.materialCode}: ${d.quantity} шт.`
            ).join('\n');

            const status = order.completed ? '✅ Завершён' : '🔄 В работе';

            alert(`📦 ЗАКАЗ №${order.number}
─────────────────────
📋 Статус: ${status}
🏢 Откуда: ${order.fromWarehouseCode} - ${order.fromWarehouseName}
🏢 Куда: ${order.toWarehouseCode} - ${order.toWarehouseName}
📅 Создан: ${order.date}
📅 Плановая дата: ${order.plannedDate || 'не указана'}

📦 Материалы:
${detailsText || 'нет'}`);
        } catch (error) {
            console.error('Ошибка:', error);
            alert('Ошибка загрузки деталей заказа');
        }
    },

    async editTransferOrder(orderId) {
        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`);
            if (!response.ok) throw new Error('Заказ не найден');
            const order = await response.json();

            if (order.completed) {
                alert('Нельзя редактировать завершённый заказ');
                return;
            }

            const editor = document.getElementById('transfer-editor');
            if (!editor) {
                console.error('Editor not found');
                return;
            }

            const fromSelect = document.getElementById('transfer-from');
            const toSelect = document.getElementById('transfer-to');
            const dateInput = document.getElementById('transfer-planned-date');

            if (fromSelect) {
                fromSelect.value = order.fromWarehouseId;
                this.selectedFromWarehouse = order.fromWarehouseId;
            }
            if (toSelect) {
                toSelect.value = order.toWarehouseId;
                this.selectedToWarehouse = order.toWarehouseId;
            }
            if (dateInput) {
                dateInput.value = order.plannedDate || '';
                this.selectedPlannedDate = order.plannedDate || '';
            }

            this.transferItems = order.details.map(d => ({
                materialId: d.materialId,
                materialCode: d.materialCode,
                quantity: d.quantity
            }));

            this.editingOrderId = orderId;

            this.renderTransferItems();
            editor.style.display = 'flex';

            const saveBtn = document.getElementById('save-transfer-btn');
            if (saveBtn) {
                saveBtn.textContent = '💾 Обновить заказ';
            }

            let cancelBtn = document.getElementById('cancel-edit-btn');
            if (!cancelBtn) {
                const actionsDiv = document.querySelector('.transfer-actions');
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
            console.error('Ошибка загрузки заказа для редактирования:', error);
            alert('Не удалось загрузить заказ для редактирования');
        }
    },

    async completeTransferOrder(orderId) {
        if (!confirm('Завершить заказ?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${orderId}/complete`, {
                method: 'PUT'
            });

            if (response.ok) {
                await this.loadTransferOrders();
                const list = document.getElementById('transfers-list');
                if (list) {
                    list.innerHTML = this.renderTransferOrdersList();
                }
            } else {
                const err = await response.json();
                alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
            }
        } catch (error) {
            console.error('Ошибка завершения заказа:', error);
            alert('Ошибка завершения заказа');
        }
    },

    async deleteTransferOrder(orderId) {
        if (!confirm('Удалить заказ?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`, { method: 'DELETE' });
            if (response.ok) {
                await this.loadTransferOrders();
                const list = document.getElementById('transfers-list');
                if (list) {
                    list.innerHTML = this.renderTransferOrdersList();
                }
            } else {
                const err = await response.json();
                alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
            }
        } catch (error) {
            console.error('Ошибка удаления заказа:', error);
            alert('Ошибка удаления заказа');
        }
    },

    closeEditor() {
        this.transferItems = [];
        this.editingOrderId = null;
        this.selectedFromWarehouse = null;
        this.selectedToWarehouse = null;
        this.selectedPlannedDate = null;

        this.renderTransferItems();

        const editor = document.getElementById('transfer-editor');
        const fromSelect = document.getElementById('transfer-from');
        const toSelect = document.getElementById('transfer-to');
        const dateInput = document.getElementById('transfer-planned-date');

        if (editor) {
            editor.style.display = 'none';
        }
        if (fromSelect) {
            fromSelect.value = '';
        }
        if (toSelect) {
            toSelect.value = '';
        }
        if (dateInput) {
            dateInput.value = '';
        }

        const saveBtn = document.getElementById('save-transfer-btn');
        if (saveBtn) {
            saveBtn.textContent = '💾 Сохранить заказ';
        }

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

    initWebSocket() {
        try {
            const ws = new WebSocket(`ws://${window.location.host}/ws`);

            ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    if (message.type === 'transfer_order_created' ||
                        message.type === 'transfer_order_completed') {
                        this.loadTransferOrders();
                        const list = document.getElementById('transfers-list');
                        if (list) {
                            list.innerHTML = this.renderTransferOrdersList();
                        }
                    }
                } catch (error) {
                    console.error('WebSocket error:', error);
                }
            };

            ws.onclose = () => {
                setTimeout(() => this.initWebSocket(), 5000);
            };
        } catch (error) {
            console.error('WebSocket initialization error:', error);
        }
    },

    destroy() {
        // очистка
    }
};

window.TransfersModule = TransfersModule;