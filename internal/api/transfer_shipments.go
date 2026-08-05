package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// TransferShipmentRequest структура запроса на создание отгрузки
type TransferShipmentRequest struct {
	TransferID int `json:"transferId"`
	MaterialID int `json:"materialId"`
	Quantity   int `json:"quantity"`
}

// TransferShipmentResponse структура ответа для отгрузки
type TransferShipmentResponse struct {
	ShipmentID   int    `json:"shipmentId"`
	TransferID   int    `json:"transferId"`
	MaterialID   int    `json:"materialId"`
	MaterialCode string `json:"materialCode"`
	MaterialDesc string `json:"materialDesc"`
	Quantity     int    `json:"quantity"`
	CreatedAt    string `json:"createdAt"`
	CreatedBy    string `json:"createdBy"`
}

// handleTransferShipments обрабатывает запросы к /api/transfer-shipments
func handleTransferShipments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreateTransferShipment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateTransferShipment создаёт новую отгрузку по заказу
func handleCreateTransferShipment(w http.ResponseWriter, r *http.Request) {
	var req TransferShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("API /api/transfer-shipments: ошибка парсинга: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TransferID == 0 {
		http.Error(w, "transferId is required", http.StatusBadRequest)
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

	// Создаём отгрузку
	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		createdBy = "api"
	}

	shipmentID, err := database.CreateTransferShipment(req.TransferID, req.MaterialID, req.Quantity, createdBy)
	if err != nil {
		logger.Error("API /api/transfer-shipments (create): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.Broadcast("transfer_shipment_created", map[string]interface{}{
			"transferId": req.TransferID,
			"shipmentId": shipmentID,
			"materialId": req.MaterialID,
			"quantity":   req.Quantity,
		})
	}

	logger.Info("API: Создана отгрузка по заказу %d, материал %d, %d шт.", req.TransferID, req.MaterialID, req.Quantity)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "Transfer shipment created",
		"shipmentId": shipmentID,
	})
}

// handleGetTransferShipments возвращает отгрузки по заказу
func handleGetTransferShipments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем transferId из URL: /api/transfer-shipments/123
	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-shipments/")
	idStr := strings.Split(path, "/")[0]

	transferID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transfer ID", http.StatusBadRequest)
		return
	}

	shipments, err := database.GetTransferShipmentsByTransferID(transferID)
	if err != nil {
		logger.Error("API /api/transfer-shipments/%d: %v", transferID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]TransferShipmentResponse, 0, len(shipments))
	for _, s := range shipments {
		response = append(response, TransferShipmentResponse{
			ShipmentID:   s.ShipmentID,
			TransferID:   s.TransferID,
			MaterialID:   s.MaterialID,
			MaterialCode: s.MaterialCode,
			MaterialDesc: s.MaterialDesc,
			Quantity:     s.Quantity,
			CreatedAt:    s.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:    s.CreatedBy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTransferShipmentsGrouped возвращает отгрузки сгруппированные по материалам
func handleGetTransferShipmentsGrouped(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-shipments/")
	// Ожидаем /api/transfer-shipments/123/grouped
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "grouped" {
		http.Error(w, "Invalid URL format. Expected /api/transfer-shipments/{id}/grouped", http.StatusBadRequest)
		return
	}

	transferID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid transfer ID", http.StatusBadRequest)
		return
	}

	grouped, err := database.GetTransferShipmentsByTransferIDGrouped(transferID)
	if err != nil {
		logger.Error("API /api/transfer-shipments/%d/grouped: %v", transferID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(grouped)
}

// handleDeleteTransferShipment удаляет отгрузку
func handleDeleteTransferShipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-shipments/")
	shipmentID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	err = database.DeleteTransferShipment(shipmentID)
	if err != nil {
		logger.Error("API /api/transfer-shipments/%d (delete): %v", shipmentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("API: Удалена отгрузка ID=%d", shipmentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Transfer shipment deleted",
	})
}
