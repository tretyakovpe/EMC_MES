// ========== МОДУЛЬ ОТГРУЗОК ==========

const ShipmentsModule = {
    currentShipment: null,
    shipmentItems: [],
    shipments: [],
    viewMode: 'table', // 'cards' или 'table'
    sortField: null,
    sortDirection: 'asc',
    filterStatus: 'all',

    async render(container) {
        // Загружаем сохранённый режим
        this.viewMode = localStorage.getItem('shipmentsViewMode') || 'cards';
        this.sortField = localStorage.getItem('shipmentsSortField') || null;
        this.sortDirection = localStorage.getItem('shipmentsSortDirection') || 'asc';
        this.filterStatus = localStorage.getItem('shipmentsFilterStatus') || 'all';

        container.innerHTML = `
            <div class="logistics-container">
                <div class="shipments-layout">
                    <!-- Левая панель: список отгрузок -->
                    <div class="shipments-list-panel">
                        <div class="panel-header">
                            <h3>🚛 Отгрузки</h3>
                            
                            <!-- ПЕРЕКЛЮЧАТЕЛЬ ВИДА -->
                            <div class="view-toggle">
                                <button class="view-toggle-btn ${this.viewMode === 'cards' ? 'active' : ''}" 
                                        data-view="cards" 
                                        title="Вид карточками">
                                    🃏 Карточки
                                </button>
                                <button class="view-toggle-btn ${this.viewMode === 'table' ? 'active' : ''}" 
                                        data-view="table" 
                                        title="Табличный вид">
                                    📊 Таблица
                                </button>
                            </div>
                            
                            <button id="create-shipment-btn" class="btn-primary">➕ Новая отгрузка</button>
                            <button id="clipboard-shipment-btn" class="btn-secondary">📋 Из буфера</button>
                        </div>
                        
                        <!-- Фильтры для таблицы -->
                        <div class="table-filters" id="table-filters" style="${this.viewMode === 'table' ? 'display:flex' : 'display:none'}">
                            <select id="status-filter" class="form-select">
                                <option value="all" ${this.filterStatus === 'all' ? 'selected' : ''}>Все статусы</option>
                                <option value="active" ${this.filterStatus === 'active' ? 'selected' : ''}>🔄 В работе</option>
                                <option value="completed" ${this.filterStatus === 'completed' ? 'selected' : ''}>📦 Готова</option>
                                <option value="done" ${this.filterStatus === 'done' ? 'selected' : ''}>🚚 Отгружена</option>
                            </select>
                            <input type="text" id="table-search" placeholder="🔍 Поиск по номеру..." class="form-input" style="width:200px;">
                            <button id="clear-filters-btn" class="btn-secondary">Очистить</button>
                        </div>
                        
                        <!-- КАРТОЧКИ -->
                        <div class="shipments-grid ${this.viewMode === 'cards' ? 'visible' : 'hidden'}" id="shipments-grid">
                            <div class="loading">Загрузка...</div>
                        </div>
                        
                        <!-- ТАБЛИЦА -->
                        <div class="shipments-table-wrapper ${this.viewMode === 'table' ? 'visible' : ''}" id="shipments-table-wrapper">
                            <table class="shipments-table" id="shipments-table">
                                <thead>
                                    <tr>
                                        <th data-sort="shipmentId">ID <span class="sort-icon">↕</span></th>
                                        <th data-sort="number">№ накладной <span class="sort-icon">↕</span></th>
                                        <th data-sort="date">Дата <span class="sort-icon">↕</span></th>
                                        <th data-sort="status">Статус <span class="sort-icon">↕</span></th>
                                        <th data-sort="progress">Прогресс <span class="sort-icon">↕</span></th>
                                        <th data-sort="totalItems">Позиций</th>
                                        <th data-sort="totalBoxes">Коробок</th>
                                        <th>Действия</th>
                                    </tr>
                                </thead>
                                <tbody id="shipments-table-body">
                                    <tr><td colspan="8" class="empty-table">Загрузка...</td></tr>
                                </tbody>
                            </table>
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
        this.applyStoredSort();
        this.applyFilter();
    },

    async loadShipments() {
        try {
            const shipments = await API.getShipments({ done: false });
            this.shipments = shipments;
            this.renderView();
        } catch (error) {
            console.error('Ошибка загрузки отгрузок:', error);
            const grid = document.getElementById('shipments-grid');
            const tableBody = document.getElementById('shipments-table-body');
            const errorMsg = '<div class="empty-state">❌ Ошибка загрузки отгрузок</div>';
            if (grid) grid.innerHTML = errorMsg;
            if (tableBody) tableBody.innerHTML = `<tr><td colspan="8" class="empty-table">❌ Ошибка загрузки</td></tr>`;
        }
    },

    renderView() {
        if (this.viewMode === 'cards') {
            this.renderCardsView();
        } else {
            this.renderTableView();
        }
    },

    renderCardsView() {
        const container = document.getElementById('shipments-grid');
        if (!container) return;

        // Показываем карточки, скрываем таблицу
        document.getElementById('shipments-grid').className = 'shipments-grid visible';
        document.getElementById('shipments-table-wrapper').className = 'shipments-table-wrapper';
        document.getElementById('table-filters').style.display = 'none';

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

    renderTableView() {
        const tableBody = document.getElementById('shipments-table-body');
        const wrapper = document.getElementById('shipments-table-wrapper');
        const filters = document.getElementById('table-filters');

        if (!tableBody || !wrapper) return;

        // Показываем таблицу, скрываем карточки
        document.getElementById('shipments-grid').className = 'shipments-grid hidden';
        wrapper.className = 'shipments-table-wrapper visible';
        if (filters) filters.style.display = 'flex';

        // Применяем фильтр и сортировку
        let filtered = this.filterShipments(this.shipments);
        filtered = this.sortShipments(filtered);

        if (filtered.length === 0) {
            tableBody.innerHTML = `<tr><td colspan="8" class="empty-table">Нет отгрузок</td></tr>`;
            return;
        }

        tableBody.innerHTML = filtered.map(s => `
            <tr data-id="${s.shipmentId}">
                <td>${s.shipmentId}</td>
                <td><strong>${s.number || s.shipmentId}</strong></td>
                <td>${this.formatDate(s.date)}</td>
                <td><span class="status-badge ${s.done ? 'done' : (s.completed ? 'completed' : 'active')}">${this.getStatusText(s)}</span></td>
                <td>
                    <div class="progress-mini"><div class="progress-mini-fill" style="width: ${s.progress}%"></div></div>
                    <span class="progress-text">${s.progress}%</span>
                </td>
                <td>${s.details ? s.details.length : 0}</td>
                <td>${s.details ? s.details.reduce((sum, d) => sum + d.boxes, 0) : 0}</td>
                <td>
                    <div class="table-actions">
                        <button class="btn-view" onclick="ShipmentsModule.viewShipment(${s.shipmentId})" title="Просмотр">👁</button>
                        ${!s.done ? `
                            <button class="btn-edit" onclick="ShipmentsModule.editShipment(${s.shipmentId})" title="Редактировать">✏️</button>
                            <button class="btn-delete" onclick="ShipmentsModule.deleteShipment(${s.shipmentId})" title="Удалить">🗑</button>
                        ` : `
                            <button class="btn-delete" disabled title="Завершённая отгрузка">🗑</button>
                        `}
                    </div>
                </td>
            </tr>
        `).join('');

        // Восстанавливаем индикаторы сортировки
        this.updateSortIndicators();
    },

    filterShipments(shipments) {
        if (this.filterStatus === 'all') return shipments;

        return shipments.filter(s => {
            if (this.filterStatus === 'active') return !s.done && !s.completed;
            if (this.filterStatus === 'completed') return s.completed && !s.done;
            if (this.filterStatus === 'done') return s.done;
            return true;
        });
    },

    sortShipments(shipments) {
        if (!this.sortField) return shipments;

        const field = this.sortField;
        const direction = this.sortDirection === 'asc' ? 1 : -1;

        return [...shipments].sort((a, b) => {
            let aVal = a[field];
            let bVal = b[field];

            // Специальная обработка для разных типов
            if (field === 'status') {
                const statusOrder = { active: 0, completed: 1, done: 2 };
                aVal = statusOrder[aVal] ?? 0;
                bVal = statusOrder[bVal] ?? 0;
            } else if (field === 'number') {
                aVal = aVal || 0;
                bVal = bVal || 0;
            } else if (field === 'progress') {
                aVal = aVal || 0;
                bVal = bVal || 0;
            }

            if (aVal < bVal) return -1 * direction;
            if (aVal > bVal) return 1 * direction;
            return 0;
        });
    },

    applyFilter() {
        // Применяем сохранённый фильтр
        const filterSelect = document.getElementById('status-filter');
        if (filterSelect) {
            filterSelect.value = this.filterStatus;
        }
        if (this.viewMode === 'table') {
            this.renderTableView();
        }
    },

    applyStoredSort() {
        // Применяем сохранённую сортировку к заголовкам таблицы
        if (this.sortField) {
            const headers = document.querySelectorAll('#shipments-table th[data-sort]');
            headers.forEach(th => {
                if (th.dataset.sort === this.sortField) {
                    th.classList.add(this.sortDirection === 'asc' ? 'sorted-asc' : 'sorted-desc');
                }
            });
        }
    },

    updateSortIndicators() {
        // Обновляем классы сортировки на заголовках
        document.querySelectorAll('#shipments-table th[data-sort]').forEach(th => {
            th.classList.remove('sorted-asc', 'sorted-desc');
            if (th.dataset.sort === this.sortField) {
                th.classList.add(this.sortDirection === 'asc' ? 'sorted-asc' : 'sorted-desc');
            }
        });
    },

    attachEvents() {
        // Существующие события
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

        // для переключателя вида
        document.querySelectorAll('.view-toggle-btn').forEach(btn => {
            btn.onclick = () => this.toggleView(btn.dataset.view);
        });

        // События для сортировки таблицы
        document.querySelectorAll('#shipments-table th[data-sort]').forEach(th => {
            th.onclick = () => this.handleSort(th.dataset.sort);
        });

        // События для фильтров
        const statusFilter = document.getElementById('status-filter');
        if (statusFilter) {
            statusFilter.onchange = (e) => {
                this.filterStatus = e.target.value;
                localStorage.setItem('shipmentsFilterStatus', this.filterStatus);
                this.renderTableView();
            };
        }

        const searchInput = document.getElementById('table-search');
        if (searchInput) {
            searchInput.oninput = (e) => {
                this.searchQuery = e.target.value.toLowerCase();
                this.renderTableView();
            };
        }

        const clearFilters = document.getElementById('clear-filters-btn');
        if (clearFilters) {
            clearFilters.onclick = () => {
                const filterSelect = document.getElementById('status-filter');
                const searchInput = document.getElementById('table-search');
                if (filterSelect) filterSelect.value = 'all';
                if (searchInput) searchInput.value = '';
                this.filterStatus = 'all';
                this.searchQuery = '';
                localStorage.setItem('shipmentsFilterStatus', 'all');
                this.renderTableView();
            };
        }
    },

    toggleView(mode) {
        if (this.viewMode === mode) return;
        console.info(mode);
        this.viewMode = mode;
        localStorage.setItem('shipmentsViewMode', mode);

        // Обновляем активную кнопку
        const toggleBtns = document.querySelectorAll('.view-toggle-btn');
        toggleBtns.forEach(btn => {
            if (btn.dataset.view === mode) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
            btn.onclick = (e) => {
                this.toggleView(btn.dataset.view);
            };
        });

        // Перерисовываем вид
        this.renderView();

        // Показываем/скрываем фильтры
        const filters = document.getElementById('table-filters');
        if (filters) {
            filters.style.display = mode === 'table' ? 'flex' : 'none';
        }
    },

    handleSort(field) {
        if (this.sortField === field) {
            // Меняем направление
            this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            this.sortField = field;
            this.sortDirection = 'asc';
        }

        localStorage.setItem('shipmentsSortField', this.sortField);
        localStorage.setItem('shipmentsSortDirection', this.sortDirection);

        this.renderTableView();
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

            // 2. Проверяем, что отгрузка не завершена
            if (shipment.completed) {
                alert('Нельзя редактировать готовую отгрузку');
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
        try {
            // Получаем данные отгрузки для проверки статуса
            const shipment = await API.getShipmentById(shipmentId);
            if (!shipment) {
                alert('Отгрузка не найдена');
                return;
            }

            // Проверяем статус
            if (shipment.done) {
                alert('❌ Нельзя удалить завершённую отгрузку');
                return;
            }

            if (shipment.completed) {
                alert('❌ Нельзя удалить готовую отгрузку');
                return;
            }
            if (!confirm('Удалить отгрузку?')) return;
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