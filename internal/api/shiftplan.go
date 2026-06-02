package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// ShiftPlanMaterial структура материала в сменном задании
type ShiftPlanMaterial struct {
	MaterialCode    string `json:"materialCode"`
	PlannedAmount   int    `json:"plannedAmount"`
	PlannedBoxes    int    `json:"plannedBoxes"`
	CompletedAmount int    `json:"completedAmount"`
	CompletedBoxes  int    `json:"completedBoxes"`
	QuantityInHU    int    `json:"quantityInHU"`
}

// ShiftPlanResponse структура ответа
type ShiftPlanResponse struct {
	Line      string              `json:"line"`
	Date      string              `json:"date"`
	Shift     string              `json:"shift"`
	Materials []ShiftPlanMaterial `json:"materials"`
}

// handleShiftPlan возвращает сменное задание для линии
func handleShiftPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем имя линии из URL: /api/shiftplan/25
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Missing line name", http.StatusBadRequest)
		return
	}
	lineName := parts[len(parts)-1]

	// Определяем текущую смену
	now := time.Now()
	shift := getCurrentShift(now)

	// Получаем планы на сегодня
	today := now.Format("2006-01-02")
	plans, err := database.GetPlansForDateAndLine(today, shift, lineName)
	if err != nil {
		logger.Error("API /api/shiftplan/%s: %v", lineName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Получаем фактические коробки за смену (ключ map - MaterialCode, не ID!)
	completedBoxes, err := database.GetCompletedBoxesForShift(lineName, now, shift)
	if err != nil {
		logger.Error("API /api/shiftplan/%s (completed): %v", lineName, err)
		completedBoxes = make(map[string][]database.CompletedBoxInfo)
	}

	response := ShiftPlanResponse{
		Line:      lineName,
		Date:      today,
		Shift:     shift,
		Materials: []ShiftPlanMaterial{},
	}

	for _, plan := range plans {
		material, _ := database.GetMaterialByID(plan.MaterialID)
		quantityInHU := 50 // значение по умолчанию
		if material != nil && material.QuantityInHU > 0 {
			quantityInHU = material.QuantityInHU
		}

		// Округляем вверх до целых коробок
		plannedBoxes := (plan.PlannedAmount + quantityInHU - 1) / quantityInHU

		// Используем MaterialCode для поиска в map
		completedForMaterial := completedBoxes[plan.MaterialCode]
		completedBoxesCount := len(completedForMaterial)
		completedAmount := 0
		for _, box := range completedForMaterial {
			completedAmount += box.Amount
		}

		response.Materials = append(response.Materials, ShiftPlanMaterial{
			MaterialCode:    plan.MaterialCode,
			PlannedAmount:   plan.PlannedAmount,
			PlannedBoxes:    plannedBoxes,
			CompletedAmount: completedAmount,
			CompletedBoxes:  completedBoxesCount,
			QuantityInHU:    quantityInHU,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getCurrentShift определяет текущую смену по времени
func getCurrentShift(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 6 && hour < 14:
		return "1"
	case hour >= 14 && hour < 22:
		return "2"
	default:
		return "3"
	}
}
