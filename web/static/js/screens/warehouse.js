// ========== МОДУЛЬ СКЛАДА ГОТОВОЙ ПРОДУКЦИИ ==========

const WarehouseModule = {
    config: null,
    stacksData: [],
    updateInterval: null,
    container: null,

    async render(container) {
        this.container = container;

        // Загружаем конфиг напрямую
        try {
            const response = await fetch('/static/config/warehouse.json');
            if (response.ok) {
                this.config = await response.json();
            } else {
                console.warn('Конфиг не найден, используем значения по умолчанию');
                this.config = this.getDefaultConfig();
            }
        } catch (error) {
            console.error('Ошибка загрузки конфига:', error);
            this.config = this.getDefaultConfig();
        }

        // Отображаем контейнер
        container.innerHTML = `
            <div class="logistics-container">
                <div class="warehouse-header">
                    <h3>🏭 Склад готовой продукции</h3>
                </div>
                <div id="warehouse-content" class="warehouse-content">
                    <div class="loading">Загрузка...</div>
                </div>
            </div>
        `;

        // Загружаем данные
        await this.loadData();
        this.renderStacks();

        // Обновление по кнопке
        document.getElementById('refresh-warehouse')?.addEventListener('click', () => {
            this.refreshData();
        });

        // WebSocket для реального времени
        this.initWebSocket();

        // Автообновление каждые 30 секунд
        this.updateInterval = setInterval(() => this.refreshData(), 30000);

        console.log('WarehouseModule initialized');
    },

    getDefaultConfig() {
        return {
            layout: {
                rows: 2,
                aisleWidth: 40,
                stackWidth: 100,
                stackHeight: 200,
                maxBoxes: 40
            },
            stacks: []
        };
    },

    async loadData() {
        try {
            const response = await fetch('/api/warehouse/stacks');
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            this.stacksData = await response.json();
            console.log('Warehouse data:', this.stacksData);
        } catch (error) {
            console.error('Ошибка загрузки склада:', error);
            this.stacksData = [];
        }
    },

    async refreshData() {
        await this.loadData();
        this.renderStacks();
    },

    renderStacks() {
        const content = document.getElementById('warehouse-content');
        if (!content) return;

        const layout = this.config.layout || {};
        const rows = layout.rows || 2;
        const aisleWidth = layout.aisleWidth || 40;
        const stackWidth = layout.stackWidth || 100;
        const stackHeight = layout.stackHeight || 200;
        const stacks = this.config.stacks || [];
        const defaultMaxBoxes = layout.maxBoxes || 40;

        // Группируем стеки по материалам
        const stacksByMaterial = {};
        for (const stack of stacks) {
            const materialCode = stack.materialCode;
            if (!stacksByMaterial[materialCode]) {
                stacksByMaterial[materialCode] = [];
            }
            stacksByMaterial[materialCode].push(stack);
        }

        // Распределяем коробки по стекам
        for (const materialCode in stacksByMaterial) {
            const materialStacks = stacksByMaterial[materialCode];
            const data = this.stacksData.find(s => s.materialCode === materialCode);
            const totalBoxes = data?.boxCount || 0;

            let remaining = totalBoxes;
            for (const stack of materialStacks) {
                const maxBoxes = stack.maxBoxes || defaultMaxBoxes;
                const filled = Math.min(remaining, maxBoxes);
                stack._fillCount = filled;
                stack._maxCount = maxBoxes;
                remaining -= filled;
            }

            // Если остались коробки, а стеков больше нет — помечаем последний стек как переполненный
            if (remaining > 0 && materialStacks.length > 0) {
                const lastStack = materialStacks[materialStacks.length - 1];
                lastStack._overfilled = true;
                lastStack._remaining = remaining;
            }
        }

        // Размещаем стеки по рядам
        const rowsData = [[], []];
        for (const stack of stacks) {
            if (stack.position.row < rows) {
                rowsData[stack.position.row][stack.position.col] = stack;
            }
        }

        let html = `<div class="warehouse-grid" style="min-width: ${(rowsData[0].length * (stackWidth + 16) + 40)}px;">`;

        for (let row = 0; row < rows; row++) {
            html += `<div class="warehouse-row">`;

            for (let col = 0; col < rowsData[row].length; col++) {
                const stack = rowsData[row][col];
                const materialCode = stack?.materialCode;
                const maxBoxes = stack?._maxCount || defaultMaxBoxes;
                const fillCount = stack?._fillCount || 0;
                const fillPercent = Math.min(100, (fillCount / maxBoxes) * 100);
                const isEmpty = !materialCode || materialCode === '';
                const isOverfilled = stack?._overfilled || false;
                const remaining = stack?._remaining || 0;
                const isFull = fillCount >= maxBoxes && !isOverfilled;
                const isPartiallyFilled = fillCount > 0 && fillCount < maxBoxes;

                // Определяем класс состояния
                let stateClass = '';
                if (isOverfilled) stateClass = 'overfilled';
                else if (isFull) stateClass = 'full';
                else if (isPartiallyFilled) stateClass = 'partial';
                else if (isEmpty) stateClass = 'empty';

                // Цвет воды
                let waterColor = '#28a745'; // зелёный
                if (isOverfilled) waterColor = '#cc0000'; // красный
                else if (isFull) waterColor = '#ffaa00'; // жёлтый

                html += `
                    <div class="stack-card ${stateClass}" 
                         data-material="${materialCode || '🆓'}"
                         style="width: ${stackWidth}px; height: ${stackHeight}px;">
                        <div class="stack-material">${materialCode || ''}</div>
                        <div class="stack-tank">
                            <div class="stack-water" style="height: ${fillPercent}%; background: ${waterColor};"></div>
                        </div>
                        <div class="stack-stats">
                            <div>📦 ${fillCount}/${maxBoxes}</div>
                            ${isOverfilled ? `<div class="overfill-warning">⚠️ +${remaining} кор.</div>` : ''}
                        </div>
                    </div>
                `;
            }
            html += `</div>`;

            // Проход между рядами
            if (row === 0 && rows > 1) {
                html += `<div class="warehouse-aisle" style="height: ${aisleWidth}px;"></div>`;
            }
        }

        html += `</div>`;
        content.innerHTML = html;

        // Привязываем обработчики кликов
        document.querySelectorAll('.stack-card:not(.empty)').forEach(card => {
            card.addEventListener('click', () => {
                const materialCode = card.dataset.material;
                if (materialCode) {
                    this.showBoxesList(materialCode);
                }
            });
        });
    },

    async showBoxesList(materialCode) {
        try {
            const response = await fetch(`/api/boxes?materialCode=${encodeURIComponent(materialCode)}&status=Произведена`);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            const boxes = await response.json();

            console.log(`Коробки материала ${materialCode}:`, boxes);
            const title = `Коробки: ${materialCode}`;
            window.location.href = `/table-view?view=boxes-by-material&material=${encodeURIComponent(materialCode)}&title=${encodeURIComponent(title)}`;
        } catch (error) {
            console.error('Ошибка загрузки коробок:', error);
            alert('Ошибка загрузки коробок');
        }
    },

    initWebSocket() {
        const ws = new WebSocket(`ws://${window.location.host}/ws`);

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                if (message.type === 'box_closed') {
                    this.refreshData();
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
        if (this.updateInterval) clearInterval(this.updateInterval);
    }
};

// Глобальная функция для onclick
window.WarehouseModule = WarehouseModule;