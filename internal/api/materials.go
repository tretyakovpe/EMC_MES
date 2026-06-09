package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// MaterialResponse структура ответа для материала
type MaterialResponse struct {
	MaterialID   int    `json:"materialId"`
	MaterialCode string `json:"materialCode"`
	CustomerCode string `json:"customerCode"`
	Destination  string `json:"destination"`
	HU           string `json:"hu"`
	Netto        int    `json:"netto"`
	Brutto       int    `json:"brutto"`
	QuantityInHU int    `json:"quantityInHU"`
}

// MaterialRequest структура запроса для создания/обновления материала
type MaterialRequest struct {
	MaterialCode string `json:"materialCode"`
	CustomerCode string `json:"customerCode"`
	Destination  string `json:"destination"`
	HU           string `json:"hu"`
	Netto        int    `json:"netto"`
	Brutto       int    `json:"brutto"`
	QuantityInHU int    `json:"quantityInHU"`
	Description  string `json:"description"`
}

// handleMaterials обрабатывает запросы к /api/materials
func handleMaterials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetMaterials(w, r)
	case http.MethodPost:
		handleCreateMaterial(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetMaterials возвращает список всех материалов
func handleGetMaterials(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр prefix для фильтрации по префиксу
	prefix := r.URL.Query().Get("prefix")

	var materials []database.Material
	var err error

	if prefix != "" {
		materials, err = database.GetMaterialsByCodePrefix(prefix)
	} else {
		materials, err = database.GetAllMaterials()
	}

	if err != nil {
		logger.Error("API /api/materials: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]MaterialResponse, 0, len(materials))
	for _, m := range materials {
		response = append(response, MaterialResponse{
			MaterialID:   m.MaterialID,
			MaterialCode: strings.TrimSpace(m.MaterialCode),
			CustomerCode: strings.TrimSpace(m.CustomerCode),
			Destination:  strings.TrimSpace(m.Destination),
			HU:           strings.TrimSpace(m.HU),
			Netto:        m.Netto,
			Brutto:       m.Brutto,
			QuantityInHU: m.QuantityInHU,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateMaterial создаёт новый материал
func handleCreateMaterial(w http.ResponseWriter, r *http.Request) {
	var req MaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.MaterialCode == "" {
		http.Error(w, "MaterialCode is required", http.StatusBadRequest)
		return
	}
	if req.CustomerCode == "" {
		http.Error(w, "CustomerCode is required", http.StatusBadRequest)
		return
	}
	if req.HU == "" {
		http.Error(w, "HU is required", http.StatusBadRequest)
		return
	}

	materialID, err := database.CreateMaterial(
		req.MaterialCode,
		req.CustomerCode,
		req.Destination,
		req.HU,
		req.Netto,
		req.Brutto,
		req.QuantityInHU,
		req.Description,
	)
	if err != nil {
		logger.Error("API /api/materials (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "Material created",
		"materialId": materialID,
	})
}

// handleMaterialByID обрабатывает запросы к /api/materials/{id}
func handleMaterialByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/materials/")
	idStr := strings.Split(path, "/")[0]

	materialID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid material ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetMaterialByID(w, r, materialID)
	case http.MethodPut:
		handleUpdateMaterial(w, r, materialID)
	case http.MethodDelete:
		handleDeleteMaterial(w, r, materialID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetMaterialByID возвращает материал по ID
func handleGetMaterialByID(w http.ResponseWriter, r *http.Request, materialID int) {
	material, err := database.GetMaterialByID(materialID)
	if err != nil {
		logger.Error("API /api/materials/%d: %v", materialID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if material == nil {
		http.Error(w, "Material not found", http.StatusNotFound)
		return
	}

	response := MaterialResponse{
		MaterialID:   material.MaterialID,
		MaterialCode: strings.TrimSpace(material.MaterialCode),
		CustomerCode: strings.TrimSpace(material.CustomerCode),
		Destination:  strings.TrimSpace(material.Destination),
		HU:           strings.TrimSpace(material.HU),
		Netto:        material.Netto,
		Brutto:       material.Brutto,
		QuantityInHU: material.QuantityInHU,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateMaterial обновляет материал
func handleUpdateMaterial(w http.ResponseWriter, r *http.Request, materialID int) {
	var req MaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.CustomerCode == "" {
		http.Error(w, "CustomerCode is required", http.StatusBadRequest)
		return
	}
	if req.HU == "" {
		http.Error(w, "HU is required", http.StatusBadRequest)
		return
	}

	err := database.UpdateMaterial(
		materialID,
		req.CustomerCode,
		req.Destination,
		req.HU,
		req.Netto,
		req.Brutto,
		req.QuantityInHU,
		req.Description,
	)
	if err != nil {
		logger.Error("API /api/materials/%d (update): %v", materialID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Material updated",
	})
}

// handleDeleteMaterial удаляет материал
func handleDeleteMaterial(w http.ResponseWriter, r *http.Request, materialID int) {
	err := database.DeleteMaterial(materialID)
	if err != nil {
		logger.Error("API /api/materials/%d (delete): %v", materialID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Material deleted",
	})
}

// handleGetMaterialByCode возвращает материал по коду (через query параметр)
func handleGetMaterialByCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	materialCode := r.URL.Query().Get("code")
	if materialCode == "" {
		http.Error(w, "Missing 'code' parameter", http.StatusBadRequest)
		return
	}

	material, err := database.GetMaterialByCode(materialCode)
	if err != nil {
		logger.Error("API /api/materials/code?code=%s: %v", materialCode, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if material == nil {
		http.Error(w, "Material not found", http.StatusNotFound)
		return
	}

	response := MaterialResponse{
		MaterialID:   material.MaterialID,
		MaterialCode: strings.TrimSpace(material.MaterialCode),
		CustomerCode: strings.TrimSpace(material.CustomerCode),
		Destination:  strings.TrimSpace(material.Destination),
		HU:           strings.TrimSpace(material.HU),
		Netto:        material.Netto,
		Brutto:       material.Brutto,
		QuantityInHU: material.QuantityInHU,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
