package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"EMC_MES/internal/database"
	"EMC_MES/internal/events"
	"EMC_MES/internal/logger"
)

// ScanRequest структура запроса на сканирование
type ScanRequest struct {
	ShipmentID   int    `json:"shipmentId"`
	MaterialCode string `json:"materialCode"`
	LabelNumber  string `json:"labelNumber"`
}

// ScanResponse структура ответа на сканирование
type ScanResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Completed bool   `json:"completed"`
}

// RegisterScanRoutes регистрирует маршруты для сканирования
func RegisterScanRoutes(mux *http.ServeMux, hub *events.Hub) {
	mux.HandleFunc("/api/scan/box", func(w http.ResponseWriter, r *http.Request) {
		handleScanBox(w, r, hub)
	})
}

// handleScanBox обрабатывает запрос на сканирование коробки
func handleScanBox(w http.ResponseWriter, r *http.Request, hub *events.Hub) {
	if r.Method != http.MethodPost {
		sendScanError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("[SCAN] Ошибка чтения тела: %v", err)
		sendScanError(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Info("[SCAN] Получен запрос: %s", string(body))

	var req ScanRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error("[SCAN] Ошибка парсинга JSON: %v", err)
		sendScanError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.ShipmentID == 0 {
		sendScanError(w, "shipmentId is required", http.StatusBadRequest)
		return
	}
	if req.MaterialCode == "" {
		sendScanError(w, "materialCode is required", http.StatusBadRequest)
		return
	}
	if req.LabelNumber == "" {
		sendScanError(w, "labelNumber is required", http.StatusBadRequest)
		return
	}

	// Выполняем сканирование
	response, err := processScan(req, hub)
	if err != nil {
		logger.Error("[SCAN] Ошибка обработки: %v", err)
		sendScanError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processScan выполняет основную логику сканирования
func processScan(req ScanRequest, hub *events.Hub) (*ScanResponse, error) {
	logger.Info("[SCAN] Обработка: отгрузка=%d, материал=%s, бирка=%s",
		req.ShipmentID, req.MaterialCode, req.LabelNumber)

	// 1. Находим MaterialID по коду
	material, err := database.GetMaterialByCode(req.MaterialCode)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска материала: %w", err)
	}
	if material == nil {
		return nil, fmt.Errorf("материал %s не найден", req.MaterialCode)
	}

	// 2. Находим коробку по номеру бирки
	box, err := database.GetBoxByHUNumber(req.LabelNumber)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска коробки: %w", err)
	}
	if box == nil {
		return nil, fmt.Errorf("коробка %s не найдена", req.LabelNumber)
	}

	logger.Info("[SCAN] Найдена коробка: HUID=%d, MaterialID=%s, Status=%s",
		box.HUID, box.MaterialCode, box.CurrentStatus)

	// 3. Проверяем, не отсканирована ли уже коробка в эту отгрузку
	if box.ShipmentID != nil && *box.ShipmentID == req.ShipmentID {
		return &ScanResponse{
			Status:    "already_scanned",
			Message:   "Коробка уже отсканирована в этой отгрузке",
			Completed: false,
		}, nil
	}

	// 4. Проверяем, не в другой ли отгрузке коробка
	if box.ShipmentID != nil && *box.ShipmentID != 0 {
		return nil, fmt.Errorf("коробка уже в отгрузке %d", *box.ShipmentID)
	}

	// 5. Проверяем соответствие материала
	if box.MaterialID != material.MaterialID {
		return nil, fmt.Errorf("материал коробки (%d) не соответствует ожидаемому (%d)",
			box.MaterialID, material.MaterialID)
	}

	// 6. Проверяем статус коробки (должна быть "Произведена")
	if box.CurrentStatus != "Произведена" {
		return nil, fmt.Errorf("коробка имеет статус '%s', требуется 'Произведена'", box.CurrentStatus)
	}

	// 7. Выполняем сканирование в БД
	completed, err := database.ScanBoxForShipment(req.ShipmentID, box.HUID, material.MaterialID)
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования в БД: %w", err)
	}

	// 8. Отправляем WebSocket событие
	if hub != nil {
		shipment, _ := database.GetShipmentByID(req.ShipmentID)
		if shipment != nil {
			hub.BroadcastShipmentCompleted(req.ShipmentID, shipment.Number)
		}
	}

	message := "Коробка добавлена"
	if completed {
		message = "Отгрузка укомплектована!"
	}

	logger.Info("[SCAN] Успех: коробка %s добавлена в отгрузку %d, завершена=%v",
		req.LabelNumber, req.ShipmentID, completed)

	return &ScanResponse{
		Status:    "ok",
		Message:   message,
		Completed: completed,
	}, nil
}

// sendScanError отправляет ошибку в формате JSON
func sendScanError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ScanResponse{
		Status:  "error",
		Message: message,
	})
}
