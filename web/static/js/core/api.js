// API клиент для взаимодействия с бэкендом
const API = {
    // Базовый метод для GET запросов
    async get(endpoint) {
        const response = await fetch(`/api/${endpoint}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
    },

    // Базовый метод для POST запросов
    async post(endpoint, data = {}) {
        const response = await fetch(`/api/${endpoint}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
    },

    // Базовый метод для PUT запросов
    async put(endpoint, data = {}) {
        const response = await fetch(`/api/${endpoint}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
    },

    // Базовый метод для DELETE запросов
    async delete(endpoint) {
        const response = await fetch(`/api/${endpoint}`, {
            method: 'DELETE'
        });
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
    },

    // ========== ЛИНИИ ==========
    getLines() {
        return this.get('lines');
    },

    getLineStatus(lineName) {
        return this.get(`lines/status?name=${encodeURIComponent(lineName)}`);
    },

    setLineStatus(lineName, isOnline) {
        return this.post(`lines/status?name=${encodeURIComponent(lineName)}`, { isOnline });
    },

    // ========== МАТЕРИАЛЫ ==========
    getMaterials() {
        return this.get('materials');
    },

    getMaterialByCode(code) {
        return this.get(`materials/code?code=${encodeURIComponent(code)}`);
    },

    // ========== КОРОБКИ ==========
    getBoxes(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`boxes${queryParams ? '?' + queryParams : ''}`);
    },

    getBoxByLabel(label) {
        return this.get(`boxes/${encodeURIComponent(label)}`);
    },

    getBoxStats(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`boxes/stats${queryParams ? '?' + queryParams : ''}`);
    },

    // ========== ПЛАНЫ ==========
    getPlans(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`plans${queryParams ? '?' + queryParams : ''}`);
    },

    createPlan(planData) {
        return this.post('plans', planData);
    },

    updatePlan(planId, planData) {
        return this.put(`plans/${planId}`, planData);
    },

    deletePlan(planId) {
        return this.delete(`plans/${planId}`);
    },

    getPlanVolumes() {
        return this.get('plans/volumes');
    },

    updatePlansStatus(planDate = null) {
        return this.post('plans/status', planDate ? { planDate } : {});
    },

    // ========== ОТГРУЗКИ ==========
    getShipments(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`shipments${queryParams ? '?' + queryParams : ''}`);
    },

    getShipmentById(id) {
        return this.get(`shipments/${id}`);
    },

    createShipment(shipmentData) {
        return this.post('shipments', shipmentData);
    },

    scanBox(shipmentId, huId, materialId) {
        return this.post(`shipments/${shipmentId}/scan`, { huId, materialId });
    },

    completeShipment(shipmentId) {
        return this.post(`shipments/${shipmentId}/complete`);
    },

    deleteShipment(shipmentId) {
        return this.delete(`shipments/${shipmentId}`);
    },

    // ========== СТАТИСТИКА ==========
    getStatisticsLines() {
        return this.get('statistics/lines');
    },

    getStatisticsBoxes(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`statistics/boxes${queryParams ? '?' + queryParams : ''}`);
    },

    getStatisticsBadParts(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`statistics/bad-parts${queryParams ? '?' + queryParams : ''}`);
    },

    // ========== ОБЩАЯ СТАТИСТИКА ==========
    getStats(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`stats${queryParams ? '?' + queryParams : ''}`);
    },

    getStatsSummary() {
        return this.get('stats/summary');
    },

    getStatsProduction(params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        return this.get(`stats/production${queryParams ? '?' + queryParams : ''}`);
    }
};