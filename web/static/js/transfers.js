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
        });

        modalSave.addEventListener('click', onModalSave);
        btnCompleteTransfer.addEventListener('click', onCompleteTransfer);

        loadWarehouses();
        loadTransfers();
    }

    function loadWarehouses() {
        // Попытка получить список складов — если API отсутствует, оставим один вариант
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
            listEl.innerHTML = `<div class="text-muted p-3">Нет за��вок</div>`;
            return;
        }

        transfers.forEach(t => {
            const shipped = t.ShippedQuantity != null ? t.ShippedQuantity : (t.shipped || 0);
            const planned = t.Quantity || t.planned || 0;
            const deviation = shipped - planned;

            const item = document.createElement('a');
            item.className = 'list-group-item list-group-item-action transfer-row';
            item.href = '#';
            item.dataset.id = t.TransferID || t.transferId || t.id;

            const badge = deviation > 0 ? `<span class="badge bg-danger ms-2">+${deviation}</span>` : (deviation < 0 ? `<span class="badge bg-success ms-2">${deviation}</span>` : `<span class="badge bg-secondary ms-2">0</span>`);

            item.innerHTML = `<div class="d-flex w-100 justify-content-between align-items-center">
                <div>
                    <div><strong>${escapeHtml(t.TransferNumber || t.transferNumber || `#${item.dataset.id}`)}</strong></div>
                    <div class="small text-muted">${escapeHtml(t.MaterialCode || t.materialCode || t.MaterialID || t.materialId || '')}</div>
                </div>
                <div class="text-end">
                    <div class="small">${planned} шт.</div>
                    <div class="small">${shipped}/${planned} ${badge}</div>
                    <div class="small text-muted">${escapeHtml(t.Status || t.status || '')}</div>
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
        selectedTransfer = transfers.find(t => String((t.TransferID || t.transferId || t.id)) === String(id)) || { TransferID: id };
        await loadTransferDetails(selectedTransfer.TransferID || selectedTransfer.transferId || selectedTransfer.id);
    }

    function showDetailsEmpty() {
        detailsEl.classList.add('d-none');
        detailsEmptyEl.classList.remove('d-none');
    }

    function showDetails() {
        detailsEl.classList.remove('d-none');
        detailsEmptyEl.classList.add('d-none');
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
        if (!transfer) transfer = transfers.find(t => String((t.TransferID || t.transferId || t.id)) === String(id)) || { TransferID: id };

        // Get shipments — try multiple endpoints
        let shipments = [];
        try {
            // preferred: /api/transfers/{id}/shipments
            let res = await fetch(`/api/transfers/${id}/shipments`);
            if (!res.ok) {
                // fallback: /api/transfer_shipments?transferId=
                res = await fetch(`/api/transfer_shipments?transferId=${encodeURIComponent(id)}`);
            }
            if (res.ok) shipments = await res.json();
        } catch (err) {
            console.warn('shipments load failed', err);
        }

        const planned = transfer.Quantity || transfer.planned || 0;
        const shipped = transfer.ShippedQuantity != null ? transfer.ShippedQuantity : (shipments.reduce((s, x) => s + (x.Quantity || x.quantity || 0), 0));
        const deviation = shipped - planned;

        detailsTitle.textContent = `Заказ: ${transfer.TransferNumber || transfer.transferNumber || ('#' + id)}`;
        detailsMaterial.textContent = transfer.MaterialCode || transfer.materialCode || transfer.MaterialID || transfer.materialId || '';
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

        // Render shipments
        shipmentsTbody.innerHTML = '';
        shipments.forEach(s => {
            const tr = document.createElement('tr');
            const date = formatDate(s.CreatedAt || s.createdAt || s.CreatedAt || s.created || s.created_at);
            const qty = s.Quantity || s.quantity || 0;
            const who = s.CreatedBy || s.createdBy || s.created_by || '';
            tr.innerHTML = `<td>${escapeHtml(date)}</td><td>${qty} шт.</td><td>${escapeHtml(who)}</td>`;
            shipmentsTbody.appendChild(tr);
        });

        // Show/hide Complete button
        btnCompleteTransfer.disabled = !(deviation >= 0);

        showDetails();
    }

    async function onModalSave() {
        modalError.classList.add('d-none');
        const code = inputMaterial.value.trim();
        const qty = parseInt(inputQty.value, 10);
        const who = inputWho.value.trim() || 'Неизвестно';

        if (!selectedTransfer || !selectedTransfer.TransferID && !selectedTransfer.transferId && !selectedTransfer.id) {
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
            TransferID: selectedTransfer.TransferID || selectedTransfer.transferId || selectedTransfer.id,
            MaterialCode: code,
            Quantity: qty,
            CreatedBy: who
        };

        try {
            // try POST to /api/transfer_shipments
            let res = await fetch('/api/transfer_shipments', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (!res.ok) {
                // fallback: /api/transfers/{id}/shipments
                res = await fetch(`/api/transfers/${payload.TransferID}/shipments`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ MaterialCode: code, Quantity: qty, CreatedBy: who })
                });
            }

            if (!res.ok) {
                const txt = await res.text().catch(() => 'Ошибка сервера');
                throw new Error(txt || `HTTP ${res.status}`);
            }

            await loadTransfers();
            await selectTransfer(payload.TransferID);
            bsModal.hide();
        } catch (err) {
            console.error(err);
            modalError.textContent = err.message || 'Ошибка при записи отгрузки';
            modalError.classList.remove('d-none');
        }
    }

    async function onCompleteTransfer() {
        if (!selectedTransfer) return;
        const id = selectedTransfer.TransferID || selectedTransfer.transferId || selectedTransfer.id;
        if (!confirm('Завершить перемещение? Статус станет "Завершена".')) return;

        try {
            // try POST /api/transfers/{id}/complete
            let res = await fetch(`/api/transfers/${id}/complete`, { method: 'POST' });
            if (!res.ok) {
                // fallback: PUT /api/transfers/{id} {status: 'Завершена'}
                res = await fetch(`/api/transfers/${id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ Status: 'Завершена' })
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
