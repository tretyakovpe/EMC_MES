console.log('TransfersModule: файл загружен');

const TransfersModule = {
    transfers: [],
    warehouses: [],
    materials: [],
    currentTransfer: null,
    editingTransferId: null,

    async render(container) {
        console.log('TransfersModule.render() START');
        this.container = container;
        console.log('Container:', container);

        // Загружаем данные
        console.log('▶ Loading warehouses...');
        await this.loadWarehouses();
        console.log('✅ Warehouses loaded:', this.warehouses.length);

        console.log('▶ Loading materials...');
        await this.loadMaterials();
        console.log('✅ Materials loaded:', this.materials.length);

        console.log('▶ Loading transfers...');
        await this.loadTransfers();
        console.log('✅ Transfers loaded:', this.transfers.length);

        console.log('▶ Rendering HTML...');

        container.innerHTML = `
            <div class="transfers-container">
                <div class="transfers-header">
                    <h3>📦 Перемещения</h3>
                    <button id="create-transfer-btn" class="btn-primary">➕ Создать заявку</button>
                </div>
                <div class="transfers-list" id="transfers-list">
                    ${this.renderTransfersList()}
                </div>
            </div>
        `;

        console.log('HTML rendered');
        this.attachEvents();
        this.initWebSocket();
        console.log('TransfersModule.render() END');
    },

    renderTransfersList() {
        console.log('renderTransfersList() called, transfers:', this.transfers.length);
        if (this.transfers.length === 0) {
            return '<div class="empty-state">Нет заявок на перемещение</div>';
        }

        const statusColors = {
            'Создана': '#6c757d',
            'В работе': '#ffc107',
            'Готова': '#17a2b8',
            'Завершена': '#28a745'
        };

        return this.transfers.map(t => `
            <div class="transfer-card" data-id="${t.transferId}">
                <div class="transfer-header">
                    <span class="transfer-number">${t.transferNumber}</span>
                    <span class="transfer-status" style="background: ${statusColors[t.status] || '#6c757d'}">
                        ${t.status}
                    </span>
                </div>
                <div class="transfer-body">
                    <div class="transfer-route">
                        <span class="warehouse-from">🏢 ${t.fromWarehouseCode}</span>
                        <span class="transfer-arrow">➜</span>
                        <span class="warehouse-to">🏢 ${t.toWarehouseCode}</span>
                    </div>
                    <div class="transfer-material">
                        <strong>${t.materialCode}</strong>
                        <span>${t.quantity} шт.</span>
                    </div>
                    <div class="transfer-date">
                        ${t.createdAt}
                        ${t.completedAt ? `<span class="completed-at">✅ ${t.completedAt}</span>` : ''}
                    </div>
                </div>
                <div class="transfer-actions">
                    ${this.getStatusActions(t)}
                </div>
            </div>
        `).join('');
    },

    getStatusActions(transfer) {
        const actions = [];

        if (transfer.status === 'Создана') {
            actions.push(`<button class="btn-start" data-id="${transfer.transferId}">▶ Начать</button>`);
            actions.push(`<button class="btn-delete" data-id="${transfer.transferId}">🗑</button>`);
        } else if (transfer.status === 'В работе') {
            actions.push(`<button class="btn-complete" data-id="${transfer.transferId}">✅ Готово</button>`);
        } else if (transfer.status === 'Готова') {
            actions.push(`<button class="btn-finish" data-id="${transfer.transferId}">✅ Завершить</button>`);
        }

        if (transfer.status !== 'Завершена') {
            actions.push(`<button class="btn-detail" data-id="${transfer.transferId}">📋 Детали</button>`);
        }

        return actions.join('');
    },

    async loadWarehouses() {
        try {
            console.log('  fetch /api/warehouses?active=true');
            const response = await fetch('/api/warehouses?active=true');
            console.log('  response status:', response.status);
            if (response.ok) {
                this.warehouses = await response.json();
                console.log('  warehouses:', this.warehouses);
            } else {
                console.error('  error:', response.status);
            }
        } catch (error) {
            console.error('Ошибка загрузки складов:', error);
        }
    },

    async loadMaterials() {
        try {
            console.log('  fetch /api/materials');
            const response = await fetch('/api/materials');
            console.log('  response status:', response.status);
            if (response.ok) {
                this.materials = await response.json();
                console.log('  materials:', this.materials.length);
            } else {
                console.error('  error:', response.status);
            }
        } catch (error) {
            console.error('Ошибка загрузки материалов:', error);
        }
    },

    async loadTransfers() {
        try {
            console.log('  fetch /api/transfers');
            const response = await fetch('/api/transfers');
            console.log('  response status:', response.status);
            if (response.ok) {
                this.transfers = await response.json();
                console.log('  transfers:', this.transfers.length);
            } else {
                console.error('  error:', response.status);
            }
        } catch (error) {
            console.error('Ошибка загрузки заявок:', error);
        }
    },

    attachEvents() {
        // Создание заявки
        document.getElementById('create-transfer-btn')?.addEventListener('click', () => {
            this.showCreateModal();
        });

        // Обработчики кнопок (делегирование)
        document.addEventListener('click', (e) => {
            const target = e.target;

            if (target.classList.contains('btn-start')) {
                this.updateStatus(target.dataset.id, 'В работе');
            } else if (target.classList.contains('btn-complete')) {
                this.updateStatus(target.dataset.id, 'Готова');
            } else if (target.classList.contains('btn-finish')) {
                this.updateStatus(target.dataset.id, 'Завершена');
            } else if (target.classList.contains('btn-delete')) {
                this.deleteTransfer(target.dataset.id);
            } else if (target.classList.contains('btn-detail')) {
                this.showDetail(target.dataset.id);
            }
        });
    },

    showCreateModal() {
        const modal = document.createElement('div');
        modal.className = 'transfer-modal';
        modal.innerHTML = `
            <div class="transfer-modal-content">
                <div class="transfer-modal-header">
                    <h3>📦 Создание заявки на перемещение</h3>
                    <button class="close-modal-btn">✕</button>
                </div>
                <div class="transfer-modal-body">
                    <div class="form-group">
                        <label>Откуда:</label>
                        <select id="transfer-from" class="form-control">
                            <option value="">Выберите склад</option>
                            ${this.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Куда:</label>
                        <select id="transfer-to" class="form-control">
                            <option value="">Выберите склад</option>
                            ${this.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Материал:</label>
                        <select id="transfer-material" class="form-control">
                            <option value="">Выберите материал</option>
                            ${this.materials.map(m =>
            `<option value="${m.materialId}">${m.materialCode} - ${m.description || ''}</option>`
        ).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Количество (шт.):</label>
                        <input type="number" id="transfer-quantity" class="form-control" min="1" placeholder="Введите количество">
                    </div>
                </div>
                <div class="transfer-modal-footer">
                    <button class="btn-cancel">Отмена</button>
                    <button class="btn-save">💾 Создать</button>
                </div>
            </div>
        `;

        document.body.appendChild(modal);

        // Закрытие
        modal.querySelector('.close-modal-btn').onclick = () => modal.remove();
        modal.querySelector('.btn-cancel').onclick = () => modal.remove();
        modal.onclick = (e) => { if (e.target === modal) modal.remove(); };

        // Сохранение
        modal.querySelector('.btn-save').onclick = async () => {
            const fromId = parseInt(document.getElementById('transfer-from').value);
            const toId = parseInt(document.getElementById('transfer-to').value);
            const materialId = parseInt(document.getElementById('transfer-material').value);
            const quantity = parseInt(document.getElementById('transfer-quantity').value);

            if (!fromId || !toId || !materialId || !quantity) {
                alert('Заполните все поля');
                return;
            }

            if (fromId === toId) {
                alert('Склады должны быть разными');
                return;
            }

            try {
                const response = await fetch('/api/transfers', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ fromWarehouseId: fromId, toWarehouseId: toId, materialId, quantity })
                });

                if (response.ok) {
                    modal.remove();
                    await this.loadTransfers();
                    this.renderTransfersList();
                    // Обновляем список
                    const list = document.getElementById('transfers-list');
                    if (list) {
                        list.innerHTML = this.renderTransfersList();
                    }
                } else {
                    const err = await response.json();
                    alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
                }
            } catch (error) {
                console.error('Ошибка создания заявки:', error);
                alert('Ошибка создания заявки');
            }
        };
    },

    async updateStatus(transferId, status) {
        try {
            const response = await fetch(`/api/transfers/${transferId}/status`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ status })
            });

            if (response.ok) {
                await this.loadTransfers();
                const list = document.getElementById('transfers-list');
                if (list) {
                    list.innerHTML = this.renderTransfersList();
                }
            } else {
                const err = await response.json();
                alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
            }
        } catch (error) {
            console.error('Ошибка обновления статуса:', error);
            alert('Ошибка обновления статуса');
        }
    },

    async deleteTransfer(transferId) {
        if (!confirm('Удалить заявку?')) return;

        try {
            const response = await fetch(`/api/transfers/${transferId}`, { method: 'DELETE' });
            if (response.ok) {
                await this.loadTransfers();
                const list = document.getElementById('transfers-list');
                if (list) {
                    list.innerHTML = this.renderTransfersList();
                }
            } else {
                const err = await response.json();
                alert('Ошибка: ' + (err.message || 'Неизвестная ошибка'));
            }
        } catch (error) {
            console.error('Ошибка удаления заявки:', error);
            alert('Ошибка удаления заявки');
        }
    },

    async showDetail(transferId) {
        try {
            const response = await fetch(`/api/transfers/${transferId}`);
            if (!response.ok) throw new Error('Заявка не найдена');
            const transfer = await response.json();

            alert(`
📦 ЗАЯВКА №${transfer.transferNumber}
─────────────────────
📋 Статус: ${transfer.status}
🏢 Откуда: ${transfer.fromWarehouseCode} - ${transfer.fromWarehouseName}
🏢 Куда: ${transfer.toWarehouseCode} - ${transfer.toWarehouseName}
📦 Материал: ${transfer.materialCode}
📊 Количество: ${transfer.quantity} шт.
🕐 Создана: ${transfer.createdAt}
${transfer.completedAt ? `✅ Завершена: ${transfer.completedAt}` : ''}
            `);
        } catch (error) {
            console.error('Ошибка загрузки деталей:', error);
            alert('Ошибка загрузки деталей заявки');
        }
    },

    initWebSocket() {
        const ws = new WebSocket(`ws://${window.location.host}/ws`);

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                if (message.type === 'transfer_created' || message.type === 'transfer_status_updated') {
                    this.loadTransfers();
                    const list = document.getElementById('transfers-list');
                    if (list) {
                        list.innerHTML = this.renderTransfersList();
                    }
                }
            } catch (error) {
                console.error('WebSocket error:', error);
            }
        };

        ws.onclose = () => {
            setTimeout(() => this.initWebSocket(), 5000);
        };
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

    destroy() {
        // очистка
    }
};

window.TransfersModule = TransfersModule;