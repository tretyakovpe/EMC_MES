package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// LineResponse структура ответа для линии
type LineResponse struct {
	Name       string  `json:"name"`
	IP         string  `json:"ip"`
	Port       *int    `json:"port,omitempty"`
	Printer    *string `json:"printer,omitempty"`
	PrintLabel bool    `json:"printLabel"`
	IsOnline   bool    `json:"isOnline"`
	IsActive   bool    `json:"isActive"`
	LastSeen   *string `json:"lastSeen,omitempty"`
	Camera     *string `json:"camera,omitempty"`
}

// LineStatusRequest структура запроса для изменения статуса
type LineStatusRequest struct {
	IsOnline bool `json:"isOnline"`
}

// handleLines обрабатывает запросы к /api/lines
func handleLines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetLines(w, r)
	case http.MethodPost:
		handleCreateLine(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetLines возвращает список всех линий
func handleGetLines(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр active (если true, возвращаем только активные)
	activeOnly := r.URL.Query().Get("active") == "true"

	var lines []database.LineConfig
	var err error

	if activeOnly {
		lines, err = database.GetActiveLines()
	} else {
		lines, err = database.GetAllLines()
	}

	if err != nil {
		logger.Error("API /api/lines: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]LineResponse, 0, len(lines))
	for _, l := range lines {
		resp := LineResponse{
			Name:       strings.TrimSpace(l.Name),
			IP:         strings.TrimSpace(l.IP),
			PrintLabel: l.PrintLabel,
			IsOnline:   l.IsOnline,
			IsActive:   l.IsActive,
		}
		if l.Port.Valid {
			port := int(l.Port.Int64)
			resp.Port = &port
		}
		if l.Printer.Valid {
			printer := strings.TrimSpace(l.Printer.String)
			resp.Printer = &printer
		}
		if l.Camera.Valid {
			camera := strings.TrimSpace(l.Camera.String)
			resp.Camera = &camera
		}
		if !l.LastCheck.IsZero() {
			lastSeen := l.LastCheck.Format("2006-01-02 15:04:05")
			resp.LastSeen = &lastSeen
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateLine создаёт новую линию
func handleCreateLine(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string  `json:"name"`
		IP         string  `json:"ip"`
		Port       *int    `json:"port"`
		Printer    *string `json:"printer"`
		PrintLabel bool    `json:"printLabel"`
		Camera     *string `json:"camera"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.IP == "" {
		http.Error(w, "IP is required", http.StatusBadRequest)
		return
	}

	port := 102
	if req.Port != nil {
		port = *req.Port
	}

	printer := ""
	if req.Printer != nil {
		printer = *req.Printer
	}

	camera := ""
	if req.Camera != nil {
		camera = *req.Camera
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		createdBy = "api"
	}

	err := database.CreateLine(req.Name, req.IP, port, printer, req.PrintLabel, camera, createdBy)
	if err != nil {
		logger.Error("API /api/lines (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.BroadcastLineStatus(req.Name, false)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Line created",
		"line":    req.Name,
	})
}

// handleLineByID обрабатывает запросы к /api/lines/{name}
func handleLineByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем имя линии из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/lines/")
	// Убираем возможные подпути (например /status)
	lineName := strings.Split(path, "/")[0]

	if lineName == "" {
		http.Error(w, "Line name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetLineByName(w, r, lineName)
	case http.MethodPut:
		handleUpdateLine(w, r, lineName)
	case http.MethodDelete:
		handleDeleteLine(w, r, lineName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetLineByName возвращает информацию о конкретной линии
func handleGetLineByName(w http.ResponseWriter, r *http.Request, lineName string) {
	line, err := database.GetLineByName(lineName)
	if err != nil {
		logger.Error("API /api/lines/%s: %v", lineName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if line == nil {
		http.Error(w, "Line not found", http.StatusNotFound)
		return
	}

	resp := LineResponse{
		Name:       strings.TrimSpace(line.Name),
		IP:         strings.TrimSpace(line.IP),
		PrintLabel: line.PrintLabel,
		IsOnline:   line.IsOnline,
		IsActive:   line.IsActive,
	}
	if line.Port.Valid {
		port := int(line.Port.Int64)
		resp.Port = &port
	}
	if line.Printer.Valid {
		printer := strings.TrimSpace(line.Printer.String)
		resp.Printer = &printer
	}
	if line.Camera.Valid {
		camera := strings.TrimSpace(line.Camera.String)
		resp.Camera = &camera
	}
	if !line.LastCheck.IsZero() {
		lastSeen := line.LastCheck.Format("2006-01-02 15:04:05")
		resp.LastSeen = &lastSeen
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateLine обновляет конфигурацию линии
func handleUpdateLine(w http.ResponseWriter, r *http.Request, lineName string) {
	var req struct {
		IP         string  `json:"ip"`
		Port       *int    `json:"port"`
		Printer    *string `json:"printer"`
		PrintLabel bool    `json:"printLabel"`
		Camera     *string `json:"camera"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.IP == "" {
		http.Error(w, "IP is required", http.StatusBadRequest)
		return
	}

	port := 102
	if req.Port != nil {
		port = *req.Port
	}

	printer := ""
	if req.Printer != nil {
		printer = *req.Printer
	}

	camera := ""
	if req.Camera != nil {
		camera = *req.Camera
	}

	updatedBy := r.Header.Get("X-User")
	if updatedBy == "" {
		updatedBy = "api"
	}

	err := database.UpdateLine(lineName, req.IP, port, printer, req.PrintLabel, camera, updatedBy)
	if err != nil {
		logger.Error("API /api/lines/%s (update): %v", lineName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Line updated",
		"line":    lineName,
	})
}

// handleDeleteLine удаляет линию (деактивирует)
func handleDeleteLine(w http.ResponseWriter, r *http.Request, lineName string) {
	updatedBy := r.Header.Get("X-User")
	if updatedBy == "" {
		updatedBy = "api"
	}

	err := database.DeleteLine(lineName, updatedBy)
	if err != nil {
		logger.Error("API /api/lines/%s (delete): %v", lineName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.BroadcastLineStatus(lineName, false)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Line deactivated",
		"line":    lineName,
	})
}

// handleLineStatus обрабатывает изменение статуса линии (вкл/выкл)
func handleLineStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем имя линии из query параметра
	lineName := r.URL.Query().Get("name")
	if lineName == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	var req LineStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обновляем статус в БД
	database.UpdateLineActiveStatus(lineName, req.IsOnline)

	// Отправляем событие через WebSocket
	if globalHub != nil {
		globalHub.BroadcastLineStatus(lineName, req.IsOnline)
	}

	logger.Info("API: Статус линии %s изменён на: %v", lineName, req.IsOnline)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"message":  "Line status updated",
		"line":     lineName,
		"isOnline": req.IsOnline,
	})
}
