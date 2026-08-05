package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	mux.HandleFunc("/table-view", serveTableView)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/images/favicon.ico", http.StatusMovedPermanently)
	})

	// ==================== ВЕБСОКЕТ ====================
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(globalHub, w, r)
	})

	// ==================== API МАРШРУТЫ ====================
	// Боксы (коробки)
	mux.HandleFunc("/api/boxes", handleBoxes)
	mux.HandleFunc("/api/boxes/", handleBoxByID)
	mux.HandleFunc("/api/boxes/stats", GetBoxesStats)
	mux.HandleFunc("/api/boxes/view/", handleViewLabel)

	// События
	mux.HandleFunc("/api/events", handleEvent)

	// Линии
	mux.HandleFunc("/api/lines", handleLines)
	mux.HandleFunc("/api/lines/", handleLineByID)
	mux.HandleFunc("/api/lines/status", handleLineStatus)

	// Статистика линий
	mux.HandleFunc("/api/linestats", handleGetAllLinesStats)
	mux.HandleFunc("/api/linestats/", handleGetLineStats)

	// Материалы
	mux.HandleFunc("/api/materials", handleMaterials)
	mux.HandleFunc("/api/materials/", handleMaterialByID)
	mux.HandleFunc("/api/materials/code", handleGetMaterialByCode)

	// Планы
	mux.HandleFunc("/api/plans", handlePlans)
	mux.HandleFunc("/api/plans/", handlePlanByID)
	mux.HandleFunc("/api/plans/from-excel", handlePlansFromExcel)
	mux.HandleFunc("/api/plans/month", handleGetPlansByMonth)
	mux.HandleFunc("/api/plans/status", handleUpdatePlansStatus)
	mux.HandleFunc("/api/plans/volumes", handlePlannedVolumes)

	// Сканирование
	RegisterScanRoutes(mux, globalHub)

	// Отгрузки
	mux.HandleFunc("/api/shipments", handleShipments)
	mux.HandleFunc("/api/shipments/", handleShipmentsByID)
	mux.HandleFunc("/api/shipments/parse-clipboard", handleParseClipboard)

	// Экран отгрузок
	mux.HandleFunc("/api/shipping-screen", handleShippingScreen)

	// Сменное задание
	mux.HandleFunc("/api/shiftplan/", handleShiftPlan)

	// Статистика
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/stats/production", handleStatsProduction)
	mux.HandleFunc("/api/stats/summary", handleStatsSummary)

	// Статистика (старая версия)
	mux.HandleFunc("/api/statistics/bad-parts", handleGetBadPartsStats)
	mux.HandleFunc("/api/statistics/boxes", handleGetBoxesStats)
	mux.HandleFunc("/api/statistics/lines", handleGetLinesForFilter)

	// ==================== ЗАКАЗЫ НА ПЕРЕМЕЩЕНИЕ (НОВЫЕ) ====================
	mux.HandleFunc("/api/transfer-orders", handleTransferOrders)
	mux.HandleFunc("/api/transfer-orders/", handleTransferOrderByID)

	// Перемещения — фактические отгрузки
	mux.HandleFunc("/api/transfer-shipments", handleTransferShipments)
	mux.HandleFunc("/api/transfer-shipments/", handleTransferShipmentsByID)

	// Видео стрим
	mux.HandleFunc("/api/video/stream", handleVideoStream)

	// Склады
	mux.HandleFunc("/api/warehouses", handleWarehouses)
	mux.HandleFunc("/api/warehouses/", handleWarehouseByID)

	// Склад ГП
	mux.HandleFunc("/api/warehouse/stacks", handleWarehouseStacks)

	// ==================== HTML СТРАНИЦЫ ДЛЯ УПРАВЛЕНИЯ ====================
	mux.HandleFunc("/materials", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/materials.html")
	})

	mux.HandleFunc("/production/shiftplan/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/shiftplan.html")
	})

	mux.HandleFunc("/shipping-screen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/shipping-screen.html")
	})

	mux.HandleFunc("/warehouses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/warehouses.html")
	})

	// СТРАНИЦА ПЕРЕМЕЩЕНИЙ (для кладовщика)
	mux.HandleFunc("/transfers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./web/static/transfers.html")
	})

	return mux
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ====================

// serveIndex возвращает главную страницу
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/index.html")
}

// serveLogistics возвращает страницу логистики
func serveLogistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/logistics.html")
}

// serveProduction возвращает страницу производства
func serveProduction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/production.html")
}

// serveQuality возвращает страницу качества
func serveQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/quality.html")
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

// serveTableView возвращает страницу табличного представления
func serveTableView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/static/table-view.html")
}

// ==================== ОБРАБОТЧИКИ МАРШРУТОВ С ПАРАМЕТРАМИ ====================

// handleShipmentsByID обрабатывает запросы к /api/shipments/{id} и его подпути
func handleShipmentsByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/shipments/")

	// Обработка /api/shipments/{id}/complete
	if strings.HasSuffix(path, "/complete") {
		idStr := strings.TrimSuffix(path, "/complete")
		shipmentID, err := strconv.Atoi(idStr)
		if err == nil {
			handleCompleteShipment(w, r, shipmentID)
			return
		}
	}

	// Обработка /api/shipments/{id}/scanned
	if strings.HasSuffix(path, "/scanned") {
		idStr := strings.TrimSuffix(path, "/scanned")
		shipmentID, err := strconv.Atoi(idStr)
		if err == nil {
			handleGetScannedBoxes(w, r, shipmentID)
			return
		}
	}

	// Обработка /api/shipments/{id} - просмотр/удаление/обновление
	if len(path) > 0 && path != "" {
		shipmentID, err := strconv.Atoi(path)
		if err == nil {
			switch r.Method {
			case http.MethodGet:
				handleGetShipmentByID(w, r, shipmentID)
			case http.MethodDelete:
				handleDeleteShipment(w, r, shipmentID)
			case http.MethodPut:
				handleUpdateShipment(w, r, shipmentID)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
	}

	http.NotFound(w, r)
}

// handleTransferShipmentsByID обрабатывает запросы к /api/transfer-shipments/{id}
func handleTransferShipmentsByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/transfer-shipments/")

	// /api/transfer-shipments/{id}/grouped
	if strings.HasSuffix(path, "/grouped") {
		handleGetTransferShipmentsGrouped(w, r)
		return
	}

	// /api/transfer-shipments/{id} - получение или удаление
	if len(path) > 0 && path != "" {
		switch r.Method {
		case http.MethodGet:
			handleGetTransferShipments(w, r)
		case http.MethodDelete:
			handleDeleteTransferShipment(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}
