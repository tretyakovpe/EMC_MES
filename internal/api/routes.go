package api

import (
	"net/http"
	"os"
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
	mux.HandleFunc("/api/plans/month", handleGetPlansByMonth)

	mux.HandleFunc("/api/shipments", handleShipments)
	mux.HandleFunc("/api/shipments/", handleShipmentByID)

	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/stats/summary", handleStatsSummary)
	mux.HandleFunc("/api/stats/production", handleStatsProduction)

	mux.HandleFunc("/api/statistics/boxes", handleGetBoxesStats)
	mux.HandleFunc("/api/statistics/bad-parts", handleGetBadPartsStats)
	mux.HandleFunc("/api/statistics/lines", handleGetLinesForFilter)

	mux.HandleFunc("/api/events", handleEvent)

	mux.HandleFunc("/api/plans/from-excel", handlePlansFromExcel)

	return mux
}

// serveStatic раздаёт статические файлы с правильными MIME-типами
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Убираем префикс /static/
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Защита от path traversal
	if strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join("./web/static", path)

	// Проверяем существование файла
	info, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Если это директория — отдаём 404
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Определяем MIME-тип по расширению (явно, без автоопределения)
	ext := filepath.Ext(fullPath)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".html", ".htm":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Отключаем MIME sniffing (важно для Windows)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Отключаем кэширование для статики
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	http.ServeFile(w, r, fullPath)
}

func serveProduction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./web/static/production.html")
}

func serveLogistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./web/static/logistics.html")
}

func serveQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./web/static/quality.html")
}
