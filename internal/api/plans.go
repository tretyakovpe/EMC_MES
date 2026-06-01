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

// PlanRequest структура для создания/обновления плана
type PlanRequest struct {
	PlanDate      string  `json:"planDate"`
	Shift         *string `json:"shift"`
	MaterialID    int     `json:"materialId"`
	MaterialCode  string  `json:"materialCode"`
	PlannedAmount int     `json:"plannedAmount"`
}

// PlanResponse структура ответа
type PlanResponse struct {
	PlanID        int     `json:"planId"`
	PlanDate      string  `json:"planDate"`
	Shift         *string `json:"shift"`
	MaterialID    int     `json:"materialId"`
	MaterialCode  string  `json:"materialCode"`
	PlannedAmount int     `json:"plannedAmount"`
	ActualAmount  int     `json:"actualAmount"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	CreatedBy     *string `json:"createdBy"`
	UpdatedAt     *string `json:"updatedAt"`
	UpdatedBy     *string `json:"updatedBy"`
}

// PlannedVolumeResponse структура ответа для плановых объёмов
type PlannedVolumeResponse struct {
	VolumeID       int     `json:"volumeId"`
	MaterialID     int     `json:"materialId"`
	MaterialCode   string  `json:"materialCode"`
	Shift          *string `json:"shift"`
	PlannedPerHour int     `json:"plannedPerHour"`
	MaxPerShift    int     `json:"maxPerShift"`
	IsActive       bool    `json:"isActive"`
	ValidFrom      string  `json:"validFrom"`
	ValidTo        *string `json:"validTo"`
}

// handlePlans обрабатывает запросы к /api/plans
func handlePlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetPlans(w, r)
	case http.MethodPost:
		handleCreatePlan(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPlans возвращает список планов
func handleGetPlans(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	dateFrom := query.Get("dateFrom")
	dateTo := query.Get("dateTo")
	materialIDStr := query.Get("materialId")
	shift := query.Get("shift")

	var planDateFrom, planDateTo *time.Time
	var materialID *int

	if dateFrom != "" {
		t, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			planDateFrom = &t
		}
	}
	if dateTo != "" {
		t, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			planDateTo = &t
		}
	}
	if materialIDStr != "" {
		id, err := strconv.Atoi(materialIDStr)
		if err == nil {
			materialID = &id
		}
	}
	var shiftPtr *string
	if shift != "" && shift != "all" {
		shiftPtr = &shift
	}

	plans, err := database.GetPlans(planDateFrom, planDateTo, materialID, shiftPtr)
	if err != nil {
		logger.Error("API /api/plans: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]PlanResponse, 0, len(plans))
	for _, p := range plans {
		resp := PlanResponse{
			PlanID:        p.PlanID,
			PlanDate:      p.PlanDate.Format("2006-01-02"),
			Shift:         p.Shift,
			MaterialID:    p.MaterialID,
			MaterialCode:  p.MaterialCode,
			PlannedAmount: p.PlannedAmount,
			ActualAmount:  p.ActualAmount,
			Status:        p.Status,
			CreatedAt:     p.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:     p.CreatedBy,
		}
		if p.UpdatedAt != nil {
			upd := p.UpdatedAt.Format("2006-01-02 15:04:05")
			resp.UpdatedAt = &upd
		}
		if p.UpdatedBy != nil {
			resp.UpdatedBy = p.UpdatedBy
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreatePlan создаёт новый план
func handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	materialID := req.MaterialID
	if materialID == 0 && req.MaterialCode != "" {
		material, err := database.GetMaterialByCode(req.MaterialCode)
		if err != nil {
			logger.Error("API /api/plans: material not found: %v", err)
			http.Error(w, "Material not found", http.StatusBadRequest)
			return
		}
		materialID = material.MaterialID
	}

	if materialID == 0 {
		http.Error(w, "MaterialID or MaterialCode required", http.StatusBadRequest)
		return
	}

	planDate, err := time.Parse("2006-01-02", req.PlanDate)
	if err != nil {
		http.Error(w, "Invalid planDate format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		createdBy = "api"
	}

	planID, err := database.CreatePlan(planDate, req.Shift, materialID, req.PlannedAmount, createdBy)
	if err != nil {
		logger.Error("API /api/plans (create): %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_ = database.UpdatePlansStatus(&planDate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"planId": planID,
	})
}

// handlePlanByID обрабатывает запросы к /api/plans/{id}
func handlePlanByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/plans/")
	idStr := strings.Split(path, "/")[0]

	planID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid plan ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetPlanByID(w, r, planID)
	case http.MethodPut:
		handleUpdatePlanByID(w, r, planID)
	case http.MethodDelete:
		handleDeletePlanByID(w, r, planID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPlanByID возвращает план по ID
func handleGetPlanByID(w http.ResponseWriter, r *http.Request, planID int) {
	plan, err := database.GetPlanByID(planID)
	if err != nil {
		logger.Error("API /api/plans/%d: %v", planID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.Error(w, "Plan not found", http.StatusNotFound)
		return
	}

	resp := PlanResponse{
		PlanID:        plan.PlanID,
		PlanDate:      plan.PlanDate.Format("2006-01-02"),
		Shift:         plan.Shift,
		MaterialID:    plan.MaterialID,
		MaterialCode:  plan.MaterialCode,
		PlannedAmount: plan.PlannedAmount,
		ActualAmount:  plan.ActualAmount,
		Status:        plan.Status,
		CreatedAt:     plan.CreatedAt.Format("2006-01-02 15:04:05"),
		CreatedBy:     plan.CreatedBy,
	}
	if plan.UpdatedAt != nil {
		upd := plan.UpdatedAt.Format("2006-01-02 15:04:05")
		resp.UpdatedAt = &upd
	}
	if plan.UpdatedBy != nil {
		resp.UpdatedBy = plan.UpdatedBy
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdatePlanByID обновляет план
func handleUpdatePlanByID(w http.ResponseWriter, r *http.Request, planID int) {
	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedBy := r.Header.Get("X-User")
	if updatedBy == "" {
		updatedBy = "api"
	}

	err := database.UpdatePlan(planID, req.PlannedAmount, updatedBy)
	if err != nil {
		logger.Error("API /api/plans/%d (update): %v", planID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = database.UpdatePlansStatus(nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"planId": planID,
	})
}

// handleDeletePlanByID удаляет план
func handleDeletePlanByID(w http.ResponseWriter, r *http.Request, planID int) {
	updatedBy := r.Header.Get("X-User")
	if updatedBy == "" {
		updatedBy = "api"
	}

	err := database.DeletePlan(planID, updatedBy)
	if err != nil {
		logger.Error("API /api/plans/%d (delete): %v", planID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

// handlePlannedVolumes возвращает справочник плановых объёмов
func handlePlannedVolumes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	volumes, err := database.GetPlannedVolumes()
	if err != nil {
		logger.Error("API /api/plans/volumes: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]PlannedVolumeResponse, 0, len(volumes))
	for _, v := range volumes {
		resp := PlannedVolumeResponse{
			VolumeID:       v.VolumeID,
			MaterialID:     v.MaterialID,
			MaterialCode:   v.MaterialCode,
			Shift:          v.Shift,
			PlannedPerHour: v.PlannedPerHour,
			MaxPerShift:    v.MaxPerShift,
			IsActive:       v.IsActive,
			ValidFrom:      v.ValidFrom.Format("2006-01-02"),
		}
		if v.ValidTo != nil {
			vt := v.ValidTo.Format("2006-01-02")
			resp.ValidTo = &vt
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdatePlansStatus принудительно обновляет статусы планов
func handleUpdatePlansStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PlanDate *string `json:"planDate"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var planDate *time.Time
	if req.PlanDate != nil && *req.PlanDate != "" {
		t, err := time.Parse("2006-01-02", *req.PlanDate)
		if err == nil {
			planDate = &t
		}
	}

	err := database.UpdatePlansStatus(planDate)
	if err != nil {
		logger.Error("API /api/plans/status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}
