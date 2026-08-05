package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// TransferOrderRequest структура запроса для создания/обновления заказа
type TransferOrderRequest struct {
	Number          int                          `json:"number"`
	FromWarehouseID int                          `json:"fromWarehouseId"`
	ToWarehouseID   int                          `json:"toWarehouseId"`
	PlannedDate     string                       `json:"plannedDate"`
	Details         []TransferOrderDetailRequest `json:"details"`
}

// TransferOrderDetailRequest структура детали заказа
type TransferOrderDetailRequest struct {
	MaterialID int `json:"materialId"`
	Quantity   int `json:"quantity"`
}

// TransferOrderResponse структура ответа
type TransferOrderResponse struct {
	TransferOrderID   int                           `json:"transferOrderId"`
	Number            int                           `json:"number"`
	Date              string                        `json:"date"`
	PlannedDate       string                        `json:"plannedDate"`
	FromWarehouseID   int                           `json:"fromWarehouseId"`
	FromWarehouseCode string                        `json:"fromWarehouseCode"`
	FromWarehouseName string                        `json:"fromWarehouseName"`
	ToWarehouseID     int                           `json:"toWarehouseId"`
	ToWarehouseCode   string                        `json:"toWarehouseCode"`
	ToWarehouseName   string                        `json:"toWarehouseName"`
	Completed         bool                          `json:"completed"`
	Details           []TransferOrderDetailResponse `json:"details"`
}

// TransferOrderDetailResponse структура ответа для детали
type TransferOrderDetailResponse struct {
	MaterialID   int    `json:"materialId"`
	MaterialCode string `json:"materialCode"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
}

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

// handleGetTransferOrders возвращает список заказов
func handleGetTransferOrders(w http.ResponseWriter, r *http.Request) {
	var completed *bool

	if r.URL.Query().Get("completed") != "" {
		b, err := strconv.ParseBool(r.URL.Query().Get("completed"))
		if err == nil {
			completed = &b
		}
	}

	fromDateStr := r.URL.Query().Get("fromDate")
	toDateStr := r.URL.Query().Get("toDate")

	var fromDate, toDate *time.Time
	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			fromDate = &t
		}
	}
	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			toDate = &t
		}
	}

	orders, err := database.GetTransferOrders(completed, fromDate, toDate)
	if err != nil {
		logger.Error("API /api/transfer-orders: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]TransferOrderResponse, 0, len(orders))
	for _, o := range orders {
		resp := TransferOrderResponse{
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
			Completed:         o.Completed,
			Details:           make([]TransferOrderDetailResponse, 0),
		}

		for _, d := range o.Details {
			resp.Details = append(resp.Details, TransferOrderDetailResponse{
				MaterialID:   d.MaterialID,
				MaterialCode: d.MaterialCode,
				Description:  d.Description,
				Quantity:     d.Quantity,
			})
		}

		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateTransferOrder создаёт новый заказ
func handleCreateTransferOrder(w http.ResponseWriter, r *http.Request) {
	var req TransferOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.FromWarehouseID == 0 {
		http.Error(w, "fromWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.ToWarehouseID == 0 {
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
		if d.MaterialID == 0 {
			http.Error(w, "materialId is required for each detail", http.StatusBadRequest)
			return
		}
		if d.Quantity <= 0 {
			http.Error(w, "quantity must be greater than 0 for each detail", http.StatusBadRequest)
			return
		}
	}

	// Генерируем номер заказа
	number, err := database.GenerateTransferOrderNumber()
	if err != nil {
		logger.Error("API /api/transfer-orders (generate number): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Подготавливаем детали
	details := make([]database.TransferOrderDetail, 0, len(req.Details))
	for _, d := range req.Details {
		details = append(details, database.TransferOrderDetail{
			MaterialID: d.MaterialID,
			Quantity:   d.Quantity,
		})
	}

	orderID, err := database.CreateTransferOrder(
		number,
		req.FromWarehouseID,
		req.ToWarehouseID,
		plannedDate,
		details,
	)
	if err != nil {
		logger.Error("API /api/transfer-orders (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_created", map[string]interface{}{
			"transferOrderId": orderID,
			"number":          number,
		})
	}

	logger.Info("API: Создан заказ на перемещение №%d (ID=%d)", number, orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"message":         "Transfer order created",
		"transferOrderId": orderID,
		"number":          number,
	})
}

// handleTransferOrderByID обрабатывает запросы к /api/transfer-orders/{id}
func handleTransferOrderByID(w http.ResponseWriter, r *http.Request) {
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
		case "complete":
			if r.Method == http.MethodPut {
				handleCompleteTransferOrder(w, r, orderID)
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

// handleGetTransferOrderByID возвращает заказ по ID
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

	resp := TransferOrderResponse{
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
		Completed:         order.Completed,
		Details:           make([]TransferOrderDetailResponse, 0),
	}

	for _, d := range order.Details {
		resp.Details = append(resp.Details, TransferOrderDetailResponse{
			MaterialID:   d.MaterialID,
			MaterialCode: d.MaterialCode,
			Description:  d.Description,
			Quantity:     d.Quantity,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateTransferOrder обновляет заказ
func handleUpdateTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	var req TransferOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.FromWarehouseID == 0 {
		http.Error(w, "fromWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.ToWarehouseID == 0 {
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
		if d.MaterialID == 0 {
			http.Error(w, "materialId is required for each detail", http.StatusBadRequest)
			return
		}
		if d.Quantity <= 0 {
			http.Error(w, "quantity must be greater than 0 for each detail", http.StatusBadRequest)
			return
		}
	}

	// Проверяем, что заказ существует и не завершен
	order, err := database.GetTransferOrderByID(orderID)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d (update): %v", orderID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if order == nil {
		http.Error(w, "Transfer order not found", http.StatusNotFound)
		return
	}
	if order.Completed {
		http.Error(w, "Cannot update completed transfer order", http.StatusBadRequest)
		return
	}

	// Подготавливаем детали
	details := make([]database.TransferOrderDetail, 0, len(req.Details))
	for _, d := range req.Details {
		details = append(details, database.TransferOrderDetail{
			MaterialID: d.MaterialID,
			Quantity:   d.Quantity,
		})
	}

	// Обновляем заказ
	err = database.UpdateTransferOrder(orderID, req.Number, req.FromWarehouseID, req.ToWarehouseID, plannedDate, details)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d (update): %v", orderID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info("API: Обновлён заказ на перемещение ID=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order updated",
	})
}

// handleCompleteTransferOrder завершает заказ
func handleCompleteTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.CompleteTransferOrder(orderID)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d/complete: %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_order_completed", map[string]interface{}{
			"transferOrderId": orderID,
		})
	}

	logger.Info("API: Завершён заказ ID=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order completed",
	})
}

// handleDeleteTransferOrder удаляет заказ
func handleDeleteTransferOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.DeleteTransferOrder(orderID)
	if err != nil {
		logger.Error("API /api/transfer-orders/%d (delete): %v", orderID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("API: Удалён заказ ID=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer order deleted",
	})
}
