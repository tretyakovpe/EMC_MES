class StatisticsScreen {
    constructor() {
        this.name = 'statistics';
        this.container = null;
        this.currentMode = 'boxes'; // 'boxes' или 'bad-parts'
        this.currentData = [];
        this.selectedBox = null;
        this.lines = ['Все'];
        
        // Настройки дат (по умолчанию сегодня)
        const today = new Date();
        this.fromDate = this.formatDate(today);
        this.toDate = this.formatDate(today);
    }

    formatDate(date) {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }

    async init() {
        // Создаём контейнер
        this.container = document.getElementById('screen-container');
        this.container.innerHTML = `
            <div class="statistics-container">
                <div class="statistics-table-wrapper">
                    <div class="statistics-table-container">
                        <table class="statistics-table" id="statistics-table">
                            <thead id="table-header">
                                <tr>
                                    <th>Загрузка...</th>
                                </tr>
                            </thead>
                            <tbody id="table-body">
                                <tr><td colspan="10">Загрузка данных...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <div class="statistics-panel" id="statistics-panel">
                    <div class="panel-header">
                        <h3>📊 ФИЛЬТРЫ</h3>
                    </div>
                    <div class="panel-tabs">
                        <button class="panel-tab active" data-mode="boxes">📦 Продукция</button>
                        <button class="panel-tab" data-mode="bad-parts">⚠️ Брак</button>
                    </div>
                    <div class="panel-filters">
                        <div class="filter-group">
                            <label>Дата от</label>
                            <input type="date" id="filter-from-date" class="filter-input" value="${this.fromDate}">
                        </div>
                        <div class="filter-group">
                            <label>Дата до</label>
                            <input type="date" id="filter-to-date" class="filter-input" value="${this.toDate}">
                        </div>
                        <div class="filter-group">
                            <label>Линия</label>
                            <select id="filter-line" class="filter-select">
                                <option value="Все">Загрузка...</option>
                            </select>
                        </div>
                        <button id="apply-filters" class="filter-btn">🔍 Применить</button>
                    </div>
                    <div class="panel-actions" id="panel-actions" style="display: none;">
                        <div class="action-divider"></div>
                        <div class="action-title">📦 Управление коробкой</div>
                        <button id="view-label-btn" class="action-btn">👁 Просмотр бирки</button>
                        <button id="print-label-btn" class="action-btn">🖨 Печать бирки</button>
                    </div>
                </div>
            </div>
        `;

        // Загружаем список линий
        await this.loadLines();
        
        // Загружаем начальные данные
        await this.loadData();
        
        // Настраиваем обработчики
        this.setupEventHandlers();
        
        console.log('Statistics screen initialized');
    }

    async loadLines() {
        try {
            const lines = await API.get('/statistics/lines');
            this.lines = lines || ['Все'];
            
            const select = document.getElementById('filter-line');
            if (select) {
                select.innerHTML = this.lines.map(line => 
                    `<option value="${this.escapeHtml(line)}">${this.escapeHtml(line)}</option>`
                ).join('');
            }
        } catch (error) {
            console.error('Failed to load lines:', error);
        }
    }

    async loadData() {
        const fromDate = document.getElementById('filter-from-date')?.value || this.fromDate;
        const toDate = document.getElementById('filter-to-date')?.value || this.toDate;
        const lineSelect = document.getElementById('filter-line');
        const line = lineSelect?.value || 'Все';
        
        try {
            let url;
            if (this.currentMode === 'boxes') {
                url = `/api/statistics/boxes?fromDate=${fromDate}&toDate=${toDate}&line=${encodeURIComponent(line)}`;
            } else {
                url = `/api/statistics/bad-parts?fromDate=${fromDate}&toDate=${toDate}&line=${encodeURIComponent(line)}`;
            }
            
            this.currentData = await API.get(url.replace('/api/', ''));
            this.renderTable();
        } catch (error) {
            console.error('Failed to load data:', error);
            this.showError('Ошибка загрузки данных');
        }
    }

    renderTable() {
        const thead = document.getElementById('table-header');
        const tbody = document.getElementById('table-body');
        
        if (!thead || !tbody) return;
        
        if (this.currentMode === 'boxes') {
            // Заголовки для продукции
            thead.innerHTML = `
                <tr>
                    <th>Дата</th>
                    <th>Время</th>
                    <th>Бирка</th>
                    <th>Материал</th>
                    <th>Кол-во</th>
                </tr>
            `;
            
            if (this.currentData.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="empty-row">Нет данных за выбранный период</td></tr>';
                return;
            }
            
            tbody.innerHTML = this.currentData.map(item => `
                <tr data-label="${this.escapeHtml(item.label)}" data-type="box">
                    <td>${this.escapeHtml(item.date)}</td>
                    <td>${this.escapeHtml(item.time)}</td>
                    <td>${this.escapeHtml(item.label)}</td>
                    <td>${this.escapeHtml(item.materialCode)}</td>
                    <td>${item.amount}</td>
                </tr>
            `).join('');
            
        } else {
            // Заголовки для брака
            thead.innerHTML = `
                <tr>
                    <th>Дата и время</th>
                    <th>Линия</th>
                    <th>Тип детали</th>
                    <th>Номер</th>
                    <th>Подробности</th>
                    <th>Видео</th>
                </tr>
            `;
            
            if (this.currentData.length === 0) {
                tbody.innerHTML = '<tr><td colspan="6" class="empty-row">Нет данных за выбранный период</td></tr>';
                return;
            }
            
            tbody.innerHTML = this.currentData.map(item => `
                <tr data-counter="${item.counter}" data-type="bad-part">
                    <td>${this.escapeHtml(item.dateTime)}</td>
                    <td>${this.escapeHtml(item.line)}</td>
                    <td>${this.escapeHtml(item.material)}</td>
                    <td>${item.counter}</td>
                    <td>${this.escapeHtml(item.mkm)}</td>
                    <td>${item.video !== '—' ? `<a href="#" class="video-link" data-video="${this.escapeHtml(item.video)}">📹 Видео</a>` : '—'}</td>
                </tr>
            `).join('');
        }
        
        // Добавляем обработчики кликов по строкам
        this.attachRowClickHandlers();
    }

    attachRowClickHandlers() {
        const rows = document.querySelectorAll('#table-body tr');
        rows.forEach(row => {
            row.removeEventListener('click', this.rowClickHandler);
            this.rowClickHandler = () => {
                // Убираем выделение со всех строк
                document.querySelectorAll('#table-body tr').forEach(r => r.classList.remove('selected'));
                row.classList.add('selected');
                
                const type = row.dataset.type;
                if (type === 'box') {
                    const label = row.dataset.label;
                    this.selectedBox = this.currentData.find(item => item.label === label);
                    this.showPanelActions(true);
                } else {
                    this.selectedBox = null;
                    this.showPanelActions(false);
                }
            };
            row.addEventListener('click', this.rowClickHandler);
        });
    }

    showPanelActions(show) {
        const actionsPanel = document.getElementById('panel-actions');
        if (actionsPanel) {
            actionsPanel.style.display = show ? 'block' : 'none';
        }
    }

    showError(message) {
        const tbody = document.getElementById('table-body');
        if (tbody) {
            tbody.innerHTML = `<tr><td colspan="10" class="error-row">❌ ${message}</td></tr>`;
        }
    }

    setupEventHandlers() {
        // Переключение вкладок
        const tabs = document.querySelectorAll('.panel-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', async (e) => {
                tabs.forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
                
                this.currentMode = tab.dataset.mode;
                this.selectedBox = null;
                this.showPanelActions(false);
                await this.loadData();
            });
        });
        
        // Кнопка "Применить"
        const applyBtn = document.getElementById('apply-filters');
        if (applyBtn) {
            applyBtn.addEventListener('click', async () => {
                this.selectedBox = null;
                this.showPanelActions(false);
                await this.loadData();
            });
        }
        
        // Кнопка "Просмотр бирки" (заглушка)
        const viewBtn = document.getElementById('view-label-btn');
        if (viewBtn) {
            viewBtn.addEventListener('click', () => {
                if (this.selectedBox) {
                    alert(`Просмотр бирки ${this.selectedBox.label}\nФункция в разработке`);
                }
            });
        }
        
        // Кнопка "Печать бирки" (заглушка)
        const printBtn = document.getElementById('print-label-btn');
        if (printBtn) {
            printBtn.addEventListener('click', () => {
                if (this.selectedBox) {
                    alert(`Печать бирки ${this.selectedBox.label}\nФункция в разработке`);
                }
            });
        }
        
        // Обработка видео ссылок (делегирование)
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('video-link')) {
                e.preventDefault();
                const video = e.target.dataset.video;
                alert(`Просмотр видео: ${video}\nФункция в разработке`);
            }
        });
    }

    escapeHtml(str) {
        if (!str) return '';
        return String(str).replace(/[&<>]/g, function(m) {
            if (m === '&') return '&amp;';
            if (m === '<') return '&lt;';
            if (m === '>') return '&gt;';
            return m;
        });
    }

    destroy() {
        if (this.container) {
            this.container.innerHTML = '';
        }
        this.currentData = [];
        this.selectedBox = null;
    }
}