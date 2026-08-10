// ================================================================
// transfers.js - Интерфейс кладовщика для сборки заказов
// Версия: 2.0 (с поддержкой статусов)
// Страница: /transfers
// ================================================================

console.log('Transfers (Storekeeper) loaded');

(function () {
    'use strict';

    // ============================================================
    // СОСТОЯНИЕ
    // ============================================================
    const state = {
        orders: [],
        selectedOrderId: null,
        currentOrder: null,
        isLoading: false,
        filter: {
            status: '', // По умолчанию показываем заказы в работе
            fromWarehouse: '',
            toWarehouse: '',
            fromDate: '',
            toDate: ''
        },
        warehouses: [],
        materials: []
    };

    // ============================================================
    // ЭЛЕМЕНТЫ DOM
    // ============================================================
    const elements = {};

    function cacheElements() {
        elements.list = document.getElementById('transfers-list');
        elements.detailsPanel = document.getElementById('details-panel');
        elements.detailsEmpty = document.getElementById('details-empty');
        elements.detailsContent = document.getElementById('details-content');
        elements.detailsTitle = document.getElementById('details-title');
        elements.detailsStatus = document.getElementById('details-status');
        elements.detailsFrom = document.getElementById('details-from');
        elements.detailsTo = document.getElementById('details-to');
        elements.detailsPlanned = document.getElementById('details-planned');
        elements.detailsCreated = document.getElementById('details-created');
        elements.detailsStarted = document.getElementById('details-started');
        elements.itemsTbody = document.getElementById('items-tbody');
        elements.shipmentsTbody = document.getElementById('shipments-tbody');
        elements.progressBar = document.getElementById('progress-bar');
        elements.progressText = document.getElementById('progress-text');
        elements.btnStartOrder = document.getElementById('btn-start-order');
        elements.btnAddShipment = document.getElementById('btn-add-shipment');
        elements.btnRefresh = document.getElementById('refresh-btn');
        elements.filterStatus = document.getElementById('filter-status');
        elements.filterFromWarehouse = document.getElementById('filter-from-warehouse');
        elements.filterToWarehouse = document.getElementById('filter-to-warehouse');
        elements.filterFromDate = document.getElementById('filter-from-date');
        elements.filterToDate = document.getElementById('filter-to-date');
        elements.filterApply = document.getElementById('filter-apply-btn');

        // Модальное окно
        elements.modal = document.getElementById('modal-add-shipment');
        elements.modalClose = document.getElementById('modal-close');
        elements.modalCancel = document.getElementById('modal-cancel');
        elements.modalSave = document.getElementById('modal-save');
        elements.modalError = document.getElementById('modal-error');
        elements.inputMaterialCode = document.getElementById('input-material-code');
        elements.inputMaterialSuggestions = document.getElementById('input-material-suggestions');
        elements.inputQty = document.getElementById('input-qty');
        elements.inputWho = document.getElementById('input-who');
        elements.inputRemaining = document.getElementById('input-remaining');
    }

    // ============================================================
    // ИНИЦИАЛИЗАЦИЯ
    // ============================================================
    function init() {
        console.log('Transfers (Storekeeper) page initialized');

        // Кешируем элементы
        cacheElements();

        // Устанавливаем сегодняшнюю дату в фильтр
        const today = new Date();
        if (elements.filterFromDate) {
            elements.filterFromDate.value = today.toISOString().slice(0, 10);
        }

        // Загружаем справочники
        loadWarehouses();
        loadMaterials();

        // Загружаем заказы
        loadOrders();

        // Навешиваем события
        bindEvents();
    }

    // ============================================================
    // ЗАГРУЗКА СПРАВОЧНИКОВ
    // ============================================================
    async function loadWarehouses() {
        try {
            const response = await fetch('/api/warehouses?active=true');
            if (response.ok) {
                state.warehouses = await response.json();
                console.log('Загружено складов:', state.warehouses.length);

                // Заполняем фильтры
                [elements.filterFromWarehouse, elements.filterToWarehouse].forEach(select => {
                    if (!select) return;
                    // Очищаем, сохраняя первый option
                    while (select.options.length > 1) {
                        select.remove(1);
                    }
                    state.warehouses.forEach(w => {
                        const opt = document.createElement('option');
                        opt.value = w.warehouseId;
                        opt.textContent = `${w.code} - ${w.name}`;
                        select.appendChild(opt);
                    });
                });
            }
        } catch (error) {
            console.error('Ошибка загрузки складов:', error);
        }
    }

    async function loadMaterials() {
        try {
            const response = await fetch('/api/materials');
            if (response.ok) {
                state.materials = await response.json();
                console.log('Загружено материалов:', state.materials.length);
            }
        } catch (error) {
            console.error('Ошибка загрузки материалов:', error);
        }
    }

    // ============================================================
    // ЗАГРУЗКА ЗАКАЗОВ
    // ============================================================
    async function loadOrders() {
        state.isLoading = true;
        if (elements.list) {
            elements.list.innerHTML = '<div class="text-center text-muted py-4">⏳ Загрузка...</div>';
        }

        try {
            const params = new URLSearchParams();
            if (state.filter.status) params.append('status', state.filter.status);
            if (state.filter.fromWarehouse) params.append('fromWarehouse', state.filter.fromWarehouse);
            if (state.filter.toWarehouse) params.append('toWarehouse', state.filter.toWarehouse);
            if (state.filter.fromDate) params.append('fromDate', state.filter.fromDate);
            if (state.filter.toDate) params.append('toDate', state.filter.toDate);

            const url = `/api/transfer-orders?${params.toString()}`;
            const response = await fetch(url);
            if (!response.ok) throw new Error('Ошибка загрузки заказов');

            state.orders = await response.json();
            console.log('Загружено заказов:', state.orders.length);

            renderOrdersList();

            // Если есть заказы и нет выбранного, выбираем первый
            if (state.orders.length > 0 && !state.selectedOrderId) {
                await selectOrder(state.orders[0].transferOrderId);
            } else if (state.orders.length === 0) {
                showEmptyDetails();
            }

        } catch (error) {
            console.error('Ошибка:', error);
            if (elements.list) {
                elements.list.innerHTML = `<div class="text-center text-danger py-4">❌ ${error.message}</div>`;
            }
        }

        state.isLoading = false;
    }

    // ============================================================
    // ОТОБРАЖЕНИЕ СПИСКА ЗАКАЗОВ
    // ============================================================
    function renderOrdersList() {
        if (!elements.list) return;

        if (state.orders.length === 0) {
            elements.list.innerHTML = `
                <div class="text-center text-muted py-5">
                    <div style="font-size: 48px; margin-bottom: 16px;">📭</div>
                    <h5>Нет заказов</h5>
                    <p class="small">Попробуйте изменить фильтры</p>
                </div>
            `;
            return;
        }

        elements.list.innerHTML = state.orders.map(o => {
            const statusColor = getStatusColor(o.status);
            const statusLabel = getStatusLabel(o.status);
            const isSelected = o.transferOrderId === state.selectedOrderId;
            const canStart = o.status === 'Draft';

            return `
                <div class="transfer-card ${isSelected ? 'selected' : ''} card mb-2"
                     data-id="${o.transferOrderId}"
                     onclick="window._selectOrder(${o.transferOrderId})">
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
                        ${canStart ? `
                            <button class="btn btn-success btn-sm w-100 mt-2" 
                                    onclick="event.stopPropagation(); window._startOrder(${o.transferOrderId})">
                                ▶️ Начать сборку
                            </button>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');

        // Глобальные функции для onclick
        window._selectOrder = selectOrder;
        window._startOrder = startOrder;
    }

    // ============================================================
    // ВЫБОР ЗАКАЗА
    // ============================================================
    async function selectOrder(orderId) {
        if (!orderId) return;
        console.log('Выбор заказа:', orderId);

        state.selectedOrderId = orderId;

        // Обновляем выделение в списке
        document.querySelectorAll('.transfer-card').forEach(el => {
            el.classList.toggle('selected', parseInt(el.dataset.id) === orderId);
        });

        // Загружаем детали
        await loadOrderDetails(orderId);
    }

    async function loadOrderDetails(orderId) {
        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`);
            if (!response.ok) throw new Error('Ошибка загрузки деталей');

            state.currentOrder = await response.json();
            renderDetails(state.currentOrder);

        } catch (error) {
            console.error('Ошибка загрузки деталей:', error);
            showEmptyDetails();
        }
    }

    // ============================================================
    // ОТОБРАЖЕНИЕ ДЕТАЛЕЙ ЗАКАЗА
    // ============================================================
    function renderDetails(order) {
        if (!elements.detailsContent) return;

        elements.detailsEmpty.style.display = 'none';
        elements.detailsContent.style.display = 'block';

        // Заголовок
        elements.detailsTitle.textContent = `Заказ №${order.number}`;
        elements.detailsStatus.textContent = getStatusLabel(order.status);
        elements.detailsStatus.style.background = getStatusColor(order.status);

        elements.detailsFrom.textContent = `${order.fromWarehouseCode} - ${order.fromWarehouseName}`;
        elements.detailsTo.textContent = `${order.toWarehouseCode} - ${order.toWarehouseName}`;
        elements.detailsPlanned.textContent = order.plannedDate || 'не указана';
        elements.detailsCreated.textContent = order.date || '';

        // Кто начал сборку
        const startedText = order.startedBy ? `${order.startedBy} (${order.startedAt || ''})` : 'не начат';
        if (elements.detailsStarted) {
            elements.detailsStarted.textContent = startedText;
        }

        // Прогресс
        const total = order.details.reduce((sum, d) => sum + d.quantity, 0);
        const shipped = order.details.reduce((sum, d) => sum + d.shippedQuantity, 0);
        const progress = total > 0 ? Math.round((shipped / total) * 100) : 0;

        if (elements.progressBar) {
            elements.progressBar.style.width = `${progress}%`;
            elements.progressBar.style.background = getStatusColor(order.status);
        }
        if (elements.progressText) {
            elements.progressText.textContent = `${progress}% (${shipped}/${total} шт.)`;
        }

        // Кнопки
        if (elements.btnStartOrder) {
            elements.btnStartOrder.style.display = order.canStart ? 'inline-block' : 'none';
        }
        if (elements.btnAddShipment) {
            elements.btnAddShipment.style.display = order.canAddShipment ? 'inline-block' : 'none';
        }

        // Материалы
        renderItems(order.details || []);

        // Отгрузки
        renderShipments(order.shipments || []);
    }

    function renderItems(items) {
        if (!elements.itemsTbody) return;

        if (!items || items.length === 0) {
            elements.itemsTbody.innerHTML = `<tr><td colspan="7" class="text-muted text-center">Нет материалов</td></tr>`;
            return;
        }

        elements.itemsTbody.innerHTML = items.map(item => {
            const remaining = item.remaining || 0;
            const isDone = remaining <= 0;
            const isPartial = item.shippedQuantity > 0 && !isDone;

            let statusText = '⏳ Ожидает';
            let statusClass = 'text-muted';
            if (isDone) {
                statusText = '✅ Готово';
                statusClass = 'text-success';
            } else if (isPartial) {
                statusText = '🔄 Частично';
                statusClass = 'text-warning';
            }

            return `
                <tr>
                    <td><strong>${escapeHtml(item.materialCode)}</strong></td>
                    <td>${escapeHtml(item.description || '')}</td>
                    <td class="text-center">${item.quantity}</td>
                    <td class="text-center">${item.shippedQuantity}</td>
                    <td class="text-center ${remaining <= 0 ? 'text-success' : 'text-warning'}">${remaining}</td>
                    <td class="text-center ${statusClass}">${statusText}</td>
                    <td class="text-center">
                        <div class="progress" style="height: 6px; width: 80px; margin: 0 auto;">
                            <div class="progress-bar" role="progressbar" 
                                 style="width: ${item.quantity > 0 ? Math.round((item.shippedQuantity / item.quantity) * 100) : 0}%; 
                                        background: ${isDone ? '#28a745' : isPartial ? '#ffc107' : '#6c757d'};"></div>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    }

    function renderShipments(shipments) {
        if (!elements.shipmentsTbody) return;

        if (!shipments || shipments.length === 0) {
            elements.shipmentsTbody.innerHTML = `<tr><td colspan="4" class="text-muted text-center">Нет отгрузок</td></tr>`;
            return;
        }

        elements.shipmentsTbody.innerHTML = shipments.map(s => `
            <tr>
                <td>${s.createdAt || ''}</td>
                <td>${escapeHtml(s.materialCode)}</td>
                <td class="text-center">${s.quantity}</td>
                <td>${escapeHtml(s.createdBy || '')}</td>
            </tr>
        `).join('');
    }

    function showEmptyDetails() {
        if (elements.detailsEmpty) elements.detailsEmpty.style.display = 'flex';
        if (elements.detailsContent) elements.detailsContent.style.display = 'none';
    }

    // ============================================================
    // УПРАВЛЕНИЕ СТАТУСАМИ
    // ============================================================
    async function startOrder(orderId) {
        if (!confirm('Начать сборку заказа?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${orderId}/start`, {
                method: 'POST'
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка начала сборки');
            }

            // Обновляем данные
            await loadOrders();
            if (state.orders.length > 0) {
                await selectOrder(state.orders[0].transferOrderId);
            }

        } catch (error) {
            console.error('Ошибка начала сборки:', error);
            alert('Ошибка: ' + error.message);
        }
    }

    // ============================================================
    // МОДАЛЬНОЕ ОКНО ДОБАВЛЕНИЯ ОТГРУЗКИ
    // ============================================================
    function openModal() {
        if (!state.currentOrder || state.currentOrder.status !== 'InProgress') {
            alert('Заказ не в статусе "В работе"');
            return;
        }

        elements.modalError.style.display = 'none';
        elements.inputMaterialCode.value = '';
        elements.inputQty.value = 1;
        elements.inputWho.value = '';
        elements.inputRemaining.textContent = '';

        // Очищаем подсказки
        if (elements.inputMaterialSuggestions) {
            elements.inputMaterialSuggestions.innerHTML = '';
            elements.inputMaterialSuggestions.style.display = 'none';
        }

        elements.modal.style.display = 'flex';
        elements.modalOpen = true;

        // Фокус на поле кода материала
        setTimeout(() => {
            elements.inputMaterialCode.focus();
        }, 200);
    }

    function closeModal() {
        elements.modal.style.display = 'none';
        elements.modalError.style.display = 'none';
    }

    // ============================================================
    // ПОИСК МАТЕРИАЛА С ПОДСКАЗКАМИ
    // ============================================================
    function showMaterialSuggestions(query) {
        const suggestionsEl = elements.inputMaterialSuggestions;
        if (!suggestionsEl) return;

        if (!query || query.length < 2) {
            suggestionsEl.style.display = 'none';
            return;
        }

        const orderMaterials = state.currentOrder?.details || [];
        const matched = orderMaterials.filter(d =>
            d.materialCode.toLowerCase().includes(query.toLowerCase()) ||
            (d.description && d.description.toLowerCase().includes(query.toLowerCase()))
        );

        if (matched.length === 0) {
            suggestionsEl.innerHTML = '<div class="suggestion-item text-muted">Материал не найден в заказе</div>';
            suggestionsEl.style.display = 'block';
            return;
        }

        suggestionsEl.innerHTML = matched.map(d => `
            <div class="suggestion-item" data-material-id="${d.materialId}" data-material-code="${d.materialCode}" data-remaining="${d.remaining || 0}">
                <strong>${escapeHtml(d.materialCode)}</strong>
                <span class="text-muted">${escapeHtml(d.description || '')}</span>
                <span class="badge bg-secondary">осталось ${d.remaining || 0} шт.</span>
            </div>
        `).join('');

        suggestionsEl.style.display = 'block';

        // Клик по подсказке
        suggestionsEl.querySelectorAll('.suggestion-item').forEach(el => {
            el.addEventListener('click', () => {
                const code = el.dataset.materialCode;
                const remaining = parseInt(el.dataset.remaining);
                elements.inputMaterialCode.value = code;
                elements.inputRemaining.textContent = `Остаток: ${remaining} шт.`;
                suggestionsEl.style.display = 'none';
                elements.inputQty.focus();
            });
        });
    }

    // ============================================================
    // СОХРАНЕНИЕ ОТГРУЗКИ
    // ============================================================
    async function onSaveShipment() {
        elements.modalError.style.display = 'none';

        const materialCode = elements.inputMaterialCode.value.trim();
        const qty = parseInt(elements.inputQty.value);
        const who = elements.inputWho.value.trim() || 'Неизвестно';

        if (!materialCode) {
            showModalError('Введите код материала');
            return;
        }

        // Проверяем, что материал есть в заказе
        const orderItem = state.currentOrder?.details?.find(d =>
            d.materialCode.toLowerCase() === materialCode.toLowerCase()
        );

        if (!orderItem) {
            showModalError('Материал не найден в заказе');
            return;
        }

        if (!qty || qty <= 0) {
            showModalError('Количество должно быть положительным');
            return;
        }

        const remaining = orderItem.remaining || 0;
        if (qty > remaining) {
            showModalError(`Нельзя отгрузить больше ${remaining} шт. (остаток)`);
            return;
        }

        const payload = {
            materialCode: materialCode,
            quantity: qty,
            createdBy: who
        };

        try {
            const response = await fetch(`/api/transfer-shipments?orderId=${state.currentOrder.transferOrderId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка сохранения');
            }

            // Успешно - обновляем данные
            closeModal();
            await loadOrders();
            if (state.selectedOrderId) {
                await selectOrder(state.selectedOrderId);
            }

        } catch (error) {
            console.error('Ошибка:', error);
            showModalError(error.message || 'Ошибка при сохранении');
        }
    }

    function showModalError(message) {
        elements.modalError.textContent = message;
        elements.modalError.style.display = 'block';
    }

    // ============================================================
    // СОБЫТИЯ
    // ============================================================
    function bindEvents() {
        // Обновить
        elements.btnRefresh?.addEventListener('click', loadOrders);

        // Применить фильтры
        elements.filterApply?.addEventListener('click', () => {
            state.filter.status = elements.filterStatus?.value || '';
            state.filter.fromWarehouse = elements.filterFromWarehouse?.value || '';
            state.filter.toWarehouse = elements.filterToWarehouse?.value || '';
            state.filter.fromDate = elements.filterFromDate?.value || '';
            state.filter.toDate = elements.filterToDate?.value || '';
            loadOrders();
        });

        // Начать сборку (кнопка в деталях)
        elements.btnStartOrder?.addEventListener('click', () => {
            if (state.selectedOrderId) {
                startOrder(state.selectedOrderId);
            }
        });

        // Добавить отгрузку
        elements.btnAddShipment?.addEventListener('click', openModal);

        // Модальное окно
        elements.modalClose?.addEventListener('click', closeModal);
        elements.modalCancel?.addEventListener('click', closeModal);
        elements.modalSave?.addEventListener('click', onSaveShipment);

        // Закрытие по клику вне модалки
        elements.modal?.addEventListener('click', (e) => {
            if (e.target === elements.modal) closeModal();
        });

        // Поиск материала с подсказками
        elements.inputMaterialCode?.addEventListener('input', (e) => {
            showMaterialSuggestions(e.target.value);
        });

        elements.inputMaterialCode?.addEventListener('blur', () => {
            // Скрываем подсказки с задержкой
            setTimeout(() => {
                if (elements.inputMaterialSuggestions) {
                    elements.inputMaterialSuggestions.style.display = 'none';
                }
            }, 200);
        });

        elements.inputMaterialCode?.addEventListener('focus', () => {
            if (elements.inputMaterialCode.value.length >= 2) {
                showMaterialSuggestions(elements.inputMaterialCode.value);
            }
        });

        // Enter в полях
        elements.inputQty?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') onSaveShipment();
        });
        elements.inputWho?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') onSaveShipment();
        });
    }

    // ============================================================
    // ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
    // ============================================================
    function getStatusLabel(status) {
        const labels = {
            'Draft': 'Создан',
            'InProgress': 'В работе',
            'Ready': 'Готов',
            'Completed': 'Завершен'
        };
        return labels[status] || status;
    }

    function getStatusColor(status) {
        const colors = {
            'Draft': '#6c757d',
            'InProgress': '#ffc107',
            'Ready': '#17a2b8',
            'Completed': '#28a745'
        };
        return colors[status] || '#6c757d';
    }

    function escapeHtml(str) {
        if (str == null) return '';
        return String(str).replace(/[&<>"']/g, function (c) {
            return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c];
        });
    }

    // ============================================================
    // ЗАПУСК
    // ============================================================
    document.addEventListener('DOMContentLoaded', init);

})();