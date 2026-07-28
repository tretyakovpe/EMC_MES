package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// WarehouseRequest структура запроса для создания/обновления склада
type WarehouseRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// WarehouseResponse структура ответа для склада
type WarehouseResponse struct {
	WarehouseID int     `json:"warehouseId"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// handleWarehouses обрабатывает запросы к /api/warehouses
func handleWarehouses(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetWarehouses(w, r)
	case http.MethodPost:
		handleCreateWarehouse(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetWarehouses возвращает список складов
func handleGetWarehouses(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	warehouses, err := database.GetWarehouses(activeOnly)
	if err != nil {
		logger.Error("API /api/warehouses: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]WarehouseResponse, 0, len(warehouses))
	for _, w := range warehouses {
		resp := WarehouseResponse{
			WarehouseID: w.WarehouseID,
			Code:        w.Code,
			Name:        w.Name,
			IsActive:    w.IsActive,
		}
		if w.Description != nil {
			resp.Description = w.Description
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateWarehouse создаёт новый склад
func handleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req WarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	warehouseID, err := database.CreateWarehouse(req.Code, req.Name, req.Description)
	if err != nil {
		logger.Error("API /api/warehouses (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"message":     "Warehouse created",
		"warehouseId": warehouseID,
	})
}

// handleWarehouseByID обрабатывает запросы к /api/warehouses/{id}
func handleWarehouseByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/warehouses/")
	idStr := strings.Split(path, "/")[0]

	warehouseID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid warehouse ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetWarehouseByID(w, r, warehouseID)
	case http.MethodPut:
		handleUpdateWarehouse(w, r, warehouseID)
	case http.MethodDelete:
		handleDeleteWarehouse(w, r, warehouseID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetWarehouseByID возвращает склад по ID
func handleGetWarehouseByID(w http.ResponseWriter, r *http.Request, warehouseID int) {
	warehouse, err := database.GetWarehouseByID(warehouseID)
	if err != nil {
		logger.Error("API /api/warehouses/%d: %v", warehouseID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if warehouse == nil {
		http.Error(w, "Warehouse not found", http.StatusNotFound)
		return
	}

	resp := WarehouseResponse{
		WarehouseID: warehouse.WarehouseID,
		Code:        warehouse.Code,
		Name:        warehouse.Name,
		IsActive:    warehouse.IsActive,
	}
	if warehouse.Description != nil {
		resp.Description = warehouse.Description
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateWarehouse обновляет склад
func handleUpdateWarehouse(w http.ResponseWriter, r *http.Request, warehouseID int) {
	var req WarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	err := database.UpdateWarehouse(warehouseID, req.Code, req.Name, req.Description, req.IsActive)
	if err != nil {
		logger.Error("API /api/warehouses/%d (update): %v", warehouseID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Warehouse updated",
	})
}

// handleDeleteWarehouse удаляет склад
func handleDeleteWarehouse(w http.ResponseWriter, r *http.Request, warehouseID int) {
	err := database.DeleteWarehouse(warehouseID)
	if err != nil {
		logger.Error("API /api/warehouses/%d (delete): %v", warehouseID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Warehouse deleted",
	})
}
