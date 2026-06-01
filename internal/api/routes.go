package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"EMC_MES/internal/events"
)

var globalHub *events.Hub

// SetupRoutes настраивает все маршруты API
func SetupRoutes() *http.ServeMux {
	globalHub = events.NewHub()
	go globalHub.Run()

	mux := http.NewServeMux()

	// Статика с правильными MIME-типами
	mux.HandleFunc("/static/", serveStatic)

	// HTML страницы
	mux.HandleFunc("/production", serveProduction)
	mux.HandleFunc("/logistics", serveLogistics)
	mux.HandleFunc("/quality", serveQuality)
	mux.HandleFunc("/", serveProduction)

	// WebSocket
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(globalHub, w, r)
	})

	// API маршруты
	mux.HandleFunc("/api/lines", handleLines)
	mux.HandleFunc("/api/lines/status", handleLineStatus)
	mux.HandleFunc("/api/lines/", handleLineByID)

	mux.HandleFunc("/api/materials", handleMaterials)
	mux.HandleFunc("/api/materials/code", handleGetMaterialByCode)
	mux.HandleFunc("/api/materials/", handleMaterialByID)

	mux.HandleFunc("/api/boxes", handleBoxes)
	mux.HandleFunc("/api/boxes/stats", GetBoxesStats)
	mux.HandleFunc("/api/boxes/", handleBoxByID)

	mux.HandleFunc("/api/plans", handlePlans)
	mux.HandleFunc("/api/plans/volumes", handlePlannedVolumes)
	mux.HandleFunc("/api/plans/status", handleUpdatePlansStatus)
	mux.HandleFunc("/api/plans/", handlePlanByID)

	mux.HandleFunc("/api/shipments", handleShipments)
	mux.HandleFunc("/api/shipments/", handleShipmentByID)

	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/stats/summary", handleStatsSummary)
	mux.HandleFunc("/api/stats/production", handleStatsProduction)

	mux.HandleFunc("/api/statistics/boxes", handleGetBoxesStats)
	mux.HandleFunc("/api/statistics/bad-parts", handleGetBadPartsStats)
	mux.HandleFunc("/api/statistics/lines", handleGetLinesForFilter)

	mux.HandleFunc("/api/events", handleEvent)

	return mux
}

// serveStatic раздаёт статические файлы с правильными MIME-типами
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Убираем префикс /static/
	// Пример: /static/js/core/api.js -> js/core/api.js
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Защита от path traversal
	if strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join("./web/static", path)

	// Проверяем существование файла
	if !fileExists(fullPath) {
		http.NotFound(w, r)
		return
	}

	// Определяем MIME-тип по расширению
	ext := filepath.Ext(fullPath)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Отключаем X-Content-Type-Options для статики
	w.Header().Del("X-Content-Type-Options")

	http.ServeFile(w, r, fullPath)
}

// fileExists проверяет существование файла
func fileExists(path string) bool {
	info, err := http.Dir(".").Open(path)
	if err != nil {
		return false
	}
	info.Close()
	return true
}

func serveProduction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Del("X-Content-Type-Options")
	http.ServeFile(w, r, "./web/static/production.html")
}

func serveLogistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Del("X-Content-Type-Options")
	http.ServeFile(w, r, "./web/static/logistics.html")
}

func serveQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Del("X-Content-Type-Options")
	http.ServeFile(w, r, "./web/static/quality.html")
}
