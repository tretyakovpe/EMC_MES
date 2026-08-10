// ================================================================
// transfers.js - Модуль управления заказами на перемещение (ЛОГИСТ)
// Версия: 2.0 (с поддержкой статусов)
// ================================================================

console.log('TransfersModule (Logistics) loaded');

const TransfersModule = {
    // ============================================================
    // СОСТОЯНИЕ
    // ============================================================
    state: {
        orders: [],
        selectedOrderId: null,
        currentOrder: null,
        isLoading: false,
        filter: {
            status: '',
            fromWarehouse: '',
            toWarehouse: '',
            fromDate: '',
            toDate: ''
        },
        warehouses: []
    },

    // ============================================================
    // ИНИЦИАЛИЗАЦИЯ
    // ============================================================
    async render(container) {
        console.log('TransfersModule.render() called');

        // Загружаем склады
        await this.loadWarehouses();

        // Рендерим интерфейс
        container.innerHTML = this.getTemplate();

        // Навешиваем события
        this.bindEvents(container);

        // Загружаем заказы
        await this.loadOrders();

        // Если есть заказы, выбираем первый
        if (this.state.orders.length > 0) {
            await this.selectOrder(this.state.orders[0].transferOrderId);
        }
    },

    // ============================================================
    // ШАБЛОН
    // ============================================================
    getTemplate() {
        return `
            <div class="logistics-container transfers-module">
                <div class="transfers-header">
                    <div class="d-flex justify-content-between align-items-center mb-3">
                        <h3 class="m-0">📦 Заказы на перемещение</h3>
                        <div>
                            <button class="btn btn-primary btn-sm me-2" id="transfer-create-btn">
                                ➕ Создать заказ
                            </button>
                            <button class="btn btn-outline-secondary btn-sm" id="transfer-refresh-btn">
                                🔄 Обновить
                            </button>
                        </div>
                    </div>

                    <!-- Фильтры -->
                    <div class="filters-bar mb-3">
                        <div class="row g-2">
                            <div class="col-md-2">
                                <select class="form-select form-select-sm" id="filter-status">
                                    <option value="">Все статусы</option>
                                    <option value="Draft">Создан</option>
                                    <option value="InProgress">В работе</option>
                                    <option value="Ready">Готов</option>
                                    <option value="Completed">Завершен</option>
                                </select>
                            </div>
                            <div class="col-md-2">
                                <select class="form-select form-select-sm" id="filter-from-warehouse">
                                    <option value="">Со склада (все)</option>
                                    ${this.state.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                </select>
                            </div>
                            <div class="col-md-2">
                                <select class="form-select form-select-sm" id="filter-to-warehouse">
                                    <option value="">На склад (все)</option>
                                    ${this.state.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                </select>
                            </div>
                            <div class="col-md-2">
                                <input type="date" class="form-control form-control-sm" id="filter-from-date" placeholder="С даты">
                            </div>
                            <div class="col-md-2">
                                <input type="date" class="form-control form-control-sm" id="filter-to-date" placeholder="По дату">
                            </div>
                            <div class="col-md-2">
                                <button class="btn btn-primary btn-sm w-100" id="filter-apply-btn">Применить</button>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="transfers-body">
                    <div class="row">
                        <!-- Список заказов -->
                        <div class="col-md-4">
                            <div class="orders-list" id="transfers-list">
                                <div class="text-center text-muted py-4">Загрузка...</div>
                            </div>
                        </div>

                        <!-- Детали заказа -->
                        <div class="col-md-8">
                            <div id="transfer-details-panel">
                                <div class="empty-state text-center py-5" id="details-empty">
                                    <div class="text-muted">
                                        <h5>Выберите заказ</h5>
                                        <p class="small">Нажмите на заказ в списке слева</p>
                                    </div>
                                </div>
                                <div id="details-content" style="display: none;">
                                    <!-- Заголовок деталей -->
                                    <div class="details-header mb-3">
                                        <div class="d-flex justify-content-between align-items-start">
                                            <div>
                                                <h4 id="details-title">Заказ №0</h4>
                                                <div>
                                                    <span class="badge" id="details-status-badge">Статус</span>
                                                    <span class="ms-2 text-muted small" id="details-created">Создан: --</span>
                                                </div>
                                            </div>
                                            <div class="btn-group">
                                                <button class="btn btn-outline-primary btn-sm" id="btn-edit-order">✏️ Редактировать</button>
                                                <button class="btn btn-outline-danger btn-sm" id="btn-delete-order">🗑️ Удалить</button>
                                                <button class="btn btn-success btn-sm" id="btn-confirm-order">✅ Подтвердить</button>
                                            </div>
                                        </div>
                                        <div class="row mt-2 small">
                                            <div class="col-md-4">
                                                <span class="text-muted">Со склада:</span>
                                                <strong id="details-from">--</strong>
                                            </div>
                                            <div class="col-md-4">
                                                <span class="text-muted">На склад:</span>
                                                <strong id="details-to">--</strong>
                                            </div>
                                            <div class="col-md-4">
                                                <span class="text-muted">Плановая дата:</span>
                                                <strong id="details-planned">--</strong>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- Прогресс -->
                                    <div class="progress-container mb-3">
                                        <div class="d-flex justify-content-between small">
                                            <span>Прогресс отгрузки</span>
                                            <span id="progress-text">0%</span>
                                        </div>
                                        <div class="progress">
                                            <div class="progress-bar" id="progress-bar" role="progressbar" style="width: 0%"></div>
                                        </div>
                                    </div>

                                    <!-- Материалы -->
                                    <div class="details-section mb-3">
                                        <h6 class="section-title">📋 Материалы</h6>
                                        <div class="table-responsive">
                                            <table class="table table-sm table-hover">
                                                <thead>
                                                    <tr>
                                                        <th>Код</th>
                                                        <th>Описание</th>
                                                        <th class="text-center">План</th>
                                                        <th class="text-center">Отгружено</th>
                                                        <th class="text-center">Остаток</th>
                                                        <th class="text-center">Статус</th>
                                                    </tr>
                                                </thead>
                                                <tbody id="items-tbody">
                                                    <tr><td colspan="6" class="text-muted text-center">Нет материалов</td></tr>
                                                </tbody>
                                            </table>
                                        </div>
                                    </div>

                                    <!-- История отгрузок -->
                                    <div class="details-section">
                                        <h6 class="section-title">📤 История отгрузок</h6>
                                        <div class="table-responsive">
                                            <table class="table table-sm">
                                                <thead>
                                                    <tr>
                                                        <th>Дата</th>
                                                        <th>Материал</th>
                                                        <th class="text-center">Количество</th>
                                                        <th>Кто</th>
                                                    </tr>
                                                </thead>
                                                <tbody id="shipments-tbody">
                                                    <tr><td colspan="4" class="text-muted text-center">Нет отгрузок</td></tr>
                                                </tbody>
                                            </table>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Модальное окно создания/редактирования -->
            <div class="modal fade" id="transferModal" tabindex="-1">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="modal-title">Создание заказа</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div id="modal-error" class="alert alert-danger" style="display: none;"></div>
                            
                            <form id="transfer-form">
                                <div class="row mb-3">
                                    <div class="col-md-3">
                                        <label class="form-label">Номер заказа</label>
                                        <input type="number" class="form-control" id="input-number" required>
                                    </div>
                                    <div class="col-md-3">
                                        <label class="form-label">Плановая дата</label>
                                        <input type="date" class="form-control" id="input-planned-date" required>
                                    </div>
                                    <div class="col-md-3">
                                        <label class="form-label">Со склада</label>
                                        <select class="form-select" id="input-from-warehouse" required>
                                            <option value="">Выберите склад</option>
                                            ${this.state.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                        </select>
                                    </div>
                                    <div class="col-md-3">
                                        <label class="form-label">На склад</label>
                                        <select class="form-select" id="input-to-warehouse" required>
                                            <option value="">Выберите склад</option>
                                            ${this.state.warehouses.map(w =>
            `<option value="${w.warehouseId}">${w.code} - ${w.name}</option>`
        ).join('')}
                                        </select>
                                    </div>
                                </div>

                                <div class="mb-3">
                                    <label class="form-label">Материалы</label>
                                    <div id="details-editor">
                                        <div class="detail-row row g-2 mb-2">
                                            <div class="col-md-6">
                                                <input type="text" class="form-control form-control-sm material-code-input" placeholder="Код материала">
                                            </div>
                                            <div class="col-md-3">
                                                <input type="number" class="form-control form-control-sm material-qty-input" placeholder="Количество" min="1">
                                            </div>
                                            <div class="col-md-3">
                                                <button type="button" class="btn btn-danger btn-sm remove-detail-btn">✕</button>
                                            </div>
                                        </div>
                                    </div>
                                    <button type="button" class="btn btn-outline-secondary btn-sm mt-2" id="add-detail-btn">➕ Добавить материал</button>
                                </div>
                            </form>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Отмена</button>
                            <button type="button" class="btn btn-primary" id="modal-save-btn">Сохранить</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
    },

    // ============================================================
    // ЗАГРУЗКА ДАННЫХ
    // ============================================================
    async loadWarehouses() {
        try {
            const response = await fetch('/api/warehouses?active=true');
            if (response.ok) {
                this.state.warehouses = await response.json();
                console.log('Загружено складов:', this.state.warehouses.length);
            }
        } catch (error) {
            console.error('Ошибка загрузки складов:', error);
        }
    },

    async loadOrders() {
        this.state.isLoading = true;
        const listEl = document.getElementById('transfers-list');
        if (listEl) listEl.innerHTML = '<div class="text-center text-muted py-4">⏳ Загрузка...</div>';

        try {
            // Собираем параметры фильтрации
            const params = new URLSearchParams();
            if (this.state.filter.status) params.append('status', this.state.filter.status);
            if (this.state.filter.fromWarehouse) params.append('fromWarehouse', this.state.filter.fromWarehouse);
            if (this.state.filter.toWarehouse) params.append('toWarehouse', this.state.filter.toWarehouse);
            if (this.state.filter.fromDate) params.append('fromDate', this.state.filter.fromDate);
            if (this.state.filter.toDate) params.append('toDate', this.state.filter.toDate);

            const url = `/api/transfer-orders?${params.toString()}`;
            const response = await fetch(url);
            if (!response.ok) throw new Error('Ошибка загрузки заказов');

            this.state.orders = await response.json();
            console.log('Загружено заказов:', this.state.orders.length);

            this.renderOrdersList();

            // Если есть заказы и нет выбранного, выбираем первый
            if (this.state.orders.length > 0 && !this.state.selectedOrderId) {
                await this.selectOrder(this.state.orders[0].transferOrderId);
            } else if (this.state.orders.length === 0) {
                this.showEmptyDetails();
            }

        } catch (error) {
            console.error('Ошибка:', error);
            const listEl = document.getElementById('transfers-list');
            if (listEl) listEl.innerHTML = `<div class="text-center text-danger py-4">❌ ${error.message}</div>`;
        }

        this.state.isLoading = false;
    },

    // ============================================================
    // ОТОБРАЖЕНИЕ СПИСКА
    // ============================================================
    renderOrdersList() {
        const listEl = document.getElementById('transfers-list');
        if (!listEl) return;

        if (this.state.orders.length === 0) {
            listEl.innerHTML = '<div class="text-center text-muted py-4">📭 Нет заказов</div>';
            return;
        }

        listEl.innerHTML = this.state.orders.map(o => {
            const statusColor = this.getStatusColor(o.status);
            const statusLabel = this.getStatusLabel(o.status);
            const isSelected = o.transferOrderId === this.state.selectedOrderId;

            return `
                <div class="transfer-card ${isSelected ? 'selected' : ''} card mb-2 cursor-pointer"
                     data-id="${o.transferOrderId}"
                     onclick="TransfersModule.selectOrder(${o.transferOrderId})">
                    <div class="card-body p-3">
                        <div class="d-flex justify-content-between align-items-start">
                            <div>
                                <h6 class="mb-1">№${o.number}</h6>
                                <div class="small text-muted">
                                    ${o.fromWarehouseCode} ➜ ${o.toWarehouseCode}
                                </div>
                                <div class="small text-muted">
                                    📅 ${o.plannedDate}
                                </div>
                            </div>
                            <div>
                                <span class="badge" style="background: ${statusColor};">${statusLabel}</span>
                            </div>
                        </div>
                        <div class="mt-2">
                            <div class="small text-muted">
                                📦 ${o.totalItems} материалов · ${o.totalQuantity} шт.
                            </div>
                            <div class="progress mt-1" style="height: 6px;">
                                <div class="progress-bar" role="progressbar" 
                                     style="width: ${o.progress}%; background: ${statusColor};"></div>
                            </div>
                            <div class="small text-muted text-end">${o.progress}%</div>
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    },

    // ============================================================
    // ВЫБОР ЗАКАЗА
    // ============================================================
    async selectOrder(orderId) {
        if (!orderId) return;
        console.log('Выбор заказа:', orderId);

        this.state.selectedOrderId = orderId;

        // Обновляем выделение в списке
        document.querySelectorAll('.transfer-card').forEach(el => {
            el.classList.toggle('selected', parseInt(el.dataset.id) === orderId);
        });

        // Загружаем детали
        await this.loadOrderDetails(orderId);
    },

    async loadOrderDetails(orderId) {
        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`);
            if (!response.ok) throw new Error('Ошибка загрузки деталей');

            this.state.currentOrder = await response.json();
            this.renderDetails(this.state.currentOrder);

        } catch (error) {
            console.error('Ошибка загрузки деталей:', error);
            this.showEmptyDetails();
        }
    },

    // ============================================================
    // ОТОБРАЖЕНИЕ ДЕТАЛЕЙ
    // ============================================================
    renderDetails(order) {
        document.getElementById('details-empty').style.display = 'none';
        document.getElementById('details-content').style.display = 'block';

        // Заголовок
        document.getElementById('details-title').textContent = `Заказ №${order.number}`;

        const statusBadge = document.getElementById('details-status-badge');
        statusBadge.textContent = this.getStatusLabel(order.status);
        statusBadge.style.background = this.getStatusColor(order.status);

        document.getElementById('details-created').textContent = `Создан: ${order.date || '--'}`;
        document.getElementById('details-from').textContent = `${order.fromWarehouseCode} - ${order.fromWarehouseName}`;
        document.getElementById('details-to').textContent = `${order.toWarehouseCode} - ${order.toWarehouseName}`;
        document.getElementById('details-planned').textContent = order.plannedDate || '--';

        // Прогресс
        const total = order.details.reduce((sum, d) => sum + d.quantity, 0);
        const shipped = order.details.reduce((sum, d) => sum + d.shippedQuantity, 0);
        const progress = total > 0 ? Math.round((shipped / total) * 100) : 0;
        document.getElementById('progress-bar').style.width = `${progress}%`;
        document.getElementById('progress-text').textContent = `${progress}% (${shipped}/${total} шт.)`;

        // Кнопки
        document.getElementById('btn-edit-order').style.display = order.canEdit ? 'inline-block' : 'none';
        document.getElementById('btn-delete-order').style.display = order.canDelete ? 'inline-block' : 'none';
        document.getElementById('btn-confirm-order').style.display = order.canConfirm ? 'inline-block' : 'none';

        // Материалы
        this.renderItems(order.details || []);

        // Отгрузки
        this.renderShipments(order.shipments || []);
    },

    renderItems(items) {
        const tbody = document.getElementById('items-tbody');
        if (!tbody) return;

        if (!items || items.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-muted text-center">Нет материалов</td></tr>`;
            return;
        }

        tbody.innerHTML = items.map(item => {
            const remaining = item.remaining || 0;
            let statusClass = 'text-muted';
            let statusText = 'Ожидает';

            if (item.isFullyShipped) {
                statusClass = 'text-success';
                statusText = '✅ Готово';
            } else if (item.shippedQuantity > 0) {
                statusClass = 'text-warning';
                statusText = '🔄 Частично';
            }

            return `
                <tr>
                    <td><strong>${escapeHtml(item.materialCode)}</strong></td>
                    <td>${escapeHtml(item.description || '')}</td>
                    <td class="text-center">${item.quantity}</td>
                    <td class="text-center">${item.shippedQuantity}</td>
                    <td class="text-center ${remaining <= 0 ? 'text-success' : 'text-warning'}">${remaining}</td>
                    <td class="text-center ${statusClass}">${statusText}</td>
                </tr>
            `;
        }).join('');
    },

    renderShipments(shipments) {
        const tbody = document.getElementById('shipments-tbody');
        if (!tbody) return;

        if (!shipments || shipments.length === 0) {
            tbody.innerHTML = `<tr><td colspan="4" class="text-muted text-center">Нет отгрузок</td></tr>`;
            return;
        }

        tbody.innerHTML = shipments.map(s => `
            <tr>
                <td>${s.createdAt || ''}</td>
                <td>${escapeHtml(s.materialCode)}</td>
                <td class="text-center">${s.quantity}</td>
                <td>${escapeHtml(s.createdBy || '')}</td>
            </tr>
        `).join('');
    },

    showEmptyDetails() {
        document.getElementById('details-empty').style.display = 'flex';
        document.getElementById('details-content').style.display = 'none';
    },

    // ============================================================
    // СОЗДАНИЕ/РЕДАКТИРОВАНИЕ ЗАКАЗА
    // ============================================================
    openCreateModal() {
        this.openModal(null);
    },

    openEditModal(orderId) {
        const order = this.state.orders.find(o => o.transferOrderId === orderId);
        if (order) {
            this.openModal(order);
        }
    },

    async openModal(order) {
        const modal = new bootstrap.Modal(document.getElementById('transferModal'));
        const isEdit = !!order;

        // Заголовок
        document.getElementById('modal-title').textContent = isEdit ? '✏️ Редактирование заказа' : '➕ Создание заказа';
        document.getElementById('modal-error').style.display = 'none';

        // Заполняем поля
        if (isEdit) {
            document.getElementById('input-number').value = order.number;
            document.getElementById('input-planned-date').value = order.plannedDate || '';
            document.getElementById('input-from-warehouse').value = order.fromWarehouseId || '';
            document.getElementById('input-to-warehouse').value = order.toWarehouseId || '';

            // Загружаем детали заказа
            const response = await fetch(`/api/transfer-orders/${order.transferOrderId}`);
            if (response.ok) {
                const fullOrder = await response.json();
                this.renderDetailRows(fullOrder.details || []);
            }
        } else {
            document.getElementById('input-number').value = '';
            document.getElementById('input-planned-date').value = new Date().toISOString().slice(0, 10);
            document.getElementById('input-from-warehouse').value = '';
            document.getElementById('input-to-warehouse').value = '';
            this.renderDetailRows([]);
        }

        // Сохраняем ID для редактирования
        document.getElementById('modal-save-btn').dataset.orderId = isEdit ? order.transferOrderId : '';

        modal.show();
    },

    renderDetailRows(details) {
        const container = document.getElementById('details-editor');
        if (!container) return;

        if (details.length === 0) {
            container.innerHTML = this.getDetailRowTemplate('');
            return;
        }

        container.innerHTML = details.map(d =>
            this.getDetailRowTemplate(d.materialCode, d.quantity)
        ).join('');
    },

    getDetailRowTemplate(materialCode = '', quantity = '') {
        return `
            <div class="detail-row row g-2 mb-2">
                <div class="col-md-6">
                    <input type="text" class="form-control form-control-sm material-code-input" 
                           value="${escapeHtml(materialCode)}" placeholder="Код материала">
                </div>
                <div class="col-md-3">
                    <input type="number" class="form-control form-control-sm material-qty-input" 
                           value="${quantity}" placeholder="Количество" min="1">
                </div>
                <div class="col-md-3">
                    <button type="button" class="btn btn-danger btn-sm remove-detail-btn">✕</button>
                </div>
            </div>
        `;
    },

    async saveOrder() {
        const errorEl = document.getElementById('modal-error');
        errorEl.style.display = 'none';

        // Собираем данные
        const number = parseInt(document.getElementById('input-number').value);
        const plannedDate = document.getElementById('input-planned-date').value;
        const fromWarehouseId = parseInt(document.getElementById('input-from-warehouse').value);
        const toWarehouseId = parseInt(document.getElementById('input-to-warehouse').value);

        // Валидация
        if (!number || number <= 0) {
            this.showModalError('Введите номер заказа');
            return;
        }
        if (!plannedDate) {
            this.showModalError('Выберите плановую дату');
            return;
        }
        if (!fromWarehouseId) {
            this.showModalError('Выберите склад отправителя');
            return;
        }
        if (!toWarehouseId) {
            this.showModalError('Выберите склад получателя');
            return;
        }
        if (fromWarehouseId === toWarehouseId) {
            this.showModalError('Склад отправителя и получателя не могут совпадать');
            return;
        }

        // Собираем материалы
        const rows = document.querySelectorAll('.detail-row');
        const details = [];
        let hasError = false;

        rows.forEach(row => {
            const code = row.querySelector('.material-code-input').value.trim();
            const qty = parseInt(row.querySelector('.material-qty-input').value);

            if (!code) {
                hasError = true;
                return;
            }
            if (!qty || qty <= 0) {
                hasError = true;
                return;
            }

            details.push({ materialCode: code, quantity: qty });
        });

        if (hasError || details.length === 0) {
            this.showModalError('Заполните все поля материалов (код и количество)');
            return;
        }

        const orderId = document.getElementById('modal-save-btn').dataset.orderId;
        const isEdit = !!orderId;

        const payload = {
            number,
            fromWarehouseId,
            toWarehouseId,
            plannedDate,
            details
        };

        try {
            const url = isEdit ? `/api/transfer-orders/${orderId}` : '/api/transfer-orders';
            const method = isEdit ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка сохранения');
            }

            // Закрываем модалку
            bootstrap.Modal.getInstance(document.getElementById('transferModal')).hide();

            // Обновляем данные
            await this.loadOrders();
            if (this.state.orders.length > 0) {
                await this.selectOrder(this.state.orders[0].transferOrderId);
            }

        } catch (error) {
            console.error('Ошибка сохранения:', error);
            this.showModalError(error.message || 'Ошибка при сохранении');
        }
    },

    showModalError(message) {
        const errorEl = document.getElementById('modal-error');
        errorEl.textContent = message;
        errorEl.style.display = 'block';
    },

    // ============================================================
    // УДАЛЕНИЕ ЗАКАЗА
    // ============================================================
    async deleteOrder(orderId) {
        if (!confirm('Удалить заказ?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка удаления');
            }

            await this.loadOrders();
            if (this.state.orders.length > 0) {
                await this.selectOrder(this.state.orders[0].transferOrderId);
            } else {
                this.showEmptyDetails();
            }

        } catch (error) {
            console.error('Ошибка удаления:', error);
            alert('Ошибка удаления: ' + error.message);
        }
    },

    // ============================================================
    // ПОДТВЕРЖДЕНИЕ ЗАКАЗА
    // ============================================================
    async confirmOrder(orderId) {
        if (!confirm('Подтвердить завершение заказа?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${orderId}/confirm`, {
                method: 'POST'
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка подтверждения');
            }

            await this.loadOrders();
            if (this.state.orders.length > 0) {
                await this.selectOrder(this.state.orders[0].transferOrderId);
            }

        } catch (error) {
            console.error('Ошибка подтверждения:', error);
            alert('Ошибка подтверждения: ' + error.message);
        }
    },

    // ============================================================
    // СОБЫТИЯ
    // ============================================================
    bindEvents(container) {
        // Обновить
        container.querySelector('#transfer-refresh-btn')?.addEventListener('click', () => {
            this.loadOrders();
        });

        // Создать
        container.querySelector('#transfer-create-btn')?.addEventListener('click', () => {
            this.openCreateModal();
        });

        // Фильтры
        container.querySelector('#filter-apply-btn')?.addEventListener('click', () => {
            this.state.filter.status = document.getElementById('filter-status').value;
            this.state.filter.fromWarehouse = document.getElementById('filter-from-warehouse').value;
            this.state.filter.toWarehouse = document.getElementById('filter-to-warehouse').value;
            this.state.filter.fromDate = document.getElementById('filter-from-date').value;
            this.state.filter.toDate = document.getElementById('filter-to-date').value;
            this.loadOrders();
        });

        // Кнопки в деталях
        container.querySelector('#btn-edit-order')?.addEventListener('click', () => {
            if (this.state.selectedOrderId) {
                this.openEditModal(this.state.selectedOrderId);
            }
        });

        container.querySelector('#btn-delete-order')?.addEventListener('click', () => {
            if (this.state.selectedOrderId) {
                this.deleteOrder(this.state.selectedOrderId);
            }
        });

        container.querySelector('#btn-confirm-order')?.addEventListener('click', () => {
            if (this.state.selectedOrderId) {
                this.confirmOrder(this.state.selectedOrderId);
            }
        });

        // Модальное окно
        const modal = document.getElementById('transferModal');
        if (modal) {
            // Добавить строку материала
            modal.querySelector('#add-detail-btn')?.addEventListener('click', () => {
                const container = document.getElementById('details-editor');
                container.insertAdjacentHTML('beforeend', this.getDetailRowTemplate());
            });

            // Удалить строку материала (делегирование)
            modal.addEventListener('click', (e) => {
                if (e.target.classList.contains('remove-detail-btn')) {
                    const row = e.target.closest('.detail-row');
                    if (row && document.querySelectorAll('.detail-row').length > 1) {
                        row.remove();
                    } else {
                        alert('Должен быть хотя бы один материал');
                    }
                }
            });

            // Сохранить
            modal.querySelector('#modal-save-btn')?.addEventListener('click', () => {
                this.saveOrder();
            });
        }
    },

    // ============================================================
    // ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
    // ============================================================
    getStatusLabel(status) {
        const labels = {
            'Draft': 'Создан',
            'InProgress': 'В работе',
            'Ready': 'Готов',
            'Completed': 'Завершен'
        };
        return labels[status] || status;
    },

    getStatusColor(status) {
        const colors = {
            'Draft': '#6c757d',      // серый
            'InProgress': '#ffc107', // жёлтый
            'Ready': '#17a2b8',      // голубой
            'Completed': '#28a745'   // зелёный
        };
        return colors[status] || '#6c757d';
    }
};

// ================================================================
// ГЛОБАЛЬНЫЕ ФУНКЦИИ
// ================================================================

// Для вызова из HTML (onclick)
window.TransfersModule = TransfersModule;

// Экспорт для модулей
if (typeof module !== 'undefined' && module.exports) {
    module.exports = TransfersModule;
}