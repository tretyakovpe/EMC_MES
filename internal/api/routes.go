package api

import (
	"net/http"

	"EMC_MES/internal/database"
	"EMC_MES/internal/events"
)

var globalHub *events.Hub

// SetupRoutes настраивает все маршруты API
func SetupRoutes() *http.ServeMux {
	// Инициализируем WebSocket Hub
	globalHub = events.NewHub()
	go globalHub.Run()
	database.SetHub(globalHub)

	mux := http.NewServeMux()

	// Статика (фронтенд)
	mux.Handle("/", http.FileServer(http.Dir("./web/static")))

	// WebSocket
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(globalHub, w, r)
	})

	// === API маршруты ===

	// Линии
	mux.HandleFunc("/api/lines", handleLines)
	mux.HandleFunc("/api/lines/status", handleLineStatus)
	mux.HandleFunc("/api/lines/", handleLineByID)

	// Материалы
	mux.HandleFunc("/api/materials", handleMaterials)
	mux.HandleFunc("/api/materials/code", handleGetMaterialByCode)
	mux.HandleFunc("/api/materials/", handleMaterialByID)

	// Коробки
	mux.HandleFunc("/api/boxes", handleBoxes)
	mux.HandleFunc("/api/boxes/stats", GetBoxesStats)
	mux.HandleFunc("/api/boxes/", handleBoxByID)

	// Планы
	mux.HandleFunc("/api/plans", handlePlans)
	mux.HandleFunc("/api/plans/volumes", handlePlannedVolumes)
	mux.HandleFunc("/api/plans/status", handleUpdatePlansStatus)
	mux.HandleFunc("/api/plans/", handlePlanByID)

	// Отгрузки
	mux.HandleFunc("/api/shipments", handleShipments)
	mux.HandleFunc("/api/shipments/", handleShipmentByID)

	// Статистика
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/stats/summary", handleStatsSummary)
	mux.HandleFunc("/api/stats/production", handleStatsProduction)

	// События (для Data Collector)
	mux.HandleFunc("/api/events", handleEvent)

	// Статистика
	mux.HandleFunc("/api/statistics/boxes", handleGetBoxesStats)
	mux.HandleFunc("/api/statistics/bad-parts", handleGetBadPartsStats)
	mux.HandleFunc("/api/statistics/lines", handleGetLinesForFilter)
	return mux
}

// GetHub возвращает глобальный Hub для отправки событий
func GetHub() *events.Hub {
	return globalHub
}
