// transfers.js - Управление заказами на перемещение (новая структура)

console.log('transfers.js loaded');

(function () {
    'use strict';

    // ============================================
    // ЭЛЕМЕНТЫ DOM
    // ============================================
    const listEl = document.getElementById('transfers-list');
    const detailsPanel = document.getElementById('details-panel');
    const detailsEmpty = document.getElementById('details-empty');
    const detailsContent = document.getElementById('details-content');
    const detailsTitle = document.getElementById('details-title');
    const detailsStatus = document.getElementById('details-status');
    const detailsFrom = document.getElementById('details-from');
    const detailsTo = document.getElementById('details-to');
    const detailsPlanned = document.getElementById('details-planned');
    const detailsCreated = document.getElementById('details-created');
    const itemsTbody = document.getElementById('items-tbody');
    const shipmentsTbody = document.getElementById('shipments-tbody');
    const btnAddShipment = document.getElementById('btn-add-shipment');
    const btnCompleteOrder = document.getElementById('btn-complete-order');
    const btnCreate = document.getElementById('create-transfer-btn');
    const btnRefresh = document.getElementById('refresh-btn');
    const filterDate = document.getElementById('filter-date');
    const filterWarehouse = document.getElementById('filter-warehouse');

    // Модальное окно
    const modalEl = document.getElementById('modal-add-shipment');
    const modalClose = document.getElementById('modal-close');
    const modalCancel = document.getElementById('modal-cancel');
    const modalSave = document.getElementById('modal-save');
    const modalError = document.getElementById('modal-error');
    const inputMaterialSelect = document.getElementById('input-material-select');
    const inputMaterialCode = document.getElementById('input-material-code');
    const inputQty = document.getElementById('input-qty');
    const inputWho = document.getElementById('input-who');

    // ============================================
    // СОСТОЯНИЕ
    // ============================================
    let orders = [];
    let selectedOrder = null;
    let currentItems = [];
    let currentShipments = [];
    let warehouses = [];
    let modalOpen = false;

    // ============================================
    // ИНИЦИАЛИЗАЦИЯ
    // ============================================
    function init() {
        console.log('Transfers page initialized');

        // Устанавливаем сегодняшнюю дату в фильтр
        const today = new Date();
        filterDate.value = today.toISOString().slice(0, 10);

        // Загружаем склады для фильтра
        loadWarehouses();

        // Загружаем заказы
        loadOrders();

        // Навешиваем события
        btnCreate.addEventListener('click', onCreateOrder);
        btnRefresh.addEventListener('click', () => loadOrders());
        btnAddShipment.addEventListener('click', openModal);
        btnCompleteOrder.addEventListener('click', onCompleteOrder);

        filterDate.addEventListener('change', loadOrders);
        filterWarehouse.addEventListener('change', loadOrders);

        // Модальное окно
        modalClose.addEventListener('click', closeModal);
        modalCancel.addEventListener('click', closeModal);
        modalSave.addEventListener('click', onSaveShipment);

        // Закрытие по клику вне модалки
        modalEl.addEventListener('click', (e) => {
            if (e.target === modalEl) closeModal();
        });

        // Переносим фокус при смене режима ввода материала
        inputMaterialSelect.addEventListener('change', () => {
            if (inputMaterialSelect.style.display !== 'none') {
                inputMaterialSelect.focus();
            }
        });

        // Обработка Enter в полях
        inputQty.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') onSaveShipment();
        });
        inputWho.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') onSaveShipment();
        });
    }

    // ============================================
    // ЗАГРУЗКА ДАННЫХ
    // ============================================
    async function loadWarehouses() {
        try {
            const response = await fetch('/api/warehouses?active=true');
            if (response.ok) {
                warehouses = await response.json();
                console.log('Загружено складов:', warehouses.length);

                // Заполняем фильтр
                warehouses.forEach(w => {
                    const opt = document.createElement('option');
                    opt.value = w.warehouseId;
                    opt.textContent = `${w.code} - ${w.name}`;
                    filterWarehouse.appendChild(opt);
                });
            }
        } catch (error) {
            console.error('Ошибка загрузки складов:', error);
        }
    }

    async function loadOrders() {
        console.log('Loading orders...');
        listEl.innerHTML = '<div class="loading">Загрузка...</div>';
        selectedOrder = null;
        showEmptyDetails();

        const date = filterDate.value;
        const warehouse = filterWarehouse.value;

        let url = '/api/transfer-orders?completed=false';
        if (date) url += `&plannedDate=${date}`;
        if (warehouse) url += `&fromWarehouse=${warehouse}`;

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error('Ошибка загрузки заказов');
            orders = await response.json();
            console.log('Загружено заказов:', orders.length);

            renderOrdersList();

            // Если есть заказы, выбираем первый
            if (orders.length > 0) {
                selectOrder(orders[0].transferOrderId);
            }
        } catch (error) {
            console.error('Ошибка:', error);
            listEl.innerHTML = `<div class="error-state">❌ ${error.message}</div>`;
        }
    }

    // ============================================
    // ОТОБРАЖЕНИЕ СПИСКА
    // ============================================
    function renderOrdersList() {
        if (orders.length === 0) {
            listEl.innerHTML = '<div class="empty-state">Нет заказов на перемещение</div>';
            return;
        }

        listEl.innerHTML = orders.map(o => {
            const status = o.completed ? '✅ Завершён' : '🔄 В работе';
            const statusClass = o.completed ? 'completed' : 'active';
            const totalItems = o.details ? o.details.length : 0;
            const totalQty = o.details ? o.details.reduce((sum, d) => sum + d.quantity, 0) : 0;

            return `
                <div class="transfer-card ${statusClass} ${selectedOrder && selectedOrder.transferOrderId === o.transferOrderId ? 'selected' : ''}" 
                     data-id="${o.transferOrderId}"
                     onclick="window._selectOrder(${o.transferOrderId})">
                    <div class="transfer-header">
                        <span class="transfer-number">№${o.number}</span>
                        <span class="transfer-status" style="background: ${o.completed ? '#28a745' : '#ffc107'}">
                            ${status}
                        </span>
                    </div>
                    <div class="transfer-body">
                        <div class="transfer-route">
                            <span>🏢 ${o.fromWarehouseCode}</span>
                            <span class="transfer-arrow">➜</span>
                            <span>🏢 ${o.toWarehouseCode}</span>
                        </div>
                        <div class="transfer-summary">
                            <span>📦 ${totalItems} материалов</span>
                            <span>📊 ${totalQty} шт.</span>
                        </div>
                        <div class="transfer-date">
                            📅 ${o.plannedDate || o.date}
                        </div>
                    </div>
                </div>
            `;
        }).join('');

        // Делаем функцию выбора глобальной для onclick
        window._selectOrder = selectOrder;
    }

    // ============================================
    // ВЫБОР ЗАКАЗА
    // ============================================
    async function selectOrder(orderId) {
        if (!orderId) return;
        console.log('Selecting order:', orderId);

        // Находим заказ в списке
        const order = orders.find(o => o.transferOrderId === orderId);
        if (!order) {
            console.error('Order not found:', orderId);
            return;
        }

        selectedOrder = order;

        // Обновляем активный класс в списке
        document.querySelectorAll('.transfer-card').forEach(el => {
            el.classList.toggle('selected', parseInt(el.dataset.id) === orderId);
        });

        // Загружаем детали
        await loadOrderDetails(orderId);
    }

    // ============================================
    // ЗАГРУЗКА ДЕТАЛЕЙ ЗАКАЗА
    // ============================================
    async function loadOrderDetails(orderId) {
        try {
            const response = await fetch(`/api/transfer-orders/${orderId}`);
            if (!response.ok) throw new Error('Ошибка загрузки деталей');
            const order = await response.json();

            // Обновляем текущие данные
            selectedOrder = order;
            currentItems = order.details || [];

            // Получаем историю отгрузок (если есть эндпоинт)
            await loadShipmentHistory(orderId);

            renderDetails(order);

        } catch (error) {
            console.error('Ошибка загрузки деталей:', error);
            showEmptyDetails();
        }
    }

    async function loadShipmentHistory(orderId) {
        try {
            // Пробуем получить историю отгрузок
            const response = await fetch(`/api/transfer-orders/${orderId}/shipments`);
            if (response.ok) {
                currentShipments = await response.json();
            } else {
                currentShipments = [];
            }
        } catch (error) {
            console.warn('Не удалось загрузить историю отгрузок:', error);
            currentShipments = [];
        }
    }

    // ============================================
    // ОТОБРАЖЕНИЕ ДЕТАЛЕЙ
    // ============================================
    function renderDetails(order) {
        detailsEmpty.style.display = 'none';
        detailsContent.style.display = 'block';

        const status = order.completed ? '✅ Завершён' : '🔄 В работе';
        const statusColor = order.completed ? '#28a745' : '#ffc107';

        detailsTitle.textContent = `Заказ №${order.number}`;
        detailsStatus.textContent = status;
        detailsStatus.style.background = statusColor;
        detailsFrom.textContent = `${order.fromWarehouseCode} - ${order.fromWarehouseName}`;
        detailsTo.textContent = `${order.toWarehouseCode} - ${order.toWarehouseName}`;
        detailsPlanned.textContent = order.plannedDate || 'не указана';
        detailsCreated.textContent = order.date || '';

        // Рендерим материалы
        renderItems(order.details || []);

        // Рендерим историю отгрузок
        renderShipments(currentShipments || []);

        // Обновляем кнопки
        const allCompleted = (order.details || []).every(d => d.shippedQuantity >= d.quantity);
        btnCompleteOrder.disabled = order.completed || !allCompleted;
        btnCompleteOrder.textContent = order.completed ? '✅ Завершён' : '✅ Завершить заказ';
        btnAddShipment.disabled = order.completed;
    }

    function renderItems(items) {
        if (!items || items.length === 0) {
            itemsTbody.innerHTML = `<tr><td colspan="6" class="text-muted">Нет материалов</td></tr>`;
            return;
        }

        itemsTbody.innerHTML = items.map(item => {
            const shipped = item.shippedQuantity || 0;
            const remaining = item.quantity - shipped;
            let status = 'В работе';
            let statusClass = '';

            if (shipped >= item.quantity) {
                status = '✅ Готово';
                statusClass = 'status-done';
            } else if (shipped > 0) {
                status = '🔄 Частично';
                statusClass = 'status-partial';
            } else {
                status = '⏳ Ожидает';
                statusClass = 'status-pending';
            }

            return `
                <tr>
                    <td><strong>${escapeHtml(item.materialCode)}</strong></td>
                    <td>${escapeHtml(item.description || '')}</td>
                    <td>${item.quantity}</td>
                    <td>${shipped}</td>
                    <td>${remaining}</td>
                    <td><span class="${statusClass}">${status}</span></td>
                </tr>
            `;
        }).join('');

        // Обновляем селект в модалке
        updateMaterialSelect(items);
    }

    function renderShipments(shipments) {
        if (!shipments || shipments.length === 0) {
            shipmentsTbody.innerHTML = `<tr><td colspan="4" class="text-muted">Нет отгрузок</td></tr>`;
            return;
        }

        shipmentsTbody.innerHTML = shipments.map(s => `
            <tr>
                <td>${formatDate(s.createdAt || s.date)}</td>
                <td>${escapeHtml(s.materialCode || '')}</td>
                <td>${s.quantity || 0} шт.</td>
                <td>${escapeHtml(s.createdBy || s.who || '')}</td>
            </tr>
        `).join('');
    }

    // ============================================
    // МОДАЛЬНОЕ ОКНО
    // ============================================
    function openModal() {
        if (!selectedOrder || selectedOrder.completed) {
            alert('Заказ завершён, нельзя добавить отгрузку');
            return;
        }

        modalError.style.display = 'none';
        inputQty.value = 1;
        inputWho.value = '';

        // Заполняем выбор материалов
        updateMaterialSelect(selectedOrder.details || []);

        // Определяем режим ввода
        const items = selectedOrder.details || [];
        if (items.length > 1) {
            inputMaterialSelect.style.display = 'block';
            inputMaterialCode.style.display = 'none';
            if (inputMaterialSelect.options.length > 0) {
                inputMaterialSelect.selectedIndex = 0;
            }
        } else if (items.length === 1) {
            inputMaterialSelect.style.display = 'none';
            inputMaterialCode.style.display = 'block';
            inputMaterialCode.value = items[0].materialCode || '';
        } else {
            inputMaterialSelect.style.display = 'none';
            inputMaterialCode.style.display = 'block';
            inputMaterialCode.value = '';
        }

        modalEl.style.display = 'flex';
        modalOpen = true;

        // Фокус на первое поле
        setTimeout(() => {
            if (inputMaterialSelect.style.display !== 'none') {
                inputMaterialSelect.focus();
            } else {
                inputMaterialCode.focus();
            }
        }, 200);
    }

    function closeModal() {
        modalEl.style.display = 'none';
        modalOpen = false;
        modalError.style.display = 'none';
    }

    function updateMaterialSelect(items) {
        inputMaterialSelect.innerHTML = '';
        items.forEach(item => {
            const opt = document.createElement('option');
            opt.value = item.materialId;
            const shipped = item.shippedQuantity || 0;
            const remaining = item.quantity - shipped;
            opt.textContent = `${item.materialCode} — осталось ${remaining} шт. (план ${item.quantity})`;
            opt.disabled = remaining <= 0;
            inputMaterialSelect.appendChild(opt);
        });

        // Если есть только один материал, скрываем селект
        if (items.length <= 1) {
            inputMaterialSelect.style.display = 'none';
            inputMaterialCode.style.display = 'block';
            if (items.length === 1) {
                inputMaterialCode.value = items[0].materialCode || '';
            }
        } else {
            inputMaterialSelect.style.display = 'block';
            inputMaterialCode.style.display = 'none';
        }
    }

    // ============================================
    // СОХРАНЕНИЕ ОТГРУЗКИ
    // ============================================
    async function onSaveShipment() {
        modalError.style.display = 'none';

        const qty = parseInt(inputQty.value, 10);
        const who = inputWho.value.trim() || 'Неизвестно';

        if (!selectedOrder || !selectedOrder.transferOrderId) {
            showModalError('Не выбран заказ');
            return;
        }

        if (!qty || qty <= 0) {
            showModalError('Количество должно быть положительным');
            return;
        }

        // Определяем материал
        let materialId = null;
        let materialCode = null;

        if (inputMaterialSelect.style.display !== 'none') {
            materialId = parseInt(inputMaterialSelect.value, 10);
        } else {
            materialCode = inputMaterialCode.value.trim();
            // Ищем материал по коду
            const item = (selectedOrder.details || []).find(d =>
                d.materialCode && d.materialCode.toLowerCase() === materialCode.toLowerCase()
            );
            if (item) {
                materialId = item.materialId;
            }
        }

        if (!materialId) {
            showModalError('Материал не найден');
            return;
        }

        // Проверяем, что не превышаем остаток
        const item = (selectedOrder.details || []).find(d => d.materialId === materialId);
        if (item) {
            const shipped = item.shippedQuantity || 0;
            const remaining = item.quantity - shipped;
            if (qty > remaining) {
                showModalError(`Нельзя отгрузить больше ${remaining} шт. (остаток)`);
                return;
            }
        }

        const payload = {
            transferOrderId: selectedOrder.transferOrderId,
            materialId: materialId,
            quantity: qty,
            createdBy: who
        };

        try {
            const response = await fetch('/api/transfer-orders/shipments', {
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
            if (selectedOrder) {
                await selectOrder(selectedOrder.transferOrderId);
            }
            alert('Отгрузка добавлена');

        } catch (error) {
            console.error('Ошибка:', error);
            showModalError(error.message || 'Ошибка при сохранении');
        }
    }

    function showModalError(message) {
        modalError.textContent = message;
        modalError.style.display = 'block';
    }

    // ============================================
    // ЗАВЕРШЕНИЕ ЗАКАЗА
    // ============================================
    async function onCompleteOrder() {
        if (!selectedOrder) return;
        if (selectedOrder.completed) {
            alert('Заказ уже завершён');
            return;
        }

        // Проверяем, все ли материалы отгружены
        const allCompleted = (selectedOrder.details || []).every(d =>
            (d.shippedQuantity || 0) >= d.quantity
        );

        if (!allCompleted) {
            alert('Не все материалы отгружены полностью');
            return;
        }

        if (!confirm('Завершить заказ?')) return;

        try {
            const response = await fetch(`/api/transfer-orders/${selectedOrder.transferOrderId}/complete`, {
                method: 'PUT'
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.message || 'Ошибка завершения');
            }

            await loadOrders();
            if (selectedOrder) {
                await selectOrder(selectedOrder.transferOrderId);
            }
            alert('Заказ завершён');

        } catch (error) {
            console.error('Ошибка:', error);
            alert('Ошибка завершения заказа: ' + error.message);
        }
    }

    // ============================================
    // СОЗДАНИЕ ЗАКАЗА
    // ============================================
    function onCreateOrder() {
        // Перенаправляем на страницу создания через модуль
        window.location.href = '/transfers/new';
    }

    // ============================================
    // ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
    // ============================================
    function showEmptyDetails() {
        detailsEmpty.style.display = 'flex';
        detailsContent.style.display = 'none';
    }

    function formatDate(s) {
        if (!s) return '';
        const d = new Date(s);
        if (isNaN(d.getTime())) return String(s);
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    function pad(n) { return n < 10 ? '0' + n : n; }

    function escapeHtml(str) {
        if (str == null) return '';
        return String(str).replace(/[&<>"']/g, function (c) {
            return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c];
        });
    }

    // ============================================
    // ЗАПУСК
    // ============================================
    document.addEventListener('DOMContentLoaded', init);

})();