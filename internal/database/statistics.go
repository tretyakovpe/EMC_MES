package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// BoxStatRecord запись о коробке для статистики
type BoxStatRecord struct {
	Date         string
	Time         string
	Label        string
	Line         string
	MaterialCode string
	Amount       int
}

// BadPartRecord запись о бракованной детали для статистики
type BadPartRecord struct {
	DateTime string
	Line     string
	Material string
	Counter  int
	Mkm      string
	Video    string
	Details  string
}

// GetBoxesByPeriod возвращает коробки за период с фильтром по линии
func GetBoxesByPeriod(fromDate, toDate time.Time, lineName string) ([]BoxStatRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT 
			p.date,
			CONVERT(VARCHAR(8), p.time, 108) as Time,
			RTRIM(p.label) as Label,
			RTRIM(p.line) as Line,
			RTRIM(p.material) as Material,
			p.amount as Amount
		FROM prod p
		WHERE p.date >= ? 
		  AND p.date <= ?
	`
	args := []interface{}{fromDate, toDate}

	if lineName != "" && lineName != "Все" {
		query += " AND RTRIM(p.line) = ?"
		args = append(args, lineName)
	}

	query += " ORDER BY p.date DESC, p.time DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса коробок: %w", err)
	}
	defer rows.Close()

	var records []BoxStatRecord
	for rows.Next() {
		var r BoxStatRecord
		var date time.Time
		var timeStr, label, line, material string
		var amount int

		err := rows.Scan(&date, &timeStr, &label, &line, &material, &amount)
		if err != nil {
			logger.Error("Ошибка сканирования коробки: %v", err)
			continue
		}

		// Форматируем дату из time.Time в строку YYYY-MM-DD
		r.Date = date.Format("2006-01-02")
		r.Time = timeStr
		r.Label = label
		r.Line = line
		r.MaterialCode = material
		r.Amount = amount

		records = append(records, r)
	}

	return records, nil
}

// GetBadPartsByPeriod возвращает бракованные детали за период с фильтром по линии
func GetBadPartsByPeriod(fromDate, toDate time.Time, lineName string) ([]BadPartRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Используем ? вместо именованных параметров
	query := `
		SELECT 
			FORMAT(pn.datetime, 'yyyy-MM-dd HH:mm:ss') as DateTime,
			RTRIM(pn.line) as Line,
			RTRIM(pn.name) as Material,
			pn.counter as Counter,
			pn.mkm as Mkm,
			pn.video as Video
		FROM partNok pn
		WHERE pn.datetime >= ? 
		  AND pn.datetime <= ?
	`
	args := []interface{}{fromDate, toDate}

	// Если lineName != "Все", добавляем фильтр
	if lineName != "" && lineName != "Все" {
		query += " AND RTRIM(pn.line) = ?"
		args = append(args, lineName)
	}

	query += " ORDER BY pn.datetime DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса бракованных деталей: %w", err)
	}
	defer rows.Close()

	var records []BadPartRecord
	for rows.Next() {
		var r BadPartRecord
		var mkmBytes []byte
		var video sql.NullString

		err := rows.Scan(&r.DateTime, &r.Line, &r.Material, &r.Counter, &mkmBytes, &video)
		if err != nil {
			logger.Error("Ошибка сканирования брака: %v", err)
			continue
		}

		// Преобразуем MKM байты в читаемую строку
		if len(mkmBytes) >= 4 {
			r.Mkm = fmt.Sprintf("%02X %02X %02X %02X", mkmBytes[0], mkmBytes[1], mkmBytes[2], mkmBytes[3])
		} else {
			r.Mkm = "—"
		}

		if video.Valid && video.String != "" && video.String != "0" {
			r.Video = video.String
		} else {
			r.Video = "—"
		}

		r.Details = "—"
		records = append(records, r)
	}

	return records, nil
}

// GetAllLinesForFilter возвращает список всех линий для выпадающего списка
func GetAllLinesForFilter() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT DISTINCT RTRIM([name]) as LineName
		FROM [dbo].[plc]
		ORDER BY LineName`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса линий: %w", err)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		lines = append(lines, line)
	}

	return lines, nil
}
