// ========== ПЛАНЫ (сворачиваемые дни) ==========

let plansModule = {
    container: null,
    currentMonth: null,
    materials: null,
    plansMap: null,
    expandedDays: new Set(),  // храним развёрнутые дни (YYYY-MM-DD)
    
    async render(container) {
        this.container = container;
        
        // Загружаем данные
        const [plans, materials] = await Promise.all([
            API.getPlans(),
            API.getMaterials()
        ]);
        this.materials = materials;
        
        // Группируем планы
        this.plansMap = new Map();
        plans.forEach(p => {
            const key = `${p.materialId}_${p.planDate}_${p.shift}`;
            this.plansMap.set(key, p.plannedAmount);
        });
        
        // Получаем месяц
        this.currentMonth = localStorage.getItem('plans_current_month') || new Date().toISOString().slice(0, 7);
        const dates = this.getMonthDates(this.currentMonth);
        
        container.innerHTML = `
            <div class="plans-container">
                <div class="plans-filter-bar">
                    <div class="filter-group">
                        <label>📅 Месяц</label>
                        <input type="month" id="month-selector" value="${this.currentMonth}" class="filter-input">
                    </div>
                    <button id="save-plans" class="btn-success">💾 Сохранить все</button>
                </div>
                <div class="plans-table-wrapper">
                    <table class="plans-table" id="plans-table">
                        <thead id="plans-thead">
                            ${this.renderHeader(dates)}
                        </thead>
                        <tbody id="plans-table-body">
                            ${this.materials.map(m => this.renderRow(m, dates)).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        `;
        
        this.bindEvents();
    },
    
    renderHeader(dates) {
        // Верхняя строка с датами
        let headerRow = `<tr><th class="sticky-col">Материал</th>`;
        
        for (const date of dates) {
            const isExpanded = this.expandedDays.has(date);
            const colspan = isExpanded ? 4 : 1;
            headerRow += `<th colspan="${colspan}" class="date-header" data-date="${date}">
                             ${this.formatDate(date)} 
                             <span class="expand-icon">${isExpanded ? '▼' : '▶'}</span>
                          </th>`;
        }
        headerRow += `<th class="month-total-header">Месяц</th>`;
        headerRow += `</tr>`;
        
        // Вторая строка с названиями колонок (только для развёрнутых дней)
        let subHeaderRow = `<tr><th class="sticky-col"></th>`;
        
        for (const date of dates) {
            const isExpanded = this.expandedDays.has(date);
            if (isExpanded) {
                subHeaderRow += `<th>1см</th><th>2см</th><th>3см</th><th>Итого</th>`;
            } else {
                subHeaderRow += `<th>Итого</th>`;
            }
        }
        subHeaderRow += `<th></th></tr>`;
        
        return headerRow + subHeaderRow;
    },
    
    renderRow(material, dates) {
        let monthTotal = 0;
        
        const cells = dates.map(date => {
            const shift1 = this.getPlanAmount(material.materialId, date, "1");
            const shift2 = this.getPlanAmount(material.materialId, date, "2");
            const shift3 = this.getPlanAmount(material.materialId, date, "3");
            const dayTotal = shift1 + shift2 + shift3;
            monthTotal += dayTotal;
            
            const isExpanded = this.expandedDays.has(date);
            
            if (isExpanded) {
                // Развёрнутый день: 4 колонки
                return `
                    <td class="editable-cell" data-material-id="${material.materialId}" data-date="${date}" data-shift="1">
                        <input type="number" class="plan-input" value="${shift1}" step="10">
                    </td>
                    <td class="editable-cell" data-material-id="${material.materialId}" data-date="${date}" data-shift="2">
                        <input type="number" class="plan-input" value="${shift2}" step="10">
                    </td>
                    <td class="editable-cell" data-material-id="${material.materialId}" data-date="${date}" data-shift="3">
                        <input type="number" class="plan-input" value="${shift3}" step="10">
                    </td>
                    <td class="day-total-cell" data-date="${date}">${dayTotal}</td>
                `;
            } else {
                // Свёрнутый день: только итого
                return `
                    <td class="day-total-cell collapsed" data-date="${date}" data-material-id="${material.materialId}">
                        ${dayTotal}
                    </td>
                `;
            }
        }).join('');
        
        return `
            <tr>
                <td class="material-cell" data-material-id="${material.materialId}">
                    ${escapeHtml(material.materialCode)}
                </td>
                ${cells}
                <td class="month-total-cell">${monthTotal}</td>
            </tr>
        `;
    },
    
    getMonthDates(monthStr) {
        const [year, month] = monthStr.split('-').map(Number);
        const firstDay = new Date(year, month - 1, 1);
        const lastDay = new Date(year, month, 0);
        const dates = [];
        for (let d = new Date(firstDay); d <= lastDay; d.setDate(d.getDate() + 1)) {
            dates.push(new Date(d).toISOString().slice(0, 10));
        }
        return dates;
    },
    
    formatDate(dateStr) {
        const parts = dateStr.split('-');
        return `${parts[2]}.${parts[1]}`;
    },
    
    getPlanAmount(materialId, date, shift) {
        const key = `${materialId}_${date}_${shift}`;
        return this.plansMap.get(key) || 0;
    },
    
    bindEvents() {
        // Сохранение месяца
        const monthSelector = document.getElementById('month-selector');
        if (monthSelector) {
            monthSelector.addEventListener('change', (e) => {
                localStorage.setItem('plans_current_month', e.target.value);
                this.render(this.container);
            });
        }
        
        // Сохранение планов
        document.getElementById('save-plans')?.addEventListener('click', () => this.saveAll());
        
        // Обработчик клика по заголовку дня
        document.querySelectorAll('.date-header').forEach(header => {
            header.removeEventListener('click', this.onDateClick);
            header.addEventListener('click', this.onDateClick.bind(this));
        });
        
        // Обработчики изменения ячеек
        this.attachCellHandlers();
    },
    
    onDateClick(e) {
        const header = e.currentTarget;
        const date = header.dataset.date;
        
        if (this.expandedDays.has(date)) {
            this.expandedDays.delete(date);
        } else {
            this.expandedDays.add(date);
        }
        
        // Перерисовываем таблицу (сохраняем состояние)
        this.render(this.container);
    },
    
    attachCellHandlers() {
        document.querySelectorAll('.plan-input').forEach(input => {
            input.removeEventListener('change', this.onPlanChange);
            input.addEventListener('change', this.onPlanChange.bind(this));
        });
        
        // Также для свёрнутых ячеек (показываем подсказку)
        document.querySelectorAll('.day-total-cell.collapsed').forEach(cell => {
            cell.style.cursor = 'pointer';
            cell.title = 'Разверните день для редактирования';
        });
    },
    
    onPlanChange(e) {
        const row = e.target.closest('tr');
        const td = e.target.closest('td');
        const date = td.dataset.date;
        
        // Обновляем итого по дню
        const dayTotalCell = row.querySelector(`.day-total-cell[data-date="${date}"]`);
        if (dayTotalCell) {
            const shift1 = parseInt(row.querySelector(`.editable-cell[data-date="${date}"][data-shift="1"] input`)?.value) || 0;
            const shift2 = parseInt(row.querySelector(`.editable-cell[data-date="${date}"][data-shift="2"] input`)?.value) || 0;
            const shift3 = parseInt(row.querySelector(`.editable-cell[data-date="${date}"][data-shift="3"] input`)?.value) || 0;
            dayTotalCell.textContent = shift1 + shift2 + shift3;
        }
        
        // Обновляем итого по месяцу
        let monthTotal = 0;
        row.querySelectorAll('.day-total-cell').forEach(cell => {
            monthTotal += parseInt(cell.textContent) || 0;
        });
        const monthTotalCell = row.querySelector('.month-total-cell');
        if (monthTotalCell) monthTotalCell.textContent = monthTotal;
    },
    
    async saveAll() {
        const plansToSave = [];
        
        document.querySelectorAll('#plans-table-body tr').forEach(row => {
            const materialId = row.querySelector('.material-cell')?.dataset.materialId;
            const inputs = row.querySelectorAll('.plan-input');
            
            inputs.forEach(input => {
                const td = input.closest('td');
                const date = td.dataset.date;
                const shift = td.dataset.shift;
                const amount = parseInt(input.value) || 0;
                
                if (amount > 0) {
                    plansToSave.push({
                        planDate: date,
                        shift: shift,
                        materialId: parseInt(materialId),
                        plannedAmount: amount
                    });
                }
            });
        });
        
        const saveBtn = document.getElementById('save-plans');
        const originalText = saveBtn.textContent;
        saveBtn.textContent = '💾 Сохранение...';
        saveBtn.disabled = true;
        
        let successCount = 0;
        let errorCount = 0;
        
        for (const plan of plansToSave) {
            try {
                const existingPlans = await API.getPlans({
                    dateFrom: plan.planDate,
                    dateTo: plan.planDate,
                    materialId: plan.materialId
                });
                
                const existing = existingPlans.find(p => p.shift === plan.shift);
                
                if (existing) {
                    await API.updatePlan(existing.planId, { plannedAmount: plan.plannedAmount });
                } else {
                    await API.createPlan(plan);
                }
                successCount++;
            } catch (error) {
                console.error('Ошибка:', error);
                errorCount++;
            }
        }
        
        saveBtn.textContent = originalText;
        saveBtn.disabled = false;
        
        alert(`Сохранено: ${successCount}\nОшибок: ${errorCount}`);
        await this.render(this.container);
    }
};

window.PlansModule = plansModule;