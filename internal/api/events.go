package api

import (
	"encoding/json"
	"net/http"

	"EMC_MES/internal/logger"
)

// Event структура события от Data Collector
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Ts   int64           `json:"ts"`
}

// BoxClosedEvent событие закрытия коробки
type BoxClosedEvent struct {
	Line      string `json:"line"`
	Material  string `json:"material"`
	LabelCode string `json:"labelCode"`
	Amount    int    `json:"amount"`
}

// PartEvent событие производства детали
type PartEvent struct {
	Line      string `json:"line"`
	Material  string `json:"material"`
	Counter   int    `json:"counter"`
	BoxVolume int    `json:"boxVolume"`
}

// LineStatusEvent событие изменения статуса линии
type LineStatusEvent struct {
	Line     string `json:"line"`
	IsOnline bool   `json:"isOnline"`
}

// handleEvent обрабатывает входящие события от Data Collector
func handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		logger.Error("API /api/events: ошибка парсинга JSON: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.Debug("API /api/events: получено событие типа %s", event.Type)

	// Обрабатываем событие в зависимости от типа
	switch event.Type {
	case "box_closed":
		var data BoxClosedEvent
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга box_closed: %v", err)
			http.Error(w, "Invalid box_closed event data", http.StatusBadRequest)
			return
		}
		handleBoxClosedEvent(data)

	case "part_ok":
		var data PartEvent
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга part_ok: %v", err)
			http.Error(w, "Invalid part_ok event data", http.StatusBadRequest)
			return
		}
		handlePartEvent(data, true)

	case "part_nok":
		var data PartEvent
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга part_nok: %v", err)
			http.Error(w, "Invalid part_nok event data", http.StatusBadRequest)
			return
		}
		handlePartEvent(data, false)

	case "line_card_update":
		var data PartEvent
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга при обновлении карточек: %v", err)
			http.Error(w, "Invalid line card update event data", http.StatusBadRequest)
			return
		}
		hadleLineCardUpdate(data)

	case "line_status":
		var data LineStatusEvent
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга line_status: %v", err)
			http.Error(w, "Invalid line_status event data", http.StatusBadRequest)
			return
		}
		handleLineStatusEvent(data)
	case "plan_updated":
		var data map[string]interface{}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error("API /api/events: ошибка парсинга plan_updated: %v", err)
			return
		}
		handlePlanUpdated(data)

	default:
		logger.Warn("API /api/events: неизвестный тип события %s", event.Type)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

func hadleLineCardUpdate(data PartEvent) {
	// Отправляем событие всем подключённым WebSocket клиентам
	if globalHub != nil {
		globalHub.BroadcastLineCardUpdate(data.Line, data.Material, data.Counter, data.BoxVolume)
	}
}

// handleBoxClosedEvent обрабатывает событие закрытия коробки
func handleBoxClosedEvent(data BoxClosedEvent) {
	logger.Info("Событие: закрыта коробка на линии %s, материал %s, бирка %s, кол-во %d",
		data.Line, data.Material, data.LabelCode, data.Amount)

	// Отправляем событие всем подключённым WebSocket клиентам
	if globalHub != nil {
		globalHub.BroadcastBoxClosed(data.Line, data.Material, data.LabelCode, data.Amount)
	}
}

// handlePartEvent обрабатывает событие производства детали
func handlePartEvent(data PartEvent, isGood bool) {
	status := "OK"
	if !isGood {
		status = "NOK"
	}
	logger.Info("Событие: деталь %s на линии %s, счётчик %d/%d, статус %s",
		data.Material, data.Line, data.Counter, data.BoxVolume, status)

	// Отправляем событие всем подключённым WebSocket клиентам
	if globalHub != nil {
		globalHub.BroadcastPartProduced(data.Line, data.Material, data.Counter, data.BoxVolume, isGood)
	}
}

// handlePlanUpdated отправляет сообщение клиентам для обновления сменного задания
func handlePlanUpdated(data map[string]interface{}) {
	logger.Info("Событие: обновлён план ID=%v", data["planId"])

	if globalHub != nil {
		globalHub.Broadcast("plan_updated", data)
	}
}

// handleLineStatusEvent обрабатывает событие изменения статуса линии
func handleLineStatusEvent(data LineStatusEvent) {
	status := "ONLINE"
	if !data.IsOnline {
		status = "OFFLINE"
	}
	logger.Info("Событие: линия %s изменила статус на %s", data.Line, status)

	// Отправляем событие всем подключённым WebSocket клиентам
	if globalHub != nil {
		globalHub.BroadcastLineStatus(data.Line, data.IsOnline)
	}
}

// handleLineStatusEvent обрабатывает событие включения/отключения линии
func handleLineActiveEvent(data LineStatusEvent) {
	status := "ACTIVE"
	if !data.IsOnline {
		status = "DISABLED"
	}
	logger.Info("Событие: линия %s изменила статус на %s", data.Line, status)

	// Отправляем событие всем подключённым WebSocket клиентам
	if globalHub != nil {
		globalHub.BroadcastLineActive(data.Line, data.IsOnline)
	}
}
