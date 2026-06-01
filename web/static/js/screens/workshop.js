class WorkshopScreen {
    constructor() {
        this.name = 'workshop';
        this.lines = new Map();
        this.selectedLine = null;
        this.container = null;
        this.ws = null;
    }

    async init() {
        // Создаём контейнер
        this.container = document.getElementById('screen-container');
        this.container.innerHTML = `
            <div class="workshop-container">
                <div class="lines-grid" id="lines-grid"></div>
                <div class="detail-panel" id="detailPanel">
                    <div class="panel-header">
                        <h3>🔧 ДЕТАЛИ ЛИНИИ</h3>
                        <div class="panel-status" id="panelStatus">⚪ НЕ ВЫБРАНО</div>
                    </div>
                    <div class="panel-content" id="panelContent">
                        <div class="placeholder">
                            <div class="icon">🏭</div>
                            <p>Нажмите на линию<br>для просмотра деталей</p>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Загружаем начальные данные (один раз)
        await this.loadLines();

        // Подключаем WebSocket для живых событий
        this.initWebSocket();

        console.log('Workshop screen initialized');
    }

    async loadLines() {
        const linesData = await API.getLines();
        this.lines.clear();

        for (const data of linesData) {
            const line = {
                name: data.name,
                isOnline: data.isOnline,
                isActive: data.isActive,
                printer: data.printer,
                ip: data.ip,
                currentMaterial: null,
                currentCount: 0,
                maxCount: 100,
                targetCount: 0
            };
            this.lines.set(data.name, line);
        }

        this.renderLines();
    }

    renderLines() {
        const grid = document.getElementById('lines-grid');
        if (!grid) return;

        grid.innerHTML = '';

        for (const [name, line] of this.lines) {
            const statusClass = line.isOnline ? 'online' : 'offline';
            const statusText = line.isOnline ? '🟢 ONLINE' : '🔴 OFFLINE';
            const fillPercent = line.maxCount > 0 ? (line.currentCount / line.maxCount * 100) : 0;
            const isSelected = this.selectedLine && this.selectedLine.name === name;

            const card = document.createElement('div');
            card.className = `line-card ${isSelected ? 'selected' : ''}`;
            card.dataset.line = name;
            card.innerHTML = `
                <div class="line-header">
                    <div class="line-name">${this.escapeHtml(name)}</div>
                    <div class="line-status ${statusClass}">${statusText}</div>
                </div>
                <div class="line-body">
                    <div class="line-material">
                        📦 Материал: <span>${this.escapeHtml(line.currentMaterial || '—')}</span>
                    </div>
                    <div class="progress-container">
                        <div class="progress-label">
                            <span>Заполнение</span>
                            <span>${Math.round(fillPercent)}%</span>
                        </div>
                        <div class="progress-bar-bg">
                            <div class="progress-bar-fill" style="width: ${fillPercent}%"></div>
                        </div>
                        <div class="line-counter">${line.currentCount}/${line.maxCount} шт.</div>
                    </div>
                </div>
                <div class="line-footer">
                    <div class="line-ip">🌐 <span>${this.escapeHtml(line.ip || '—')}</span></div>
                    <button class="btn-toggle-status ${line.isActive ? 'btn-offline' : 'btn-online'}" 
                            data-line="${name}" 
                            data-status="${line.isActive}">
                        ${line.isActive ? '🟢 Включена' : '🔴 Выключена'}
                    </button>
                </div>
            `;

            card.addEventListener('click', (e) => {
                if (!e.target.classList.contains('btn-toggle-status')) {
                    this.selectLine(name);
                }
            });

            grid.appendChild(card);
        }

        this.attachStatusButtonHandlers();
    }

    attachStatusButtonHandlers() {
        const buttons = document.querySelectorAll('.btn-toggle-status');
        buttons.forEach(btn => {
            const newBtn = btn.cloneNode(true);
            btn.parentNode.replaceChild(newBtn, btn);

            newBtn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const lineName = newBtn.dataset.line;
                const currentStatus = newBtn.dataset.status === 'true';
                const newStatus = !currentStatus;

                try {
                    const response = await fetch(`/api/lines/status?name=${encodeURIComponent(lineName)}`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ isOnline: newStatus })
                    });

                    if (!response.ok) throw new Error(`HTTP ${response.status}`);

                    // Обновляем локально (сервер сам разошлёт событие через WebSocket)
                    const line = this.lines.get(lineName);
                    if (line) {
                        line.isOnline = newStatus;
                        this.renderLines();
                        if (this.selectedLine && this.selectedLine.name === lineName) {
                            this.selectedLine.isOnline = newStatus;
                            this.updateDetailPanel(this.selectedLine);
                        }
                    }
                } catch (error) {
                    console.error('Failed to toggle status:', error);
                }
            });
        });
    }

    selectLine(lineName) {
        const line = this.lines.get(lineName);
        if (!line) return;

        this.selectedLine = line;
        this.updateDetailPanel(line);
        this.renderLines();
    }

    updateDetailPanel(line) {
        const panelContent = document.getElementById('panelContent');
        const panelStatus = document.getElementById('panelStatus');

        if (!line) {
            panelContent.innerHTML = `
                <div class="placeholder">
                    <div class="icon">🏭</div>
                    <p>Нажмите на линию<br>для просмотра деталей</p>
                </div>
            `;
            panelStatus.innerHTML = '⚪ НЕ ВЫБРАНО';
            panelStatus.className = 'panel-status';
            return;
        }

        const statusClass = line.isOnline ? 'online' : 'offline';
        const statusText = line.isOnline ? 'ОНЛАЙН' : 'ОФФЛАЙН';
        const statusIcon = line.isOnline ? '🟢' : '🔴';
        const fillPercent = line.maxCount > 0 ? (line.currentCount / line.maxCount * 100) : 0;

        panelStatus.innerHTML = `${statusIcon} ${statusText}`;
        panelStatus.className = `panel-status ${statusClass}`;

        panelContent.innerHTML = `
            <div class="line-detail">
                <div class="detail-row">
                    <div class="detail-label">🔧 ЛИНИЯ</div>
                    <div class="detail-value large">${this.escapeHtml(line.name)}</div>
                </div>
                
                <div class="detail-row">
                    <div class="detail-label">📡 СТАТУС</div>
                    <div class="detail-value">
                        <span class="status-badge ${statusClass}">${statusText}</span>
                    </div>
                </div>
                
                <div class="detail-row">
                    <div class="detail-label">🌐 IP АДРЕС</div>
                    <div class="detail-value">${this.escapeHtml(line.ip || '—')}</div>
                </div>
                
                <div class="detail-row">
                    <div class="detail-label">🖨️ ПРИНТЕР</div>
                    <div class="detail-value">${this.escapeHtml(line.printer || '—')}</div>
                </div>
                
                <div class="detail-row">
                    <div class="detail-label">📦 ТЕКУЩАЯ КОРОБКА</div>
                    <div class="detail-value">${this.escapeHtml(line.currentMaterial || '—')}</div>
                    <div class="progress-bar-bg" style="margin-top: 8px;">
                        <div class="progress-bar-fill" style="width: ${fillPercent}%"></div>
                    </div>
                    <div style="font-size: 11px; margin-top: 4px;">${line.currentCount}/${line.maxCount} шт.</div>
                </div>
            </div>
        `;
    }

    initWebSocket() {
        this.ws = new WebSocket(`ws://${window.location.host}/ws`);

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);

                switch (message.type) {
                    case 'box_closed':
                        this.handleBoxClosed(message.data);
                        break;

                    case 'part_ok':
                        this.handlePartProduced(message.data, true);
                        break;

                    case 'part_nok':
                        this.handlePartProduced(message.data, false);
                        break;

                    case 'line_status':
                        this.handleLineStatusChange(message.data);
                        break;
                    case 'line_active':
                        this.handleLineActiveChange(message.data);
                        break;
                    case 'line_card_update':
                        this.handleLineCardUpdate(message.data);
                        break;

                }
            } catch (error) {
                console.error('WebSocket parse error:', error);
            }
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected, reconnecting in 5s...');
            setTimeout(() => this.initWebSocket(), 5000);
        };
    }

    handleBoxClosed(data) {
        const line = this.lines.get(data.line);
        if (line) {
            line.currentMaterial = data.material;
            line.currentCount = 0;
            line.targetCount = 0;
            this.renderLines();
            if (this.selectedLine && this.selectedLine.name === data.line) {
                this.updateDetailPanel(line);
            }
        }
    }

    handlePartProduced(data, isGood) {
        const line = this.lines.get(data.line);
        if (line) {
            line.currentMaterial = data.material;
            line.currentCount = data.counter;
            line.maxCount = data.boxVolume;
        }
    }

    handleLineStatusChange(data) {
        const line = this.lines.get(data.line);
        if (line && line.isOnline !== data.isOnline) {
            line.isOnline = data.isOnline;
            this.renderLines();
            if (this.selectedLine && this.selectedLine.name === data.line) {
                this.selectedLine.isOnline = data.isOnline;
                this.updateDetailPanel(this.selectedLine);
            }
        }
    }

    handleLineActiveChange(data) {
        const line = this.lines.get(data.line);
        if (line && line.isActive !== data.isActive) {
            line.isActive = data.isActive;
            this.renderLines();
            if (this.selectedLine && this.selectedLine.name === data.line) {
                this.selectedLine.isActive = data.isActive;
                this.updateDetailPanel(this.selectedLine);
            }
        }
    }

    handleLineCardUpdate(data) {
        const line = this.lines.get(data.line);
        if (line) {
            line.currentMaterial = data.material;
            line.currentCount = data.counter;
            line.maxCount = data.boxVolume;
        }
            this.renderLines();
    }

    animateLine(lineName, type) {
        const cards = document.querySelectorAll('.line-card');
        for (const card of cards) {
            if (card.dataset.line === lineName) {
                card.classList.add('producing');
                setTimeout(() => {
                    card.classList.remove('producing');
                }, 1);
                break;
            }
        }
    }

    animateCounter(line) {
/*        const start = line.currentCount-1;
        const end = line.currentCount;
        const duration = 300;
        const startTime = performance.now();
        
        const animate = (now) => {
            const elapsed = now - startTime;
            const progress = Math.min(1, elapsed / duration);
            line.currentCount = Math.floor(start + (end - start) * progress);
*/            this.renderLines();
        if (this.selectedLine === line) this.updateDetailPanel(line);

        if (progress < 1) {
            requestAnimationFrame(animate);
        }
        //};

        requestAnimationFrame(animate);
    }

    escapeHtml(str) {
        if (!str) return '';
        return str.replace(/[&<>]/g, function (m) {
            if (m === '&') return '&amp;';
            if (m === '<') return '&lt;';
            if (m === '>') return '&gt;';
            return m;
        });
    }

    destroy() {
        if (this.ws) this.ws.close();
        if (this.container) this.container.innerHTML = '';
        this.lines.clear();
    }
}