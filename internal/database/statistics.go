package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// ShippingScreenItem структура для экрана отгрузок
type ShippingScreenItem struct {
	CustomerCode string `json:"customerCode"`
	MaterialCode string `json:"materialCode"`
	BoxAmount    int    `json:"boxAmount"`
	Destination  string `json:"destination"`
	Today        int    `json:"today"`
	Tomorrow     int    `json:"tomorrow"`
	DayAfter     int    `json:"dayAfter"`
}

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
	ID       int
	DateTime string
	Line     string
	Material string
	Counter  int
	Mkm      string
	Video    string
	Details  string
}

// PartNokRecord структура для записи о бракованной детали
type PartNokRecord struct {
	ID       int       `json:"id"`
	Datetime time.Time `json:"datetime"`
	Line     string    `json:"line"`
	Material string    `json:"material"`
	Counter  int       `json:"counter"`
	Mkm      []byte    `json:"mkm"`
	Video    string    `json:"video"`
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
			pn.id as Id,
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

		err := rows.Scan(&r.ID, &r.DateTime, &r.Line, &r.Material, &r.Counter, &mkmBytes, &video)
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

func GetPartNokByID(id string) (*PartNokRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        SELECT 
            id,
            datetime,
            line,
            name,
            counter,
            mkm,
            video
        FROM partNok
        WHERE id = ?
    `
	var p PartNokRecord
	err := DB.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Datetime, &p.Line, &p.Material,
		&p.Counter, &p.Mkm, &p.Video,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetShippingScreenData возвращает данные для экрана отгрузок
func GetShippingScreenData() ([]ShippingScreenItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
        SELECT 
            RTRIM(m.CustomerCode) as CustomerCode,
            RTRIM(m.MaterialCode) as MaterialCode,
            m.QuantityInHU as BoxAmount,
            RTRIM(m.Destination) as Destination,
            ISNULL(SUM(CASE 
                WHEN s.Date = CAST(GETDATE() AS DATE) AND s.Done = 0 
                THEN sd.Boxes*m.QuantityInHU
                ELSE 0 
            END), 0) as Today,
            ISNULL(SUM(CASE 
                WHEN s.Date = CAST(DATEADD(day, 1, GETDATE()) AS DATE) AND s.Done = 0 
                THEN sd.Boxes*m.QuantityInHU
                ELSE 0 
            END), 0) as Tomorrow,
            ISNULL(SUM(CASE 
                WHEN s.Date = CAST(DATEADD(day, 2, GETDATE()) AS DATE) AND s.Done = 0 
                THEN sd.Boxes*m.QuantityInHU
                ELSE 0 
            END), 0) as DayAfter
        FROM materials m
        LEFT JOIN ShipmentDetails sd ON m.MaterialID = sd.MaterialID
        LEFT JOIN Shipments s ON sd.ShipmentID = s.ShipmentID
        WHERE s.Done = 0 OR s.Done IS NULL
        GROUP BY m.CustomerCode, m.MaterialCode, m.QuantityInHU, m.Destination
        HAVING 
            SUM(CASE WHEN s.Date = CAST(GETDATE() AS DATE) AND s.Done = 0 THEN sd.Boxes - sd.ScannedBoxes ELSE 0 END) > 0
            OR SUM(CASE WHEN s.Date = CAST(DATEADD(day, 1, GETDATE()) AS DATE) AND s.Done = 0 THEN sd.Boxes - sd.ScannedBoxes ELSE 0 END) > 0
            OR SUM(CASE WHEN s.Date = CAST(DATEADD(day, 2, GETDATE()) AS DATE) AND s.Done = 0 THEN sd.Boxes - sd.ScannedBoxes ELSE 0 END) > 0
        ORDER BY m.MaterialCode`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса данных отгрузок: %w", err)
	}
	defer rows.Close()

	var items []ShippingScreenItem
	for rows.Next() {
		var item ShippingScreenItem
		err := rows.Scan(
			&item.CustomerCode,
			&item.MaterialCode,
			&item.BoxAmount,
			&item.Destination,
			&item.Today,
			&item.Tomorrow,
			&item.DayAfter,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}
