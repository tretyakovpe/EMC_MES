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

// BoxResponse структура ответа для коробки
type BoxResponse struct {
	HUID          int                     `json:"huId"`
	MaterialID    int                     `json:"materialId"`
	MaterialCode  string                  `json:"materialCode"`
	HUNumber      *string                 `json:"huNumber,omitempty"`
	Amount        int                     `json:"amount"`
	ShipmentID    *int                    `json:"shipmentId,omitempty"`
	CurrentStatus string                  `json:"currentStatus"`
	StatusHistory []StatusHistoryResponse `json:"statusHistory,omitempty"`
}

// StatusHistoryResponse структура истории статусов
type StatusHistoryResponse struct {
	Status    string `json:"status"`
	ChangedAt string `json:"changedAt"`
}

// BoxesGroupResponse структура для сгруппированных коробок (склад ГП)
type BoxesGroupResponse struct {
	MaterialCode string `json:"materialCode"`
	Description  string `json:"description"`
	BoxCount     int    `json:"boxCount"`
	TotalAmount  int    `json:"totalAmount"`
}

// handleBoxes обрабатывает запросы к /api/boxes
func handleBoxes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetBoxes(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetBoxes возвращает список коробок с фильтрацией
func handleGetBoxes(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	materialIDStr := r.URL.Query().Get("materialId")
	grouped := r.URL.Query().Get("grouped") == "true"

	// Если нужна группировка по материалам (для склада ГП)
	if grouped {
		handleGetBoxesGrouped(w, r)
		return
	}

	// Фильтр по статусу
	if status != "" {
		limit := 0
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			l, err := strconv.Atoi(limitStr)
			if err == nil && l > 0 {
				limit = l
			}
		}

		boxes, err := database.GetBoxesByStatus(status, limit)
		if err != nil {
			logger.Error("API /api/boxes (by status): %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := make([]BoxResponse, 0, len(boxes))
		for _, b := range boxes {
			resp := boxToResponse(b)
			response = append(response, resp)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Фильтр по материалу
	if materialIDStr != "" {
		materialID, err := strconv.Atoi(materialIDStr)
		if err != nil {
			http.Error(w, "Invalid materialId", http.StatusBadRequest)
			return
		}

		boxes, err := database.GetBoxesByMaterial(materialID, "")
		if err != nil {
			logger.Error("API /api/boxes (by material): %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := make([]BoxResponse, 0, len(boxes))
		for _, b := range boxes {
			resp := boxToResponse(b)
			response = append(response, resp)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Если нет фильтров - возвращаем последние 100 коробок
	boxes, err := database.GetBoxesByStatus("", 100)
	if err != nil {
		logger.Error("API /api/boxes (all): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]BoxResponse, 0, len(boxes))
	for _, b := range boxes {
		resp := boxToResponse(b)
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetBoxesGrouped возвращает коробки сгруппированные по материалам
func handleGetBoxesGrouped(w http.ResponseWriter, r *http.Request) {
	groups, err := database.GetBoxesGroupedByMaterial()
	if err != nil {
		logger.Error("API /api/boxes?grouped=true: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]BoxesGroupResponse, 0, len(groups))
	for _, g := range groups {
		resp := BoxesGroupResponse{
			MaterialCode: g["materialCode"].(string),
			Description:  g["description"].(string),
			BoxCount:     g["boxCount"].(int),
			TotalAmount:  g["totalAmount"].(int),
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBoxByID обрабатывает запросы к /api/boxes/{id}
func handleBoxByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/boxes/")
	idStr := strings.Split(path, "/")[0]

	// Пробуем как число (HUID)
	if huID, err := strconv.Atoi(idStr); err == nil {
		handleGetBoxByHUID(w, r, huID)
		return
	}

	// Если не число - пробуем как номер бирки (HUNumber)
	handleGetBoxByHUNumber(w, r, idStr)
}

// handleGetBoxByHUID возвращает коробку по HUID
func handleGetBoxByHUID(w http.ResponseWriter, r *http.Request, huID int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем историю статусов
	history, err := database.GetBoxStatusHistory(huID)
	if err != nil {
		logger.Error("API /api/boxes/%d (history): %v", huID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(history) == 0 {
		http.Error(w, "Box not found", http.StatusNotFound)
		return
	}

	// Получаем текущий статус и информацию о коробке
	currentStatus := history[len(history)-1].Status

	// Получаем коробку с деталями через GetBoxesByStatus (упрощённо)
	var boxInfo *database.BoxWithStatus
	boxes, err := database.GetBoxesByStatus("", 0)
	if err == nil {
		for _, b := range boxes {
			if b.HUID == huID {
				boxInfo = &b
				break
			}
		}
	}

	response := BoxResponse{
		HUID:          huID,
		CurrentStatus: currentStatus,
		StatusHistory: make([]StatusHistoryResponse, 0, len(history)),
	}

	if boxInfo != nil {
		response.MaterialID = boxInfo.MaterialID
		response.MaterialCode = boxInfo.MaterialCode
		response.Amount = boxInfo.Amount
		if boxInfo.HUNumber != nil {
			response.HUNumber = boxInfo.HUNumber
		}
		if boxInfo.ShipmentID != nil {
			response.ShipmentID = boxInfo.ShipmentID
		}
	}

	for _, hs := range history {
		response.StatusHistory = append(response.StatusHistory, StatusHistoryResponse{
			Status:    hs.Status,
			ChangedAt: hs.ChangedAt.Format("2006-01-02 15:04:05"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetBoxByHUNumber возвращает коробку по номеру бирки
func handleGetBoxByHUNumber(w http.ResponseWriter, r *http.Request, huNumber string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	box, err := database.GetBoxByHUNumber(huNumber)
	if err != nil {
		logger.Error("API /api/boxes/%s: %v", huNumber, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if box == nil {
		http.Error(w, "Box not found", http.StatusNotFound)
		return
	}

	// Получаем историю статусов
	history, err := database.GetBoxStatusHistory(box.HUID)
	if err != nil {
		logger.Error("API /api/boxes/%s (history): %v", huNumber, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := boxToResponse(*box)
	response.StatusHistory = make([]StatusHistoryResponse, 0, len(history))

	for _, hs := range history {
		response.StatusHistory = append(response.StatusHistory, StatusHistoryResponse{
			Status:    hs.Status,
			ChangedAt: hs.ChangedAt.Format("2006-01-02 15:04:05"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// boxToResponse преобразует BoxWithStatus в BoxResponse
func boxToResponse(b database.BoxWithStatus) BoxResponse {
	resp := BoxResponse{
		HUID:          b.HUID,
		MaterialID:    b.MaterialID,
		MaterialCode:  b.MaterialCode,
		Amount:        b.Amount,
		CurrentStatus: b.CurrentStatus,
	}
	if b.HUNumber != nil {
		resp.HUNumber = b.HUNumber
	}
	if b.ShipmentID != nil {
		resp.ShipmentID = b.ShipmentID
	}
	return resp
}

// GetBoxesStats возвращает статистику по коробкам за период
func GetBoxesStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим параметры дат
	fromDateStr := r.URL.Query().Get("fromDate")
	toDateStr := r.URL.Query().Get("toDate")

	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("2006-01-02", fromDateStr)
		if err != nil {
			http.Error(w, "Invalid fromDate format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	} else {
		fromDate = time.Now().AddDate(0, 0, -30) // последние 30 дней
	}

	if toDateStr != "" {
		toDate, err = time.Parse("2006-01-02", toDateStr)
		if err != nil {
			http.Error(w, "Invalid toDate format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		// Добавляем время до конца дня
		toDate = toDate.Add(24*time.Hour - time.Second)
	} else {
		toDate = time.Now()
	}

	// Получаем количество коробок
	boxesCount, err := database.GetProducedBoxesCount(fromDate, toDate)
	if err != nil {
		logger.Error("API /api/boxes/stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Получаем количество по материалам
	materialStats, err := database.GetProducedAmountByMaterial(fromDate, toDate)
	if err != nil {
		logger.Error("API /api/boxes/stats (by material): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"fromDate":   fromDate.Format("2006-01-02"),
		"toDate":     toDate.Format("2006-01-02"),
		"totalBoxes": boxesCount,
		"byMaterial": materialStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
