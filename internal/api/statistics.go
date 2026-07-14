package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// BoxStatResponse структура ответа для коробок
type BoxStatResponse struct {
	Date         string `json:"date"`
	Time         string `json:"time"`
	Label        string `json:"label"`
	Line         string `json:"line"`
	MaterialCode string `json:"materialCode"`
	Amount       int    `json:"amount"`
}

// BadPartStatResponse структура ответа для брака
type BadPartStatResponse struct {
	ID       int    `json:"id"`
	DateTime string `json:"dateTime"`
	Line     string `json:"line"`
	Material string `json:"material"`
	Counter  int    `json:"counter"`
	Mkm      string `json:"mkm"`
	Video    string `json:"video"`
	Details  string `json:"details"`
}

// handleGetBoxesStats обрабатывает запрос /api/statistics/boxes
func handleGetBoxesStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим параметры
	query := r.URL.Query()
	fromDateStr := query.Get("fromDate")
	toDateStr := query.Get("toDate")
	lineName := query.Get("line")
	// Если lineName == "Все", передаём пустую строку в БД
	if lineName == "Все" {
		lineName = ""
	}
	// Устанавливаем даты по умолчанию (сегодня)
	now := time.Now()
	fromDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	toDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	if fromDateStr != "" {
		if t, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			fromDate = t
		} else {
			logger.Error("API /api/statistics/boxes: ошибка парсинга fromDate: %v", err)
		}
	}

	if toDateStr != "" {
		if t, err := time.Parse("2006-01-02", toDateStr); err == nil {
			toDate = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		} else {
			logger.Error("API /api/statistics/boxes: ошибка парсинга toDate: %v", err)
		}
	}

	// Получаем данные
	records, err := database.GetBoxesByPeriod(fromDate, toDate, lineName)
	if err != nil {
		logger.Error("API /api/statistics/boxes: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Конвертируем в response
	response := make([]BoxStatResponse, 0, len(records))
	for _, r := range records {
		response = append(response, BoxStatResponse{
			Date:         r.Date,
			Time:         r.Time,
			Label:        r.Label,
			Line:         r.Line,
			MaterialCode: r.MaterialCode,
			Amount:       r.Amount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetBadPartsStats обрабатывает запрос /api/statistics/bad-parts
func handleGetBadPartsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим параметры
	query := r.URL.Query()
	fromDateStr := query.Get("fromDate")
	toDateStr := query.Get("toDate")
	lineName := query.Get("line")

	// Устанавливаем даты по умолчанию (сегодня)
	now := time.Now()
	fromDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	toDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	if fromDateStr != "" {
		if t, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			fromDate = t
		} else {
			logger.Error("API /api/statistics/bad-parts: ошибка парсинга fromDate: %v", err)
		}
	}

	if toDateStr != "" {
		if t, err := time.Parse("2006-01-02", toDateStr); err == nil {
			toDate = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		} else {
			logger.Error("API /api/statistics/bad-parts: ошибка парсинга toDate: %v", err)
		}
	}

	// Получаем данные
	records, err := database.GetBadPartsByPeriod(fromDate, toDate, lineName)
	if err != nil {
		logger.Error("API /api/statistics/bad-parts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Конвертируем в response
	response := make([]BadPartStatResponse, 0, len(records))
	for _, r := range records {
		response = append(response, BadPartStatResponse{
			ID:       r.ID,
			DateTime: r.DateTime,
			Line:     r.Line,
			Material: r.Material,
			Counter:  r.Counter,
			Mkm:      r.Mkm,
			Video:    r.Video,
			Details:  r.Details,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetLinesForFilter обрабатывает запрос /api/statistics/lines
func handleGetLinesForFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lines, err := database.GetAllLinesForFilter()
	if err != nil {
		logger.Error("API /api/statistics/lines: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Добавляем пункт "Все"
	response := []string{"Все"}
	response = append(response, lines...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetVideo возвращает видеофайл для просмотра
func handleGetVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	// Путь к папке video в Data Collector

	dataCollectorDir := `C:\ProductionManagement\DataCollector`
	videoDir := filepath.Join(dataCollectorDir, "video")
	fullPath := filepath.Join(videoDir, filename)

	// Проверяем существование файла
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.Error("Видео в %s не найдено", fullPath)
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	// Отдаём файл
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeFile(w, r, fullPath)
}
