package api

import (
	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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

type ParseClipboardResponse struct {
	Rows     []ParsedRow `json:"rows"`
	Errors   []string    `json:"errors,omitempty"`
	Warnings []string    `json:"warnings,omitempty"` // НОВОЕ
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

// handleGetScannedBoxes возвращает список отсканированных коробок, сгруппированных по материалам
func handleGetScannedBoxes(w http.ResponseWriter, r *http.Request, shipmentID int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scannedBoxes, err := database.GetScannedBoxesByShipment(shipmentID)
	if err != nil {
		logger.Error("API /api/shipments/%d/scanned: %v", shipmentID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Добавляем пустые массивы для материалов, которые есть в отгрузке, но ещё не отсканированы
	shipment, err := database.GetShipmentByID(shipmentID)
	if err == nil && shipment != nil {
		for _, detail := range shipment.Details {
			materialCode := strings.TrimSpace(detail.MaterialCode)
			if _, exists := scannedBoxes[materialCode]; !exists {
				scannedBoxes[materialCode] = []string{}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scannedBoxes)
}

// ParseClipboardRequest структура запроса на парсинг буфера
type ParseClipboardRequest struct {
	Text string `json:"text"`
}

// ParsedRow результат парсинга одной строки
type ParsedRow struct {
	CustomerCode string `json:"customerCode"`
	MaterialCode string `json:"materialCode"`
	MaterialID   int    `json:"materialId"`
	Boxes        int    `json:"boxes"`
	Amount       int    `json:"amount"`
	Valid        bool   `json:"valid"`
	Error        string `json:"error,omitempty"`
}

// handleParseClipboard парсит текст из буфера обмена
func handleParseClipboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseClipboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	response := parseClipboardText(req.Text)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseClipboardText основная логика парсинга
func parseClipboardText(text string) ParseClipboardResponse {
	lines := strings.Split(text, "\n")
	response := ParseClipboardResponse{
		Rows:     make([]ParsedRow, 0),
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	for lineIdx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Разбиваем по табуляции
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			response.Errors = append(response.Errors, fmt.Sprintf("Строка %d: не удалось разбить по табуляции", lineIdx+1))
			continue
		}

		// CustomerCode — первое поле (убираем возможный слеш и пробелы)
		customerCodeField := strings.TrimSpace(fields[0])
		// Если есть слеш, берём первую часть
		if idx := strings.Index(customerCodeField, "/"); idx > 0 {
			customerCodeField = customerCodeField[:idx]
		}
		customerCode := extractCustomerCode(customerCodeField)
		if customerCode == "" {
			response.Errors = append(response.Errors, fmt.Sprintf("Строка %d: не найден арт. в поле '%s'", lineIdx+1, customerCodeField))
			continue
		}

		// Количество — последнее поле
		lastField := strings.TrimSpace(fields[len(fields)-1])
		amount := extractAmount(lastField)
		if amount == 0 {
			response.Errors = append(response.Errors, fmt.Sprintf("Строка %d: не найдено количество в поле '%s'", lineIdx+1, lastField))
			continue
		}

		// Ищем материал по CustomerCode в БД
		material, err := database.GetMaterialByCustomerCode(customerCode)
		if err != nil {
			logger.Error("[PARSE] Ошибка поиска материала по арт. %s: %v", customerCode, err)
			response.Rows = append(response.Rows, ParsedRow{
				CustomerCode: customerCode,
				Valid:        false,
				Error:        fmt.Sprintf("Ошибка БД: %v", err),
			})
			continue
		}

		if material == nil {
			response.Rows = append(response.Rows, ParsedRow{
				CustomerCode: customerCode,
				Amount:       amount,
				Valid:        false,
				Error:        "Материал с таким арт. не найден",
			})
			continue
		}

		// Рассчитываем количество коробок (штуки / вместимость)
		boxesCount := amount / material.QuantityInHU
		remainder := amount % material.QuantityInHU

		if remainder > 0 {
			boxesCount++
			response.Warnings = append(response.Warnings, fmt.Sprintf(
				"Строка %d: материал %s: %d шт. не кратно вместимости коробки (%d). Добавлено %d коробок.",
				lineIdx+1, material.MaterialCode, amount, material.QuantityInHU, boxesCount))
		}

		// Всё хорошо — добавляем валидную строку
		response.Rows = append(response.Rows, ParsedRow{
			CustomerCode: customerCode,
			MaterialCode: material.MaterialCode,
			MaterialID:   material.MaterialID,
			Boxes:        boxesCount,
			Amount:       amount,
			Valid:        true,
		})
	}

	return response
}

// extractCustomerCode извлекает CustomerCode из строки
func extractCustomerCode(line string) string {
	// Ищем последовательность из 10-12 цифр
	re := regexp.MustCompile(`(\d{10,12})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractAmount извлекает количество (убирает пробелы как разделители тысяч)
func extractAmount(field string) int {
	// Убираем все пробелы (разделители тысяч)
	cleaned := strings.ReplaceAll(field, " ", "")
	// Убираем запятые
	cleaned = strings.ReplaceAll(cleaned, ",", "")

	// Ищем число
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(cleaned)
	if len(matches) > 1 {
		amount, err := strconv.Atoi(matches[1])
		if err == nil {
			return amount
		}
	}
	return 0
}
