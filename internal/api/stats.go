package api

import (
	"encoding/json"
	"net/http"
	"time"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
)

// StatsResponse структура ответа статистики
type StatsResponse struct {
	Period struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`

	// Производство
	TotalBoxes      int            `json:"totalBoxes"`
	TotalParts      int            `json:"totalParts"`
	PartsByLine     map[string]int `json:"partsByLine"`
	PartsByMaterial map[string]int `json:"partsByMaterial"`

	// Статус линий
	LinesTotal   int `json:"linesTotal"`
	LinesOnline  int `json:"linesOnline"`
	LinesOffline int `json:"linesOffline"`
	LinesActive  int `json:"linesActive"`

	// Отгрузки
	ShipmentsTotal     int `json:"shipmentsTotal"`
	ShipmentsCompleted int `json:"shipmentsCompleted"`
	ShipmentsDone      int `json:"shipmentsDone"`

	// Планы
	PlansTotal     int `json:"plansTotal"`
	PlansCompleted int `json:"plansCompleted"`
	PlansInWork    int `json:"plansInWork"`
}

// handleStats обрабатывает запросы к /api/stats
func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим параметры дат
	fromDateStr := r.URL.Query().Get("from")
	toDateStr := r.URL.Query().Get("to")

	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("2006-01-02", fromDateStr)
		if err != nil {
			http.Error(w, "Invalid from date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	} else {
		fromDate = time.Now().AddDate(0, 0, -30)
	}

	if toDateStr != "" {
		toDate, err = time.Parse("2006-01-02", toDateStr)
		if err != nil {
			http.Error(w, "Invalid to date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		toDate = toDate.Add(24*time.Hour - time.Second)
	} else {
		toDate = time.Now()
	}

	// Собираем статистику
	stats := StatsResponse{}
	stats.Period.From = fromDate.Format("2006-01-02")
	stats.Period.To = toDate.Format("2006-01-02")

	// 1. Статистика по коробкам и деталям
	boxesCount, err := database.GetProducedBoxesCount(fromDate, toDate)
	if err != nil {
		logger.Error("API /api/stats (boxes): %v", err)
	} else {
		stats.TotalBoxes = boxesCount
	}

	partsByMaterial, err := database.GetProducedAmountByMaterial(fromDate, toDate)
	if err != nil {
		logger.Error("API /api/stats (parts by material): %v", err)
	} else {
		stats.PartsByMaterial = partsByMaterial
		totalParts := 0
		for _, v := range partsByMaterial {
			totalParts += v
		}
		stats.TotalParts = totalParts
	}

	// 2. Статистика по линиям
	lines, err := database.GetAllLines()
	if err != nil {
		logger.Error("API /api/stats (lines): %v", err)
	} else {
		stats.LinesTotal = len(lines)
		stats.PartsByLine = make(map[string]int)

		for _, line := range lines {
			if line.IsOnline {
				stats.LinesOnline++
			} else {
				stats.LinesOffline++
			}
			if line.IsActive {
				stats.LinesActive++
			}
		}
	}

	// 3. Статистика по отгрузкам
	shipments, err := database.GetShipments(nil, nil, nil, nil)
	if err != nil {
		logger.Error("API /api/stats (shipments): %v", err)
	} else {
		stats.ShipmentsTotal = len(shipments)
		for _, s := range shipments {
			if s.Completed {
				stats.ShipmentsCompleted++
			}
			if s.Done {
				stats.ShipmentsDone++
			}
		}
	}

	// 4. Статистика по планам
	plans, err := database.GetPlans(&fromDate, &toDate, nil, nil)
	if err != nil {
		logger.Error("API /api/stats (plans): %v", err)
	} else {
		stats.PlansTotal = len(plans)
		for _, p := range plans {
			switch p.Status {
			case "Выполнен":
				stats.PlansCompleted++
			case "В работе":
				stats.PlansInWork++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleStatsSummary возвращает краткую статистику для виджетов на главной
func handleStatsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	todayEnd := todayStart.Add(24*time.Hour - time.Second)

	// Коробки за сегодня
	boxesToday, err := database.GetProducedBoxesCount(todayStart, todayEnd)
	if err != nil {
		logger.Error("API /api/stats/summary (boxes): %v", err)
		boxesToday = 0
	}

	// Детали за сегодня
	partsToday, err := database.GetProducedAmountByMaterial(todayStart, todayEnd)
	if err != nil {
		logger.Error("API /api/stats/summary (parts): %v", err)
	}
	totalPartsToday := 0
	for _, v := range partsToday {
		totalPartsToday += v
	}

	// Линии онлайн
	lines, err := database.GetAllLines()
	if err != nil {
		logger.Error("API /api/stats/summary (lines): %v", err)
	}
	linesOnline := 0
	for _, line := range lines {
		if line.IsOnline {
			linesOnline++
		}
	}

	// Активные отгрузки
	activeShipments := 0
	shipments, err := database.GetShipments(nil, nil, nil, nil)
	if err == nil {
		for _, s := range shipments {
			if !s.Done {
				activeShipments++
			}
		}
	}

	response := map[string]interface{}{
		"boxesToday":      boxesToday,
		"partsToday":      totalPartsToday,
		"linesOnline":     linesOnline,
		"linesTotal":      len(lines),
		"activeShipments": activeShipments,
		"timestamp":       time.Now().Format("2006-01-02 15:04:05"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStatsProduction возвращает график производства по дням
func handleStatsProduction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := time.ParseDuration(daysStr + "d"); err == nil {
			days = int(d.Hours() / 24)
		}
	}

	// Генерируем массив дат
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	type DailyStats struct {
		Date  string `json:"date"`
		Boxes int    `json:"boxes"`
		Parts int    `json:"parts"`
	}

	stats := make([]DailyStats, 0, days)

	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		dayEnd := dayStart.Add(24*time.Hour - time.Second)

		boxes, _ := database.GetProducedBoxesCount(dayStart, dayEnd)
		partsByMaterial, _ := database.GetProducedAmountByMaterial(dayStart, dayEnd)

		totalParts := 0
		for _, v := range partsByMaterial {
			totalParts += v
		}

		stats = append(stats, DailyStats{
			Date:  d.Format("2006-01-02"),
			Boxes: boxes,
			Parts: totalParts,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
