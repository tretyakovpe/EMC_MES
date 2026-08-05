// transfers.js — UI logic for Transfers / warehouse pickers
(function () {
    const listEl = document.getElementById('transfers-list');
    const detailsEl = document.getElementById('details');
    const detailsEmptyEl = document.getElementById('details-empty');
    const detailsTitle = document.getElementById('details-title');
    const detailsMaterial = document.getElementById('details-material');
    const detailsPlanned = document.getElementById('details-planned');
    const detailsShipped = document.getElementById('details-shipped');
    const detailsDeviation = document.getElementById('details-deviation');
    const shipmentsTbody = document.getElementById('shipments-tbody');
    const itemsTbody = document.getElementById('items-tbody');
    const btnStartPick = document.getElementById('btn-start-pick');
    const btnCompleteTransfer = document.getElementById('btn-complete-transfer');

    const modalEl = document.getElementById('modal-add-shipment');
    const inputMaterial = document.getElementById('input-material-code');
    const inputMaterialSelect = document.getElementById('input-material-select');
    const inputQty = document.getElementById('input-qty');
    const inputWho = document.getElementById('input-who');
    const modalError = document.getElementById('modal-error');
    const modalSave = document.getElementById('modal-save');

    let bsModal = null;
    let transfers = [];
    let selectedTransfer = null;
    let currentItems = []; // [{materialId, materialCode, planned, shipped}]

    function init() {
        // init modal
        bsModal = new bootstrap.Modal(modalEl, { backdrop: 'static' });

        document.getElementById('btn-refresh').addEventListener('click', loadTransfers);
        document.getElementById('filter-date').addEventListener('change', loadTransfers);
        document.getElementById('filter-warehouse').addEventListener('change', loadTransfers);

        btnStartPick.addEventListener('click', () => {
            openPickModal();
        });

        modalSave.addEventListener('click', onModalSave);
        btnCompleteTransfer.addEventListener('click', onCompleteTransfer);

        loadWarehouses();
        loadTransfers();
    }

    function loadWarehouses() {
        fetch('/api/warehouses')
            .then(res => { if (!res.ok) throw new Error('no'); return res.json(); })
            .then(data => {
                const sel = document.getElementById('filter-warehouse');
                data.forEach(w => {
                    const opt = document.createElement('option');
                    opt.value = w.warehouseId || w.WarehouseID || w.id || '';
                    opt.textContent = w.name || w.Name || w.title || w.Title || `${opt.value}`;
                    sel.appendChild(opt);
                });
            })
            .catch(() => { /* ignore */ });
    }

    async function loadTransfers() {
        listEl.innerHTML = '';
        selectedTransfer = null;
        showDetailsEmpty();

        const date = document.getElementById('filter-date').value;
        const warehouse = document.getElementById('filter-warehouse').value;
        let qp = [];
        if (date) qp.push(`date=${encodeURIComponent(date)}`);
        if (warehouse) qp.push(`fromWarehouse=${encodeURIComponent(warehouse)}`);
        const url = '/api/transfers' + (qp.length ? '?' + qp.join('&') : '');

        let data = [];
        try {
            const res = await fetch(url);
            if (!res.ok) throw new Error('Не удалось загрузить заявки');
            data = await res.json();
        } catch (err) {
            console.error(err);
            listEl.innerHTML = `<div class="text-danger p-3">Ошибка при загрузке заявок</div>`;
            return;
        }

        transfers = data;
        if (!Array.isArray(transfers)) transfers = [];

        if (transfers.length === 0) {
            listEl.innerHTML = `<div class="text-muted p-3">Нет заявок</div>`;
            return;
        }

        transfers.forEach(t => {
            const shipped = t.shippedQuantity != null ? t.shippedQuantity : (t.shipped || t.ShippedQuantity || 0);
            const planned = t.quantity || t.Quantity || t.planned || 0;
            const deviation = shipped - planned;

            const item = document.createElement('a');
            item.className = 'list-group-item list-group-item-action transfer-row';
            item.href = '#';
            item.dataset.id = t.transferId || t.TransferID || t.id;

            const badge = deviation > 0 ? `<span class="badge bg-warning ms-2">+${deviation}</span>` : (deviation < 0 ? `<span class="badge bg-secondary ms-2">${deviation}</span>` : `<span class="badge bg-success ms-2">0</span>`);

            item.innerHTML = `<div class="d-flex w-100 justify-content-between align-items-center">
                <div>
                    <div><strong>${escapeHtml(t.transferNumber || t.TransferNumber || `#${item.dataset.id}`)}</strong></div>
                    <div class="small text-muted">${escapeHtml(t.materialCode || t.MaterialCode || t.materialDesc || '')}</div>
                </div>
                <div class="text-end">
                    <div class="small">${planned} шт.</div>
                    <div class="small">${shipped}/${planned} ${badge}</div>
                    <div class="small text-muted">${escapeHtml(t.status || t.Status || '')}</div>
                </div>
            </div>`;

            item.addEventListener('click', e => {
                e.preventDefault();
                selectTransfer(item.dataset.id);
                document.querySelectorAll('.transfer-row').forEach(r => r.classList.remove('active'));
                item.classList.add('active');
            });

            listEl.appendChild(item);
        });
    }

    async function selectTransfer(id) {
        selectedTransfer = transfers.find(t => String((t.transferId || t.TransferID || t.id)) === String(id)) || { transferId: id };
        await loadTransferDetails(selectedTransfer.transferId || selectedTransfer.TransferID || selectedTransfer.id);
    }

    function showDetailsEmpty() {
        detailsEl.classList.add('d-none');
        detailsEmptyEl.classList.remove('d-none');
    }

    function showDetails() {
        detailsEl.classList.remove('d-none');
        detailsEmptyEl.classList.add('d-none');
    }

    async function tryFetchJson(url, options) {
        try {
            const res = await fetch(url, options);
            if (!res.ok) return { ok: false, status: res.status, text: await res.text().catch(() => '') };
            const ct = res.headers.get('content-type') || '';
            if (!ct.includes('application/json')) {
                return { ok: false, status: res.status, text: await res.text().catch(() => '') };
            }
            const json = await res.json();
            return { ok: true, json };
        } catch (err) {
            return { ok: false, status: 0, text: String(err) };
        }
    }

    async function loadTransferDetails(id) {
        if (!id) return;
        // Try to get /api/transfers/{id}
        let transfer = null;
        try {
            const res = await fetch(`/api/transfers/${id}`);
            if (res.ok) transfer = await res.json();
        } catch (err) { /* ignore */ }

        // If API didn't return detailed transfer, find in list
        if (!transfer) transfer = transfers.find(t => String((t.transferId || t.TransferID || t.id)) === String(id)) || { transferId: id };

        // Get shipments — try multiple endpoints in order
        let shipments = [];
        try {
            let attempt = await tryFetchJson(`/api/transfers/${id}/shipments`);
            if (attempt.ok) shipments = attempt.json;
            else {
                attempt = await tryFetchJson(`/api/transfer-shipments/${id}`);
                if (attempt.ok) shipments = attempt.json;
                else {
                    attempt = await tryFetchJson(`/api/transfer-shipments?transferId=${encodeURIComponent(id)}`);
                    if (attempt.ok) shipments = attempt.json;
                    else shipments = [];
                }
            }

            // Normalize
            if (!Array.isArray(shipments)) {
                if (Array.isArray(shipments.data)) shipments = shipments.data;
                else if (Array.isArray(shipments.items)) shipments = shipments.items;
                else if (Array.isArray(shipments.shipments)) shipments = shipments.shipments;
                else if (Array.isArray(shipments.result)) shipments = shipments.result;
                else if (shipments && typeof shipments === 'object') {
                    const vals = Object.values(shipments).filter(v => Array.isArray(v));
                    shipments = vals.length > 0 ? vals[0] : [];
                } else shipments = [];
            }
        } catch (err) {
            console.warn('shipments load failed', err);
            shipments = [];
        }

        // Build items list: try transfer.items or single material
        const items = [];
        if (Array.isArray(transfer.items) && transfer.items.length > 0) {
            transfer.items.forEach(it => {
                items.push({
                    materialId: it.materialId || it.MaterialID || it.id,
                    materialCode: it.materialCode || it.MaterialCode || it.code || it.Code,
                    planned: it.quantity || it.Quantity || it.planned || 0
                });
            });
        } else {
            items.push({
                materialId: transfer.materialId || transfer.MaterialID || transfer.material || transfer.Material || null,
                materialCode: transfer.materialCode || transfer.MaterialCode || transfer.materialDesc || transfer.material || '',
                planned: transfer.quantity || transfer.Quantity || 0
            });
        }

        // Aggregate shipped per material
        const shippedBy = {};
        if (Array.isArray(shipments)) {
            shipments.forEach(s => {
                const mid = s.materialId || s.MaterialID || s.materialId || s.material || s.Material || null;
                const q = Number(s.quantity || s.Quantity || s.qty || 0) || 0;
                if (!mid) return;
                shippedBy[mid] = (shippedBy[mid] || 0) + q;
            });
        }

        // Fill currentItems and render table
        currentItems = items.map(it => {
            const shipped = shippedBy[it.materialId] || 0;
            return Object.assign({}, it, { shipped, remaining: it.planned - shipped });
        });

        // Render items table
        itemsTbody.innerHTML = '';
        currentItems.forEach(it => {
            const tr = document.createElement('tr');
            let cls = '';
            if (it.shipped === it.planned) cls = 'table-success';
            else if (it.shipped > it.planned) cls = 'table-warning';
            // remaining could be negative
            tr.className = cls;
            tr.innerHTML = `<td>${escapeHtml(it.materialCode || (it.materialId || ''))}</td>
                <td>${it.planned} шт.</td>
                <td>${it.shipped} шт.</td>
                <td>${Math.max(0, it.planned - it.shipped)} шт.</td>
                <td>${it.shipped === it.planned ? 'Готово' : (it.shipped > it.planned ? 'Перебор' : 'В работе')}</td>`;
            itemsTbody.appendChild(tr);
        });

        // Prepare material select in modal
        if (currentItems.length > 1) {
            inputMaterialSelect.innerHTML = '';
            currentItems.forEach(it => {
                const opt = document.createElement('option');
                opt.value = it.materialId || it.materialCode;
                opt.textContent = `${it.materialCode || it.materialId} — план ${it.planned}`;
                inputMaterialSelect.appendChild(opt);
            });
            inputMaterialSelect.classList.remove('d-none');
            inputMaterial.classList.add('d-none');
        } else {
            inputMaterialSelect.classList.add('d-none');
            inputMaterial.classList.remove('d-none');
            inputMaterial.value = currentItems[0].materialCode || '';
        }

        const plannedTotal = items.reduce((s, it) => s + (it.planned || 0), 0);
        const shippedTotal = Object.values(shippedBy).reduce((s, v) => s + v, 0);
        const deviation = shippedTotal - plannedTotal;

        detailsTitle.textContent = `Заказ: ${transfer.transferNumber || transfer.TransferNumber || ('#' + id)}`;

        // Render shipments history
        shipmentsTbody.innerHTML = '';
        if (Array.isArray(shipments) && shipments.length > 0) {
            shipments.forEach(s => {
                const tr = document.createElement('tr');
                const date = formatDate(s.createdAt || s.CreatedAt || s.created || s.created_at || s.Created_at);
                const qty = s.quantity || s.Quantity || s.qty || 0;
                const who = s.createdBy || s.CreatedBy || s.created_by || s.createdByName || '';
                tr.innerHTML = `<td>${escapeHtml(date)}</td><td>${qty} шт.</td><td>${escapeHtml(who)}</td>`;
                shipmentsTbody.appendChild(tr);
            });
        } else {
            const tr = document.createElement('tr');
            tr.innerHTML = `<td colspan="3" class="text-muted">Нет записей об отгрузках</td>`;
            shipmentsTbody.appendChild(tr);
        }

        // Show/hide Complete button
        btnCompleteTransfer.disabled = !(deviation >= 0);

        showDetails();
    }

    function openPickModal() {
        modalError.classList.add('d-none');
        inputQty.value = 1;
        inputWho.value = '';
        if (currentItems.length > 1) {
            inputMaterialSelect.classList.remove('d-none');
            inputMaterial.classList.add('d-none');
        } else {
            inputMaterialSelect.classList.add('d-none');
            inputMaterial.classList.remove('d-none');
            inputMaterial.value = currentItems[0] ? currentItems[0].materialCode : '';
        }
        bsModal.show();
        setTimeout(() => {
            if (inputMaterial.classList.contains('d-none')) inputMaterialSelect.focus(); else inputMaterial.focus();
        }, 200);
    }

    async function onModalSave() {
        modalError.classList.add('d-none');
        const qty = parseInt(inputQty.value, 10);
        const who = inputWho.value.trim() || 'Неизвестно';

        let selectedMaterialId = null;
        let selectedMaterialCode = null;
        if (currentItems.length > 1) {
            const val = inputMaterialSelect.value;
            // value was set to materialId or materialCode
            selectedMaterialId = isNaN(Number(val)) ? null : Number(val);
            selectedMaterialCode = isNaN(Number(val)) ? val : null;
        } else {
            selectedMaterialCode = inputMaterial.value.trim();
        }

        if (!selectedTransfer || !selectedTransfer.transferId && !selectedTransfer.TransferID && !selectedTransfer.id) {
            modalError.textContent = 'Не выбрана заявка';
            modalError.classList.remove('d-none');
            return;
        }
        if (!qty || qty <= 0) {
            modalError.textContent = 'Количество должно быть положительным';
            modalError.classList.remove('d-none');
            return;
        }

        // Find material in currentItems by id or code
        let found = null;
        if (selectedMaterialId) {
            found = currentItems.find(it => String(it.materialId) === String(selectedMaterialId));
        } else if (selectedMaterialCode) {
            found = currentItems.find(it => (it.materialCode || '').toLowerCase() === selectedMaterialCode.toLowerCase());
            // if not found by code, try fetching material by code to get materialId
        }

        let materialIdToSend = null;
        if (found) materialIdToSend = found.materialId;
        else if (selectedMaterialCode) {
            // try lookup via API
            try {
                const res = await fetch(`/api/materials/code?code=${encodeURIComponent(selectedMaterialCode)}`);
                if (res.ok) {
                    const m = await res.json();
                    materialIdToSend = m.materialId || m.MaterialID || m.id || m.materialId || null;
                }
            } catch (err) { /* ignore */ }
        }

        if (!materialIdToSend) {
            modalError.textContent = 'Материал не найден в накладной';
            modalError.classList.remove('d-none');
            return;
        }

        const payload = {
            transferId: selectedTransfer.transferId || selectedTransfer.TransferID || selectedTransfer.id,
            materialId: materialIdToSend,
            quantity: qty,
            createdBy: who
        };

        try {
            let res = await fetch('/api/transfer-shipments', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (!res.ok) {
                const txt = await res.text().catch(() => 'Ошибка сервера');
                throw new Error(txt || `HTTP ${res.status}`);
            }

            await loadTransfers();
            await selectTransfer(payload.transferId);
            bsModal.hide();
        } catch (err) {
            console.error(err);
            modalError.textContent = err.message || 'Ошибка при записи отгрузки';
            modalError.classList.remove('d-none');
        }
    }

    async function onCompleteTransfer() {
        if (!selectedTransfer) return;
        const id = selectedTransfer.transferId || selectedTransfer.TransferID || selectedTransfer.id;
        if (!confirm('Завершить перемещение? Статус станет "Завершена".')) return;

        try {
            let res = await fetch(`/api/transfers/${id}/complete`, { method: 'POST' });
            if (!res.ok) {
                res = await fetch(`/api/transfers/${id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ status: 'Завершена' })
                });
            }
            if (!res.ok) throw new Error('Не удалось завершить перемещение');
            await loadTransfers();
            alert('Перемещение завершено');
        } catch (err) {
            console.error(err);
            alert('Ошибка при завершении: ' + (err.message || err));
        }
    }

    function formatDate(s) {
        if (!s) return '';
        const d = new Date(s);
        if (isNaN(d.getTime())) return String(s);
        return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    function pad(n) { return n < 10 ? '0' + n : n; }

    function escapeHtml(str) {
        if (str == null) return '';
        return String(str).replace(/[&<>"']/g, function (c) { return ({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":"&#39;" })[c]; });
    }

    // init on DOM ready
    window.addEventListener('DOMContentLoaded', init);

})();
