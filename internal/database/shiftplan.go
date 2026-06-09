package database

import (
	"context"
	"fmt"
	"time"
)

// CompletedBoxInfo информация о готовой коробке
type CompletedBoxInfo struct {
	HUNumber string
	Amount   int
	Time     time.Time
}

// GetCompletedBoxesForShift возвращает фактические коробки за смену, сгруппированные по материалам
func GetCompletedBoxesForShift(lineName string, shiftDate time.Time, shift string) (map[string][]CompletedBoxInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Определяем временной интервал смены
	var startTime, endTime time.Time
	switch shift {
	case "1":
		startTime = time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 6, 30, 0, 0, shiftDate.Location())
		endTime = startTime.Add(8*time.Hour + 30*time.Minute)
	case "2":
		startTime = time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 15, 0, 0, 0, shiftDate.Location())
		endTime = startTime.Add(8*time.Hour + 30*time.Minute)
	case "3":
		startTime = time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 23, 30, 0, 0, shiftDate.Location())
		endTime = startTime.Add(8 * time.Hour)
	default:
		return nil, fmt.Errorf("неизвестная смена: %s", shift)
	}

	query := `
		SELECT 
			m.MaterialCode,
			h.HUNumber,
			h.Amount,
			hs.ChangedAt
		FROM HU h
		JOIN materials m ON h.MaterialID = m.MaterialID
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE hs.Status = N'Произведена'
		  AND hs.ChangedAt >= ?
		  AND hs.ChangedAt <= ?
		  AND hs.ChangedAt = (
		      SELECT MAX(ChangedAt) 
		      FROM HU_Status 
		      WHERE HUID = h.HUID
		  )
		ORDER BY m.MaterialCode, hs.ChangedAt ASC`

	rows, err := DB.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса коробок за смену: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]CompletedBoxInfo)
	for rows.Next() {
		var materialCode string
		var box CompletedBoxInfo
		err := rows.Scan(&materialCode, &box.HUNumber, &box.Amount, &box.Time)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования коробки: %w", err)
		}
		result[materialCode] = append(result[materialCode], box)
	}

	return result, nil
}
