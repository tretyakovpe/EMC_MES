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

// TransferRequest структура запроса для создания заявки
type TransferRequest struct {
	FromWarehouseID int `json:"fromWarehouseId"`
	ToWarehouseID   int `json:"toWarehouseId"`
	MaterialID      int `json:"materialId"`
	Quantity        int `json:"quantity"`
}

// TransferResponse структура ответа для заявки
type TransferResponse struct {
	TransferID        int     `json:"transferId"`
	TransferNumber    string  `json:"transferNumber"`
	FromWarehouseID   int     `json:"fromWarehouseId"`
	FromWarehouseCode string  `json:"fromWarehouseCode"`
	FromWarehouseName string  `json:"fromWarehouseName"`
	ToWarehouseID     int     `json:"toWarehouseId"`
	ToWarehouseCode   string  `json:"toWarehouseCode"`
	ToWarehouseName   string  `json:"toWarehouseName"`
	MaterialID        int     `json:"materialId"`
	MaterialCode      string  `json:"materialCode"`
	MaterialDesc      string  `json:"materialDesc"`
	Quantity          int     `json:"quantity"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"createdAt"`
	CompletedAt       *string `json:"completedAt,omitempty"`
}

// handleTransfers обрабатывает запросы к /api/transfers
func handleTransfers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetTransfers(w, r)
	case http.MethodPost:
		handleCreateTransfer(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetTransfers возвращает список заявок
func handleGetTransfers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
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

	transfers, err := database.GetTransfers(status, fromDate, toDate)
	if err != nil {
		logger.Error("API /api/transfers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]TransferResponse, 0, len(transfers))
	for _, t := range transfers {
		resp := TransferResponse{
			TransferID:        t.TransferID,
			TransferNumber:    t.TransferNumber,
			FromWarehouseID:   t.FromWarehouseID,
			FromWarehouseCode: t.FromWarehouseCode,
			FromWarehouseName: t.FromWarehouseName,
			ToWarehouseID:     t.ToWarehouseID,
			ToWarehouseCode:   t.ToWarehouseCode,
			ToWarehouseName:   t.ToWarehouseName,
			MaterialID:        t.MaterialID,
			MaterialCode:      t.MaterialCode,
			MaterialDesc:      t.MaterialDesc,
			Quantity:          t.Quantity,
			Status:            t.Status,
			CreatedAt:         t.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if t.CompletedAt != nil {
			completed := t.CompletedAt.Format("2006-01-02 15:04:05")
			resp.CompletedAt = &completed
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateTransfer создаёт новую заявку
func handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FromWarehouseID == 0 {
		http.Error(w, "fromWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.ToWarehouseID == 0 {
		http.Error(w, "toWarehouseId is required", http.StatusBadRequest)
		return
	}
	if req.MaterialID == 0 {
		http.Error(w, "materialId is required", http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		http.Error(w, "fromWarehouseId and toWarehouseId cannot be the same", http.StatusBadRequest)
		return
	}

	// Генерируем номер заявки
	transferNumber, err := database.GenerateTransferNumber()
	if err != nil {
		logger.Error("API /api/transfers (generate number): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	transferID, err := database.CreateTransfer(
		transferNumber,
		req.FromWarehouseID,
		req.ToWarehouseID,
		req.MaterialID,
		req.Quantity,
	)
	if err != nil {
		logger.Error("API /api/transfers (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_created", map[string]interface{}{
			"transferId":     transferID,
			"transferNumber": transferNumber,
			"status":         "Создана",
		})
	}

	logger.Info("API: Создана заявка %s (ID=%d)", transferNumber, transferID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"message":        "Transfer created",
		"transferId":     transferID,
		"transferNumber": transferNumber,
	})
}

// handleTransferByID обрабатывает запросы к /api/transfers/{id}
func handleTransferByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/transfers/")
	idStr := strings.Split(path, "/")[0]

	transferID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transfer ID", http.StatusBadRequest)
		return
	}

	// Проверяем подпути
	if len(strings.Split(path, "/")) > 1 {
		subPath := strings.Split(path, "/")[1]
		switch subPath {
		case "status":
			handleUpdateTransferStatus(w, r, transferID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		handleGetTransferByID(w, r, transferID)
	case http.MethodDelete:
		handleDeleteTransfer(w, r, transferID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetTransferByID возвращает заявку по ID
func handleGetTransferByID(w http.ResponseWriter, r *http.Request, transferID int) {
	transfer, err := database.GetTransferByID(transferID)
	if err != nil {
		logger.Error("API /api/transfers/%d: %v", transferID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if transfer == nil {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}

	resp := TransferResponse{
		TransferID:        transfer.TransferID,
		TransferNumber:    transfer.TransferNumber,
		FromWarehouseID:   transfer.FromWarehouseID,
		FromWarehouseCode: transfer.FromWarehouseCode,
		FromWarehouseName: transfer.FromWarehouseName,
		ToWarehouseID:     transfer.ToWarehouseID,
		ToWarehouseCode:   transfer.ToWarehouseCode,
		ToWarehouseName:   transfer.ToWarehouseName,
		MaterialID:        transfer.MaterialID,
		MaterialCode:      transfer.MaterialCode,
		MaterialDesc:      transfer.MaterialDesc,
		Quantity:          transfer.Quantity,
		Status:            transfer.Status,
		CreatedAt:         transfer.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if transfer.CompletedAt != nil {
		completed := transfer.CompletedAt.Format("2006-01-02 15:04:05")
		resp.CompletedAt = &completed
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateTransferStatus обновляет статус заявки
func handleUpdateTransferStatus(w http.ResponseWriter, r *http.Request, transferID int) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	err := database.UpdateTransferStatus(transferID, req.Status)
	if err != nil {
		logger.Error("API /api/transfers/%d/status: %v", transferID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_status_updated", map[string]interface{}{
			"transferId": transferID,
			"status":     req.Status,
		})
	}

	logger.Info("API: Обновлён статус заявки ID=%d -> %s", transferID, req.Status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer status updated",
	})
}

// handleDeleteTransfer удаляет заявку
func handleDeleteTransfer(w http.ResponseWriter, r *http.Request, transferID int) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.DeleteTransfer(transferID)
	if err != nil {
		logger.Error("API /api/transfers/%d (delete): %v", transferID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("API: Удалена заявка ID=%d", transferID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer deleted",
	})
}
