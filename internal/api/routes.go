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

	// ==================== СТАТИКА (все файлы через /static/) ====================
	mux.HandleFunc("/static/", serveStatic)

	// ==================== HTML СТРАНИЦЫ (точные пути) ====================
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/production", serveProduction)
	mux.HandleFunc("/logistics", serveLogistics)
	mux.HandleFunc("/quality", serveQuality)
	mux.HandleFunc("/table-view", serveTableView) // без .html

	// ==================== ВЕБСОКЕТ ====================
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(globalHub, w, r)
	})

	// ==================== API МАРШРУТЫ ====================
	// Линии
	mux.HandleFunc("/api/lines", handleLines)
	mux.HandleFunc("/api/lines/status", handleLineStatus)
	mux.HandleFunc("/api/lines/", handleLineByID)

	// Статистика линий (отдельный эндпоинт)
	mux.HandleFunc("/api/linestats", handleGetAllLinesStats)
	mux.HandleFunc("/api/linestats/", handleGetLineStats)

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
	mux.HandleFunc("/api/plans/month", handleGetPlansByMonth)
	mux.HandleFunc("/api/plans/from-excel", handlePlansFromExcel)
	mux.HandleFunc("/api/plans/", handlePlanByID)

	// Сменное задание
	mux.HandleFunc("/api/shiftplan/", handleShiftPlan)

	// Отгрузки
	mux.HandleFunc("/api/shipments", handleShipments)
	//mux.HandleFunc("/api/shipments/", handleShipmentByID)

	// Статистика
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/stats/summary", handleStatsSummary)
	mux.HandleFunc("/api/stats/production", handleStatsProduction)

	// Статистика (старая версия)
	mux.HandleFunc("/api/statistics/boxes", handleGetBoxesStats)
	mux.HandleFunc("/api/statistics/bad-parts", handleGetBadPartsStats)
	mux.HandleFunc("/api/statistics/lines", handleGetLinesForFilter)

	// Склад
	mux.HandleFunc("/api/warehouse/stacks", handleWarehouseStacks)

	// События
	mux.HandleFunc("/api/events", handleEvent)

	// Сканирование
	RegisterScanRoutes(mux, globalHub)

	// Специальный маршрут для shiftplan (должен быть после всех API)
	mux.HandleFunc("/production/shiftplan/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/shiftplan.html")
	})
	return mux
}

// serveStatic раздаёт статические файлы
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	exeDir := filepath.Dir(exePath)

	// Убираем префикс /static/
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Защита от path traversal
	if strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(exeDir, "web", "static", path)

	// Проверяем существование файла
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// MIME тип по расширению
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
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, fullPath)
}

// HTML страницы
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/index.html")
}

func serveProduction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/production.html")
}

func serveLogistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/logistics.html")
}

func serveQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/quality.html")
}

func serveTableView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/table-view.html")
}
