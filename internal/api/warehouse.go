package api

import (
	"encoding/json"
	"net/http"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// StackResponse структура для стека на складе
type StackResponse struct {
	MaterialCode string `json:"materialCode"`
	MaterialID   int    `json:"materialId"`
	BoxCount     int    `json:"boxCount"`
	TotalAmount  int    `json:"totalAmount"`
}

// handleWarehouseStacks возвращает данные для склада ГП
func handleWarehouseStacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем сгруппированные коробки со статусом "Произведена"
	stacks, err := database.GetWarehouseStacks()
	if err != nil {
		logger.Error("API /api/warehouse/stacks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stacks)
}
