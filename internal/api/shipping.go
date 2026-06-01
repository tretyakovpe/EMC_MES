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

// ShipmentResponse структура ответа для отгрузки
type ShipmentResponse struct {
	ShipmentID int                      `json:"shipmentId"`
	Number     *int                     `json:"number,omitempty"`
	Date       string                   `json:"date"`
	Completed  bool                     `json:"completed"`
	Done       bool                     `json:"done"`
	Progress   int                      `json:"progress"` // процент выполнения (0-100)
	Details    []ShipmentDetailResponse `json:"details,omitempty"`
}

// ShipmentDetailResponse структура детали отгрузки
type ShipmentDetailResponse struct {
	MaterialID     int    `json:"materialId"`
	MaterialCode   string `json:"materialCode"`
	Boxes          int    `json:"boxes"`
	Amount         int    `json:"amount"`
	ScannedBoxes   int    `json:"scannedBoxes"`
	ScannedPercent int    `json:"scannedPercent"`
}

// CreateShipmentRequest структура запроса на создание отгрузки
type CreateShipmentRequest struct {
	Number  *int                          `json:"number"`
	Date    string                        `json:"date"`
	Details []CreateShipmentDetailRequest `json:"details"`
}

// CreateShipmentDetailRequest структура детали при создании
type CreateShipmentDetailRequest struct {
	MaterialID int `json:"materialId"`
	Boxes      int `json:"boxes"`
	Amount     int `json:"amount"`
}

// ScanBoxRequest структура запроса на сканирование коробки
type ScanBoxRequest struct {
	HUID       int `json:"huId"`
	MaterialID int `json:"materialId"`
}

// handleShipments обрабатывает запросы к /api/shipments
func handleShipments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetShipments(w, r)
	case http.MethodPost:
		handleCreateShipment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetShipments возвращает список отгрузок
func handleGetShipments(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	completedStr := r.URL.Query().Get("completed")
	doneStr := r.URL.Query().Get("done")
	fromDateStr := r.URL.Query().Get("fromDate")
	toDateStr := r.URL.Query().Get("toDate")

	var completed, done *bool
	if completedStr != "" {
		b := completedStr == "true"
		completed = &b
	}
	if doneStr != "" {
		b := doneStr == "true"
		done = &b
	}

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

	shipments, err := database.GetShipments(completed, done, fromDate, toDate)
	if err != nil {
		logger.Error("API /api/shipments: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ShipmentResponse, 0, len(shipments))
	for _, s := range shipments {
		progress, _ := database.GetShipmentProgress(s.ShipmentID)
		resp := ShipmentResponse{
			ShipmentID: s.ShipmentID,
			Number:     s.Number,
			Date:       s.Date.Format("2006-01-02"),
			Completed:  s.Completed,
			Done:       s.Done,
			Progress:   progress,
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateShipment создаёт новую отгрузку
func handleCreateShipment(w http.ResponseWriter, r *http.Request) {
	var req CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Парсим дату
	date := time.Now()
	if req.Date != "" {
		t, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		date = t
	}

	// Создаём отгрузку
	shipmentID, err := database.CreateShipment(req.Number, date)
	if err != nil {
		logger.Error("API /api/shipments (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Добавляем детали
	for _, detail := range req.Details {
		if err := database.AddShipmentDetail(shipmentID, detail.MaterialID, detail.Boxes, detail.Amount); err != nil {
			logger.Error("API /api/shipments (add detail): %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.BroadcastShipmentCompleted(shipmentID, req.Number)
	}

	logger.Info("API: Создана отгрузка ID=%d", shipmentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "Shipment created",
		"shipmentId": shipmentID,
	})
}

// handleShipmentByID обрабатывает запросы к /api/shipments/{id}
func handleShipmentByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/shipments/")
	idStr := strings.Split(path, "/")[0]

	shipmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Проверяем, есть ли подпуть (например /scan, /complete)
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		switch parts[1] {
		case "scan":
			handleScanBox(w, r, shipmentID)
			return
		case "complete":
			handleCompleteShipment(w, r, shipmentID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		handleGetShipmentByID(w, r, shipmentID)
	case http.MethodDelete:
		handleDeleteShipment(w, r, shipmentID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetShipmentByID возвращает отгрузку с деталями
func handleGetShipmentByID(w http.ResponseWriter, r *http.Request, shipmentID int) {
	shipment, err := database.GetShipmentByID(shipmentID)
	if err != nil {
		logger.Error("API /api/shipments/%d: %v", shipmentID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if shipment == nil {
		http.Error(w, "Shipment not found", http.StatusNotFound)
		return
	}

	progress, _ := database.GetShipmentProgress(shipmentID)

	response := ShipmentResponse{
		ShipmentID: shipment.ShipmentID,
		Number:     shipment.Number,
		Date:       shipment.Date.Format("2006-01-02"),
		Completed:  shipment.Completed,
		Done:       shipment.Done,
		Progress:   progress,
		Details:    make([]ShipmentDetailResponse, 0, len(shipment.Details)),
	}

	for _, d := range shipment.Details {
		scannedPercent := 0
		if d.Boxes > 0 {
			scannedPercent = (d.ScannedBoxes * 100) / d.Boxes
		}
		response.Details = append(response.Details, ShipmentDetailResponse{
			MaterialID:     d.MaterialID,
			MaterialCode:   d.MaterialCode,
			Boxes:          d.Boxes,
			Amount:         d.Amount,
			ScannedBoxes:   d.ScannedBoxes,
			ScannedPercent: scannedPercent,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleScanBox обрабатывает сканирование коробки в отгрузку
func handleScanBox(w http.ResponseWriter, r *http.Request, shipmentID int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanBoxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.HUID == 0 {
		http.Error(w, "huId is required", http.StatusBadRequest)
		return
	}
	if req.MaterialID == 0 {
		http.Error(w, "materialId is required", http.StatusBadRequest)
		return
	}

	// Выполняем сканирование
	isCompleted, err := database.ScanBoxForShipment(shipmentID, req.HUID, req.MaterialID)
	if err != nil {
		logger.Error("API /api/shipments/%d/scan: %v", shipmentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		shipment, _ := database.GetShipmentByID(shipmentID)
		if shipment != nil {
			globalHub.BroadcastShipmentCompleted(shipmentID, shipment.Number)
		}
	}

	logger.Info("API: Отсканирована коробка HU=%d в отгрузку %d, завершена: %v", req.HUID, shipmentID, isCompleted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"message":   "Box scanned successfully",
		"completed": isCompleted,
	})
}

// handleCompleteShipment завершает отгрузку (Done = true)
func handleCompleteShipment(w http.ResponseWriter, r *http.Request, shipmentID int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.CompleteShipment(shipmentID)
	if err != nil {
		logger.Error("API /api/shipments/%d/complete: %v", shipmentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем номер отгрузки для события
	shipment, _ := database.GetShipmentByID(shipmentID)
	var shipmentNumber *int
	if shipment != nil {
		shipmentNumber = shipment.Number
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.BroadcastShipmentCompleted(shipmentID, shipmentNumber)
	}

	logger.Info("API: Завершена отгрузка ID=%d", shipmentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Shipment completed",
	})
}

// handleDeleteShipment удаляет отгрузку
func handleDeleteShipment(w http.ResponseWriter, r *http.Request, shipmentID int) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.DeleteShipment(shipmentID)
	if err != nil {
		logger.Error("API /api/shipments/%d (delete): %v", shipmentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("API: Удалена отгрузка ID=%d", shipmentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Shipment deleted",
	})
}
