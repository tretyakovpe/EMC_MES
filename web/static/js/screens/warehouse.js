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
            const response = await fetch('static/config/warehouse.json');
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

        // Рендерим
        this.renderStacks();

        // WebSocket для реального времени
        this.initWebSocket();

        // Автообновление каждые 10 секунд
        //this.updateInterval = setInterval(() => this.refreshData(), 10000);

        console.log('WarehouseModule initialized');
    },

    getDefaultConfig() {
        return {
            layout: {
                rows: 2,
                aisleWidth: 10,
                stackWidth: 150,
                stackHeight: 200,
                maxBoxes: 40
            },
            stacks: [
                { "materialCode": "C22348-103", "position": { "row": 0, "col": 0 } },
                { "materialCode": "LD1100-101", "position": { "row": 0, "col": 1 } },
                { "materialCode": "LL1200-101", "position": { "row": 0, "col": 2 } },
                { "materialCode": "LO2300-100", "position": { "row": 0, "col": 3 } },
                { "materialCode": "E68986-104", "position": { "row": 0, "col": 4 } },
                { "materialCode": "LF1100-101", "position": { "row": 1, "col": 0 } },
                { "materialCode": "LF1200-101", "position": { "row": 1, "col": 1 } },
                { "materialCode": "LF2300-100", "position": { "row": 1, "col": 2 } },
                { "materialCode": "LF2400-101", "position": { "row": 1, "col": 3 } },
                { "materialCode": "LO2200-100", "position": { "row": 1, "col": 4 } }
            ]
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
        const aisleWidth = layout.aisleWidth || 10;
        const stackWidth = layout.stackWidth || 150;
        const stackHeight = layout.stackHeight || 200;
        const maxBoxes = layout.maxBoxes || 40;

        // Группируем стеки по рядам из конфига
        const stacks = this.config.stacks || [];
        const rowsData = [[], []];
        for (const stack of stacks) {
            if (stack.position.row < rows) {
                rowsData[stack.position.row][stack.position.col] = stack;
            }
        }

        if (this.stacksData.length === 0 && stacks.length === 0) {
            content.innerHTML = '<div class="empty">📭 Нет данных о коробках</div>';
            return;
        }

        let html = `<div class="warehouse-grid" style="min-width: ${(rowsData[0].length * stackWidth + 100)}px;">`;

        for (let row = 0; row < rows; row++) {
            html += `<div class="warehouse-row">`;

            for (let col = 0; col < rowsData[row].length; col++) {
                const stack = rowsData[row][col];
                const materialCode = stack?.materialCode;

                // Ищем данные по materialCode
                const data = this.stacksData.find(s => s.materialCode === materialCode);
                const boxCount = data?.boxCount || 0;
                const totalAmount = data?.totalAmount || 0;
                const fillPercent = Math.min(100, (boxCount / maxBoxes) * 100);
                const isEmpty = !materialCode || materialCode === '';

                html += `
                    <div class="stack-card ${isEmpty ? 'empty' : ''}" 
                         data-material="${materialCode || ''}"
                         style="width: ${stackWidth}px; height: ${stackHeight}px;">
                        <div class="stack-material">${materialCode || 'Свободно'}</div>
                        <div class="stack-tank">
                            <div class="stack-water" style="height: ${fillPercent}%;"></div>
                        </div>
                        <div class="stack-stats">
                            <div>📦 ${boxCount}/${maxBoxes}</div>
                            <div>📊 ${totalAmount} шт.</div>
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
                    // Переход на таблицу коробок
                    const title = `Коробки: ${materialCode}`;
                    window.location.href = `/table-view?view=boxes-by-material&material=${encodeURIComponent(materialCode)}&title=${encodeURIComponent(title)}&status="Произведена"`;
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
            alert(`Материал: ${materialCode}\nКоробок: ${boxes.length}\nИнформация в консоли (F12)`);

            // TODO: позже добавить модальное окно с печатью
        } catch (error) {
            console.error('Ошибка загрузки коробок:', error);
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

    destroy() {
        if (this.updateInterval) clearInterval(this.updateInterval);
    }
};