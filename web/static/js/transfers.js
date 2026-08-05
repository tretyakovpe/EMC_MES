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
    const btnAddShipment = document.getElementById('btn-add-shipment');
    const btnCompleteTransfer = document.getElementById('btn-complete-transfer');

    const modalEl = document.getElementById('modal-add-shipment');
    const inputMaterial = document.getElementById('input-material-code');
    const inputQty = document.getElementById('input-qty');
    const inputWho = document.getElementById('input-who');
    const modalError = document.getElementById('modal-error');
    const modalSave = document.getElementById('modal-save');

    let bsModal = null;
    let transfers = [];
    let selectedTransfer = null;

    function init() {
        // init modal
        bsModal = new bootstrap.Modal(modalEl, { backdrop: 'static' });

        document.getElementById('btn-refresh').addEventListener('click', loadTransfers);
        document.getElementById('filter-date').addEventListener('change', loadTransfers);
        document.getElementById('filter-warehouse').addEventListener('change', loadTransfers);

        btnAddShipment.addEventListener('click', () => {
            modalError.classList.add('d-none');
            inputMaterial.value = '';
            inputQty.value = 1;
            inputWho.value = '';
            bsModal.show();
            // autofocus material input for quick scanning
            setTimeout(() => inputMaterial.focus(), 200);
        });

        modalSave.addEventListener('click', onModalSave);
        btnCompleteTransfer.addEventListener('click', onCompleteTransfer);

        loadWarehouses();
        loadTransfers();
    }

    function loadWarehouses() {
        // Попытка получить спис��к складов — если API отсутствует, оставим один вариант
        fetch('/api/warehouses')
            .then(res => { if (!res.ok) throw new Error('no'); return res.json(); })
            .then(data => {
                const sel = document.getElementById('filter-warehouse');
                data.forEach(w => {
                    const opt = document.createElement('option');
                    opt.value = w.WarehouseID || w.id || w.warehouseId || '';
                    opt.textContent = w.Name || w.name || w.Title || w.title || `${opt.value}`;
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

            const badge = deviation > 0 ? `<span class="badge bg-danger ms-2">+${deviation}</span>` : (deviation < 0 ? `<span class="badge bg-success ms-2">${deviation}</span>` : `<span class="badge bg-secondary ms-2">0</span>`);

            item.innerHTML = `<div class="d-flex w-100 justify-content-between align-items-center">
                <div>
                    <div><strong>${escapeHtml(t.transferNumber || t.TransferNumber || `#${item.dataset.id}`)}</strong></div>
                    <div class="small text-muted">${escapeHtml(t.materialCode || t.MaterialCode || t.materialId || t.MaterialID || '')}</div>
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
            if (!res.ok) return { ok: false };
            const text = await res.text();
            // try parse JSON safely
            try {
                return { ok: true, json: JSON.parse(text) };
            } catch (err) {
                return { ok: false };
            }
        } catch (err) {
            return { ok: false };
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
            // 1) /api/transfers/{id}/shipments
            let attempt = await tryFetchJson(`/api/transfers/${id}/shipments`);
            if (attempt.ok) shipments = attempt.json;
            else {
                // 2) /api/transfer-shipments/{id}
                attempt = await tryFetchJson(`/api/transfer-shipments/${id}`);
                if (attempt.ok) shipments = attempt.json;
                else {
                    // 3) /api/transfer-shipments?transferId={id} (legacy)
                    attempt = await tryFetchJson(`/api/transfer-shipments?transferId=${encodeURIComponent(id)}`);
                    if (attempt.ok) shipments = attempt.json;
                    else shipments = [];
                }
            }

            // Normalize response to array
            if (!Array.isArray(shipments)) {
                if (Array.isArray(shipments.data)) shipments = shipments.data;
                else if (Array.isArray(shipments.items)) shipments = shipments.items;
                else if (Array.isArray(shipments.shipments)) shipments = shipments.shipments;
                else if (Array.isArray(shipments.result)) shipments = shipments.result;
                else if (shipments && typeof shipments === 'object') {
                    const vals = Object.values(shipments).filter(v => Array.isArray(v));
                    if (vals.length > 0) shipments = vals[0];
                    else shipments = [];
                } else {
                    shipments = [];
                }
            }
        } catch (err) {
            console.warn('shipments load failed', err);
            shipments = [];
        }

        const planned = transfer.quantity || transfer.Quantity || transfer.planned || 0;
        // shipped: if transfer has explicit shippedQuantity use it; otherwise sum shipments safely
        let shipped = 0;
        if (transfer.shippedQuantity != null) shipped = transfer.shippedQuantity;
        else if (transfer.ShippedQuantity != null) shipped = transfer.ShippedQuantity;
        else if (Array.isArray(shipments)) shipped = shipments.reduce((s, x) => s + (Number(x.quantity || x.Quantity || x.qty || 0) || 0), 0);
        else shipped = 0;
        const deviation = shipped - planned;

        detailsTitle.textContent = `Заказ: ${transfer.transferNumber || transfer.TransferNumber || ('#' + id)}`;
        detailsMaterial.textContent = transfer.materialCode || transfer.MaterialCode || transfer.materialId || transfer.MaterialID || '';
        detailsPlanned.textContent = planned + ' шт.';
        detailsShipped.textContent = shipped + ' шт.';

        detailsDeviation.className = 'badge';
        if (deviation > 0) {
            detailsDeviation.classList.add('badge-deviation-positive');
            detailsDeviation.textContent = '+' + deviation + ' шт.';
        } else if (deviation < 0) {
            detailsDeviation.classList.add('badge-deviation-negative');
            detailsDeviation.textContent = deviation + ' шт.';
        } else {
            detailsDeviation.classList.add('bg-secondary');
            detailsDeviation.textContent = '0';
        }

        // Render shipments (defensive)
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

    async function onModalSave() {
        modalError.classList.add('d-none');
        const code = inputMaterial.value.trim();
        const qty = parseInt(inputQty.value, 10);
        const who = inputWho.value.trim() || 'Неизвестно';

        if (!selectedTransfer || !selectedTransfer.transferId && !selectedTransfer.TransferID && !selectedTransfer.id) {
            modalError.textContent = 'Не выбрана заявка';
            modalError.classList.remove('d-none');
            return;
        }
        if (!code) {
            modalError.textContent = 'Укажите код материала';
            modalError.classList.remove('d-none');
            return;
        }
        if (!qty || qty <= 0) {
            modalError.textContent = 'Количество должно быть положительным';
            modalError.classList.remove('d-none');
            return;
        }

        const payload = {
            transferId: selectedTransfer.transferId || selectedTransfer.TransferID || selectedTransfer.id,
            materialCode: code,
            quantity: qty,
            createdBy: who
        };

        try {
            // try POST to /api/transfer-shipments (server expects JSON with transferId/materialId or materialCode depending on implementation)
            let res = await fetch('/api/transfer-shipments', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) {
                // fallback: /api/transfers/{id}/shipments
                res = await fetch(`/api/transfers/${payload.transferId}/shipments`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ materialCode: code, quantity: qty, createdBy: who })
                });
            }

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
            // try POST /api/transfers/{id}/complete
            let res = await fetch(`/api/transfers/${id}/complete`, { method: 'POST' });
            if (!res.ok) {
                // fallback: PUT /api/transfers/{id} {status: 'Завер��ена'}
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
