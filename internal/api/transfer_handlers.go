package api

import (
	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ================================================================
// СТРУКТУРЫ ЗАПРОСОВ/ОТВЕТОВ
// ================================================================

// CreateTransferOrderRequest запрос на создание заказа
type CreateTransferOrderRequest struct {
	Number          int                              `json:"number"`
	FromWarehouseID int                              `json:"fromWarehouseId"`
	ToWarehouseID   int                              `json:"toWarehouseId"`
	PlannedDate     string                           `json:"plannedDate"` // YYYY-MM-DD
	Details         []CreateTransferOrderDetailInput `json:"details"`
}

// CreateTransferOrderDetailInput деталь заказа при создании
type CreateTransferOrderDetailInput struct {
	MaterialCode string `json:"materialCode"`
	Quantity     int    `json:"quantity"`
}

// UpdateTransferOrderRequest запрос на обновление заказа
type UpdateTransferOrderRequest struct {
	Number          int                              `json:"number"`
	FromWarehouseID int                              `json:"fromWarehouseId"`
	ToWarehouseID   int                              `json:"toWarehouseId"`
	PlannedDate     string                           `json:"plannedDate"` // YYYY-MM-DD
	Details         []CreateTransferOrderDetailInput `json:"details"`
}

// AddShipmentRequest запрос на добавление отгрузки
type AddShipmentRequest struct {
	MaterialCode string `json:"materialCode"`
	Quantity     int    `json:"quantity"`
	CreatedBy    string `json:"createdBy"`
}

// TransferOrderListResponse список заказов с краткой информацией
type TransferOrderListResponse struct {
	TransferOrderID   int    `json:"transferOrderId"`
	Number            int    `json:"number"`
	Date              string `json:"date"`
	PlannedDate       string `json:"plannedDate"`
	FromWarehouseID   int    `json:"fromWarehouseId"`
	FromWarehouseCode string `json:"fromWarehouseCode"`
	FromWarehouseName string `json:"fromWarehouseName"`
	ToWarehouseID     int    `json:"toWarehouseId"`
	ToWarehouseCode   string `json:"toWarehouseCode"`
	ToWarehouseName   string `json:"toWarehouseName"`
	Status            string `json:"status"`
	StatusLabel       string `json:"statusLabel"`
	TotalItems        int    `json:"totalItems"`
	TotalQuantity     int    `json:"totalQuantity"`
	ShippedQuantity   int    `json:"shippedQuantity"`
	Progress          int    `json:"progress"` // 0-100
}

// TransferOrderDetailResponse полная информация о заказе
type TransferOrderDetailResponse struct {
	TransferOrderID   int     `json:"transferOrderId"`
	Number            int     `json:"number"`
	Date              string  `json:"date"`
	PlannedDate       string  `json:"plannedDate"`
	FromWarehouseID   int     `json:"fromWarehouseId"`
	FromWarehouseCode string  `json:"fromWarehouseCode"`
	FromWarehouseName string  `json:"fromWarehouseName"`
	ToWarehouseID     int     `json:"toWarehouseId"`
	ToWarehouseCode   string  `json:"toWarehouseCode"`
	ToWarehouseName   string  `json:"toWarehouseName"`
	Status            string  `json:"status"`
	StatusLabel       string  `json:"statusLabel"`
	StatusChangedAt   *string `json:"statusChangedAt,omitempty"`
	StatusChangedBy   *string `json:"statusChangedBy,omitempty"`
	StartedAt         *string `json:"startedAt,omitempty"`
	StartedBy         *string `json:"startedBy,omitempty"`
	CompletedAt       *string `json:"completedAt,omitempty"`
	CompletedBy       *string `json:"completedBy,omitempty"`
	CreatedBy         *string `json:"createdBy,omitempty"`
	// Права на действия
	CanEdit        bool `json:"canEdit"`        // можно редактировать (Draft)
	CanDelete      bool `json:"canDelete"`      // можно удалить (Draft)
	CanStart       bool `json:"canStart"`       // можно начать сборку (Draft)
	CanAddShipment bool `json:"canAddShipment"` // можно добавить отгрузку (InProgress)
	CanConfirm     bool `json:"canConfirm"`     // можно подтвердить (Ready)
	// Детали и отгрузки
	Details   []TransferOrderDetailItem   `json:"details"`
	Shipments []TransferOrderShipmentItem `json:"shipments"`
}

// TransferOrderDetailItem деталь заказа в ответе
type TransferOrderDetailItem struct {
	MaterialID      int    `json:"materialId"`
	MaterialCode    string `json:"materialCode"`
	Description     string `json:"description"`
	Quantity        int    `json:"quantity"`
	ShippedQuantity int    `json:"shippedQuantity"`
	Remaining       int    `json:"remaining"`
	IsFullyShipped  bool   `json:"isFullyShipped"`
}

// TransferOrderShipmentItem отгрузка в ответе
type TransferOrderShipmentItem struct {
	TransferOrderShipmentID int    `json:"transferOrderShipmentId"`
	MaterialID              int    `json:"materialId"`
	MaterialCode            string `json:"materialCode"`
	MaterialDescription     string `json:"materialDescription"`
	Quantity                int    `json:"quantity"`
	CreatedAt               string `json:"createdAt"`
	CreatedBy               string `json:"createdBy"`
}

// ================================================================
// ГЛАВНЫЙ ОБРАБОТЧИК /api/transfer-orders
// ================================================================

// handleTransferOrders обрабатывает запросы к /api/transfer-orders
func handleTransferOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetTransferOrders(w, r)
	case http.MethodPost:
		handleCreateTransferOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ================================================================
// GET /api/transfer-orders - список заказов
// ================================================================

func handleGetTransferOrders(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	var fromWarehouseID *int
	if s := r.URL.Query().Get("fromWarehouse"); s != "" {
		id, err := strconv.Atoi(s)
		if err == nil {
			fromWarehouseID = &id
		}
	}

	var toWarehouseID *int
	if s := r.URL.Query().Get("toWarehouse"); s != "" {
		id, err := strconv.Atoi(s)
		if err == nil {
			toWarehouseID = &id
		}
	}

	var fromDate *time.Time
	if s := r.URL.Query().Get("fromDate"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			fromDate = &t
		}
	}

	var toDate *time.Time
	if s := r.URL.Query().Get("toDate"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			toDate = &t
		}
	}

	orders, err := database.GetTransferOrders(status, fromWarehouseID, toWarehouseID, fromDate, toDate)
	if err != nil {
		logger.Error("API /api/transfer-orders: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := make([]TransferOrderListResponse, 0, len(orders))
	for _, o := range orders {
		// Загружаем детали для подсчёта прогресса
		fullOrder, _ := database.GetTransferOrderByID(o.TransferOrderID)

		totalItems := 0
		totalQuantity := 0
		shippedQuantity := 0

		if fullOrder != nil {
			totalItems = len(fullOrder.Details)
			for _, d := range fullOrder.Details {
				totalQuantity += d.Quantity
				shippedQuantity += d.ShippedQuantity
			}
		}

		progress := 0
		if totalQuantity > 0 {
			progress = int(float64(shippedQuantity) / float64(totalQuantity) * 100)
		}

		response = append(response, TransferOrderListResponse{
			TransferOrderID:   o.TransferOrderID,
			Number:            o.Number,
			Date:              o.Date.Format("2006-01-02 15:04:05"),
			PlannedDate:       o.PlannedDate.Format("2006-01-02"),
			FromWarehouseID:   o.FromWarehouseID,
			FromWarehouseCode: o.FromWarehouseCode,
			FromWarehouseName: o.FromWarehouseName,
			ToWarehouseID:     o.ToWarehouseID,
			ToWarehouseCode:   o.ToWarehouseCode,
			ToWarehouseName:   o.ToWarehouseName,
			Status:            o.Status,
			StatusLabel:       getStatusLabel(o.Status),
			TotalItems:        totalItems,
			TotalQuantity:     totalQuantity,
			ShippedQuantity:   shippedQuantity,
			Progress:          progress,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ================================================================
// POST /api/transfer-orders - создание заказа
// ================================================================

func handleCreateTransferOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateTransferOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("API /api/transfer-orders (create): ошибка парсинга: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Number <= 0 {
		http.Error(w, "number is required and must be positive", http.StatusBadRequest)
		return
	}
	if req.FromWarehouseID <= 0 {
		http.Error(w, "fromWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.ToWarehouseID <= 0 {
		http.Error(w, "toWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		http.Error(w, "fromWarehouseId and toWarehouseId cannot be the same", http.StatusBadRequest)
		return
	}
	if req.PlannedDate == "" {
		http.Error(w, "plannedDate is required", http.StatusBadRequest)
		return
	}
	if len(req.Details) == 0 {
		http.Error(w, "at least one detail is required", http.StatusBadRequest)
		return
	}

	plannedDate, err := time.Parse("2006-01-02", req.PlannedDate)
	if err != nil {
		http.Error(w, "invalid plannedDate format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	for _, d := range req.Details {
		if d.MaterialCode == "" {
			http.Error(w, "materialCode is required for each detail", http.StatusBadRequest)
			return
		}
		if d.Quantity <= 0 {
			http.Error(w, "quantity must be greater than 0 for each detail", http.StatusBadRequest)
			return
		}
	}

	// Кто создаёт (из заголовка или по умолчанию)
	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		createdBy = "system"
	}

	// Подготавливаем детали
	details := make([]database.TransferOrderDetailInput, 0, len(req.Details))
	for _, d := range req.Details {
		details = append(details, database.TransferOrderDetailInput{
			MaterialCode: d.MaterialCode,
			Quantity:     d.Quantity,
		})
	}

	orderID, err := database.CreateTransferOrder(
		req.Number,
		req.FromWarehouseID,
		req.ToWarehouseID,
		plannedDate,
		details,
		createdBy,
	)
	if err != nil {
		logger.Error("API /api/transfer-orders (create): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_created", map[string]interface{}{
			"transferOrderId": orderID,
			"number":          req.Number,
			"status":          "Draft",
		})
	}

	logger.Info("API: Создан заказ №%d (ID=%d)", req.Number, orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"message":         "Transfer order created",
		"transferOrderId": orderID,
		"number":          req.Number,
	})
}

// ================================================================
// ОБРАБОТЧИК /api/transfer-orders/{id}
// ================================================================

// handleTransferOrderByID обрабатывает запросы к /api/transfer-orders/{id}
func handleTransferOrderByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-orders/")
	idStr := strings.Split(path, "/")[0]

	orderID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transfer order ID", http.StatusBadRequest)
		return
	}

	// Проверяем подпути
	if len(strings.Split(path, "/")) > 1 {
		subPath := strings.Split(path, "/")[1]
		switch subPath {
		case "start":
			if r.Method == http.MethodPost {
				handleStartTransferOrder(w, r, orderID)
				return
			}
		case "confirm":
			if r.Method == http.MethodPost {
				handleConfirmTransferOrder(w, r, orderID)
				return
			}
		}
	}

	switch r.Method {
	case http.MethodGet:
		handleGetTransferOrderByID(w, r, orderID)
	case http.MethodPut:
		handleUpdateTransferOrder(w, r, orderID)
	case http.MethodDelete:
		handleDeleteTransferOrder(w, r, orderID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ================================================================
// GET /api/transfer-orders/{id} - получение заказа
// ================================================================

func handleGetTransferOrderByID(w http.ResponseWriter, r *http.Request, orderID int) {
	order, err := database.GetTransferOrderByID(orderID)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d: %v", orderID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if order == nil {
		http.Error(w, "Transfer order not found", http.StatusNotFound)
		return
	}

	resp := TransferOrderDetailResponse{
		TransferOrderID:   order.TransferOrderID,
		Number:            order.Number,
		Date:              order.Date.Format("2006-01-02 15:04:05"),
		PlannedDate:       order.PlannedDate.Format("2006-01-02"),
		FromWarehouseID:   order.FromWarehouseID,
		FromWarehouseCode: order.FromWarehouseCode,
		FromWarehouseName: order.FromWarehouseName,
		ToWarehouseID:     order.ToWarehouseID,
		ToWarehouseCode:   order.ToWarehouseCode,
		ToWarehouseName:   order.ToWarehouseName,
		Status:            order.Status,
		StatusLabel:       getStatusLabel(order.Status),
		CanEdit:           order.Status == "Draft",
		CanDelete:         order.Status == "Draft",
		CanStart:          order.Status == "Draft",
		CanAddShipment:    order.Status == "InProgress",
		CanConfirm:        order.Status == "Ready",
	}

	if order.StatusChangedAt != nil {
		s := order.StatusChangedAt.Format("2006-01-02 15:04:05")
		resp.StatusChangedAt = &s
	}
	if order.StatusChangedBy != nil {
		resp.StatusChangedBy = order.StatusChangedBy
	}
	if order.StartedAt != nil {
		s := order.StartedAt.Format("2006-01-02 15:04:05")
		resp.StartedAt = &s
	}
	if order.StartedBy != nil {
		resp.StartedBy = order.StartedBy
	}
	if order.CompletedAt != nil {
		s := order.CompletedAt.Format("2006-01-02 15:04:05")
		resp.CompletedAt = &s
	}
	if order.CompletedBy != nil {
		resp.CompletedBy = order.CompletedBy
	}
	if order.CreatedBy != nil {
		resp.CreatedBy = order.CreatedBy
	}

	// Детали
	resp.Details = make([]TransferOrderDetailItem, 0, len(order.Details))
	for _, d := range order.Details {
		resp.Details = append(resp.Details, TransferOrderDetailItem{
			MaterialID:      d.MaterialID,
			MaterialCode:    d.MaterialCode,
			Description:     d.Description,
			Quantity:        d.Quantity,
			ShippedQuantity: d.ShippedQuantity,
			Remaining:       d.Quantity - d.ShippedQuantity,
			IsFullyShipped:  d.ShippedQuantity >= d.Quantity,
		})
	}

	// Отгрузки
	resp.Shipments = make([]TransferOrderShipmentItem, 0, len(order.Shipments))
	for _, s := range order.Shipments {
		createdBy := ""
		if s.CreatedBy != nil {
			createdBy = *s.CreatedBy
		}
		resp.Shipments = append(resp.Shipments, TransferOrderShipmentItem{
			TransferOrderShipmentID: s.TransferOrderShipmentID,
			MaterialID:              s.MaterialID,
			MaterialCode:            s.MaterialCode,
			MaterialDescription:     s.MaterialDescription,
			Quantity:                s.Quantity,
			CreatedAt:               s.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:               createdBy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ================================================================
// PUT /api/transfer-orders/{id} - обновление заказа
// ================================================================

func handleUpdateTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	var req UpdateTransferOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("API /api/transfer-orders/%d (update): ошибка парсинга: %v", orderID, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Number <= 0 {
		http.Error(w, "number is required and must be positive", http.StatusBadRequest)
		return
	}
	if req.FromWarehouseID <= 0 {
		http.Error(w, "fromWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.ToWarehouseID <= 0 {
		http.Error(w, "toWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		http.Error(w, "fromWarehouseId and toWarehouseId cannot be the same", http.StatusBadRequest)
		return
	}
	if req.PlannedDate == "" {
		http.Error(w, "plannedDate is required", http.StatusBadRequest)
		return
	}
	if len(req.Details) == 0 {
		http.Error(w, "at least one detail is required", http.StatusBadRequest)
		return
	}

	plannedDate, err := time.Parse("2006-01-02", req.PlannedDate)
	if err != nil {
		http.Error(w, "invalid plannedDate format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	for _, d := range req.Details {
		if d.MaterialCode == "" {
			http.Error(w, "materialCode is required for each detail", http.StatusBadRequest)
			return
		}
		if d.Quantity <= 0 {
			http.Error(w, "quantity must be greater than 0 for each detail", http.StatusBadRequest)
			return
		}
	}

	updatedBy := r.Header.Get("X-User")
	if updatedBy == "" {
		updatedBy = "system"
	}

	details := make([]database.TransferOrderDetailInput, 0, len(req.Details))
	for _, d := range req.Details {
		details = append(details, database.TransferOrderDetailInput{
			MaterialCode: d.MaterialCode,
			Quantity:     d.Quantity,
		})
	}

	err = database.UpdateTransferOrder(
		orderID,
		req.Number,
		req.FromWarehouseID,
		req.ToWarehouseID,
		plannedDate,
		details,
		updatedBy,
	)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d (update): %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_updated", map[string]interface{}{
			"transferOrderId": orderID,
			"number":          req.Number,
		})
	}

	logger.Info("API: Обновлён заказ ID=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order updated",
	})
}

// ================================================================
// DELETE /api/transfer-orders/{id} - удаление заказа
// ================================================================

func handleDeleteTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	err := database.DeleteTransferOrder(orderID)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d (delete): %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_deleted", map[string]interface{}{
			"transferOrderId": orderID,
		})
	}

	logger.Info("API: Удалён заказ ID=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order deleted",
	})
}

// ================================================================
// POST /api/transfer-orders/{id}/start - начало сборки
// ================================================================

func handleStartTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	startedBy := r.Header.Get("X-User")
	if startedBy == "" {
		startedBy = "system"
	}

	err := database.StartTransferOrder(orderID, startedBy)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d/start: %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_started", map[string]interface{}{
			"transferOrderId": orderID,
			"startedBy":       startedBy,
		})
	}

	logger.Info("API: Начата сборка заказа ID=%d, пользователь %s", orderID, startedBy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order started",
	})
}

// ================================================================
// POST /api/transfer-orders/{id}/confirm - подтверждение заказа
// ================================================================

func handleConfirmTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	confirmedBy := r.Header.Get("X-User")
	if confirmedBy == "" {
		confirmedBy = "system"
	}

	err := database.ConfirmTransferOrder(orderID, confirmedBy)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d/confirm: %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_confirmed", map[string]interface{}{
			"transferOrderId": orderID,
			"confirmedBy":     confirmedBy,
		})
	}

	logger.Info("API: Подтверждён заказ ID=%d, пользователь %s", orderID, confirmedBy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order confirmed",
	})
}

// ================================================================
// ОБРАБОТЧИКИ ОТГРУЗОК /api/transfer-shipments
// ================================================================

// handleTransferShipments обрабатывает запросы к /api/transfer-shipments
func handleTransferShipments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleAddTransferShipment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ================================================================
// POST /api/transfer-shipments - добавление отгрузки
// ================================================================

func handleAddTransferShipment(w http.ResponseWriter, r *http.Request) {
	var req AddShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("API /api/transfer-shipments (add): ошибка парсинга: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.MaterialCode == "" {
		http.Error(w, "materialCode is required", http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	// Получаем orderID из query-параметра
	orderIDStr := r.URL.Query().Get("orderId")
	if orderIDStr == "" {
		http.Error(w, "orderId query parameter is required", http.StatusBadRequest)
		return
	}
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "invalid orderId", http.StatusBadRequest)
		return
	}

	createdBy := req.CreatedBy
	if createdBy == "" {
		createdBy = r.Header.Get("X-User")
	}
	if createdBy == "" {
		createdBy = "system"
	}

	shipment, err := database.AddTransferShipment(orderID, req.MaterialCode, req.Quantity, createdBy)
	if err != nil {
		logger.Error("API /api/transfer-shipments (add): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_shipment_added", map[string]interface{}{
			"transferOrderId": orderID,
			"materialCode":    req.MaterialCode,
			"quantity":        req.Quantity,
			"createdBy":       createdBy,
		})
	}

	logger.Info("API: Добавлена отгрузка %d шт. материала %s в заказ ID=%d",
		req.Quantity, req.MaterialCode, orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "ok",
		"message":      "Shipment added",
		"shipmentId":   shipment.TransferOrderShipmentID,
		"orderId":      orderID,
		"materialCode": shipment.MaterialCode,
		"quantity":     shipment.Quantity,
	})
}

// ================================================================
// ОБРАБОТЧИК /api/transfer-shipments/{id}
// ================================================================

// handleTransferShipmentByID обрабатывает запросы к /api/transfer-shipments/{id}
func handleTransferShipmentByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-shipments/")
	idStr := strings.Split(path, "/")[0]

	shipmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		handleDeleteShipment(w, r, shipmentID)
	case http.MethodGet:
		handleGetShipmentByID(w, r, shipmentID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ================================================================
// DELETE /api/transfer-shipments/{id} - удаление отгрузки
// ================================================================

func handleDeleteTransferShipment(w http.ResponseWriter, r *http.Request, shipmentID int) {
	err := database.DeleteTransferShipment(shipmentID)
	if err != nil {
		logger.Error("API /api/transfer-shipments/%d (delete): %v", shipmentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_shipment_deleted", map[string]interface{}{
			"shipmentId": shipmentID,
		})
	}

	logger.Info("API: Удалена отгрузка ID=%d", shipmentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Shipment deleted",
	})
}

// ================================================================
// GET /api/transfer-shipments/{id} - получение отгрузки
// ================================================================

func handleGetTransferShipmentByID(w http.ResponseWriter, r *http.Request, shipmentID int) {
	shipment, err := database.GetTransferShipmentByID(shipmentID)
	if err != nil {
		logger.Error("API /api/transfer-shipments/%d: %v", shipmentID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if shipment == nil {
		http.Error(w, "Shipment not found", http.StatusNotFound)
		return
	}

	createdBy := ""
	if shipment.CreatedBy != nil {
		createdBy = *shipment.CreatedBy
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transferOrderShipmentId": shipment.TransferOrderShipmentID,
		"transferOrderId":         shipment.TransferOrderID,
		"materialId":              shipment.MaterialID,
		"materialCode":            shipment.MaterialCode,
		"materialDescription":     shipment.MaterialDescription,
		"quantity":                shipment.Quantity,
		"createdAt":               shipment.CreatedAt.Format("2006-01-02 15:04:05"),
		"createdBy":               createdBy,
	})
}

// ================================================================
// GET /api/transfer-shipments - список отгрузок по заказу
// ================================================================

// handleGetShipmentsByOrder возвращает все отгрузки по заказу
func handleGetTransferShipmentsByOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := r.URL.Query().Get("orderId")
	if orderIDStr == "" {
		http.Error(w, "orderId query parameter is required", http.StatusBadRequest)
		return
	}
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "invalid orderId", http.StatusBadRequest)
		return
	}

	shipments, err := database.GetTransferShipmentsByOrderID(orderID)
	if err != nil {
		logger.Error("API /api/transfer-shipments (list): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(shipments))
	for _, s := range shipments {
		createdBy := ""
		if s.CreatedBy != nil {
			createdBy = *s.CreatedBy
		}
		response = append(response, map[string]interface{}{
			"transferOrderShipmentId": s.TransferOrderShipmentID,
			"transferOrderId":         s.TransferOrderID,
			"materialId":              s.MaterialID,
			"materialCode":            s.MaterialCode,
			"materialDescription":     s.MaterialDescription,
			"quantity":                s.Quantity,
			"createdAt":               s.CreatedAt.Format("2006-01-02 15:04:05"),
			"createdBy":               createdBy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ================================================================

// getStatusLabel возвращает человекочитаемый статус
func getStatusLabel(status string) string {
	labels := map[string]string{
		"Draft":      "Создан",
		"InProgress": "В работе",
		"Ready":      "Готов",
		"Completed":  "Завершен",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}
