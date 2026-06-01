package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"EMC_MES/internal/logger"
)

// HU представляет коробку (Handling Unit)
type HU struct {
	HUID       int
	MaterialID int
	HUNumber   *string
	Amount     int
	ShipmentID *int
}

// HUStatus представляет статус коробки в истории
type HUStatus struct {
	ID        int
	HUID      int
	Status    string
	ChangedAt time.Time
}

// BoxWithStatus объединяет коробку и её текущий статус
type BoxWithStatus struct {
	HU
	CurrentStatus string
	MaterialCode  string
}

// GetBoxesByStatus возвращает коробки с определённым статусом
func GetBoxesByStatus(status string, limit int) ([]BoxWithStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			h.HUID,
			h.MaterialID,
			h.HUNumber,
			h.Amount,
			h.ShipmentID,
			m.MaterialCode,
			hs.Status as CurrentStatus
		FROM HU h
		JOIN materials m ON h.MaterialID = m.MaterialID
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE hs.Status = ?
		AND hs.ChangedAt = (
			SELECT MAX(ChangedAt) 
			FROM HU_Status 
			WHERE HUID = h.HUID
		)
		ORDER BY hs.ChangedAt DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", limit)
	}

	rows, err := DB.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса коробок: %w", err)
	}
	defer rows.Close()

	var boxes []BoxWithStatus
	for rows.Next() {
		var b BoxWithStatus
		var huNumber sql.NullString
		var shipmentID sql.NullInt32

		err := rows.Scan(
			&b.HUID,
			&b.MaterialID,
			&huNumber,
			&b.Amount,
			&shipmentID,
			&b.MaterialCode,
			&b.CurrentStatus,
		)
		if err != nil {
			logger.Error("Ошибка сканирования коробки: %v", err)
			continue
		}

		if huNumber.Valid {
			b.HUNumber = &huNumber.String
		}
		if shipmentID.Valid {
			id := int(shipmentID.Int32)
			b.ShipmentID = &id
		}

		boxes = append(boxes, b)
	}

	return boxes, nil
}

// GetBoxesByMaterial возвращает коробки по материалу
func GetBoxesByMaterial(materialID int, status string) ([]BoxWithStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			h.HUID,
			h.MaterialID,
			h.HUNumber,
			h.Amount,
			h.ShipmentID,
			m.MaterialCode,
			hs.Status as CurrentStatus
		FROM HU h
		JOIN materials m ON h.MaterialID = m.MaterialID
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE h.MaterialID = ?
		AND hs.ChangedAt = (
			SELECT MAX(ChangedAt) 
			FROM HU_Status 
			WHERE HUID = h.HUID
		)
	`

	if status != "" {
		query += " AND hs.Status = ?"
	}

	query += " ORDER BY hs.ChangedAt DESC"

	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = DB.QueryContext(ctx, query, materialID, status)
	} else {
		rows, err = DB.QueryContext(ctx, query, materialID)
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка запроса коробок: %w", err)
	}
	defer rows.Close()

	var boxes []BoxWithStatus
	for rows.Next() {
		var b BoxWithStatus
		var huNumber sql.NullString
		var shipmentID sql.NullInt32

		err := rows.Scan(
			&b.HUID,
			&b.MaterialID,
			&huNumber,
			&b.Amount,
			&shipmentID,
			&b.MaterialCode,
			&b.CurrentStatus,
		)
		if err != nil {
			logger.Error("Ошибка сканирования коробки: %v", err)
			continue
		}

		if huNumber.Valid {
			b.HUNumber = &huNumber.String
		}
		if shipmentID.Valid {
			id := int(shipmentID.Int32)
			b.ShipmentID = &id
		}

		boxes = append(boxes, b)
	}

	return boxes, nil
}

// GetBoxByHUNumber возвращает коробку по номеру бирки
func GetBoxByHUNumber(huNumber string) (*BoxWithStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			h.HUID,
			h.MaterialID,
			h.HUNumber,
			h.Amount,
			h.ShipmentID,
			m.MaterialCode,
			hs.Status as CurrentStatus
		FROM HU h
		JOIN materials m ON h.MaterialID = m.MaterialID
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE h.HUNumber = ?
		AND hs.ChangedAt = (
			SELECT MAX(ChangedAt) 
			FROM HU_Status 
			WHERE HUID = h.HUID
		)`

	var b BoxWithStatus
	var huNumberDB sql.NullString
	var shipmentID sql.NullInt32

	err := DB.QueryRowContext(ctx, query, huNumber).Scan(
		&b.HUID,
		&b.MaterialID,
		&huNumberDB,
		&b.Amount,
		&shipmentID,
		&b.MaterialCode,
		&b.CurrentStatus,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска коробки: %w", err)
	}

	if huNumberDB.Valid {
		b.HUNumber = &huNumberDB.String
	}
	if shipmentID.Valid {
		id := int(shipmentID.Int32)
		b.ShipmentID = &id
	}

	return &b, nil
}

// GetBoxStatusHistory возвращает историю статусов коробки
func GetBoxStatusHistory(huID int) ([]HUStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, HUID, Status, ChangedAt
		FROM HU_Status
		WHERE HUID = ?
		ORDER BY ChangedAt ASC`

	rows, err := DB.QueryContext(ctx, query, huID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса истории статусов: %w", err)
	}
	defer rows.Close()

	var history []HUStatus
	for rows.Next() {
		var hs HUStatus
		err := rows.Scan(&hs.ID, &hs.HUID, &hs.Status, &hs.ChangedAt)
		if err != nil {
			logger.Error("Ошибка сканирования статуса: %v", err)
			continue
		}
		history = append(history, hs)
	}

	return history, nil
}

// GetProducedBoxesCount возвращает количество произведённых коробок за период
func GetProducedBoxesCount(fromDate, toDate time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT COUNT(DISTINCT h.HUID)
		FROM HU h
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE hs.Status = N'Произведена'
		AND hs.ChangedAt >= ?
		AND hs.ChangedAt <= ?`

	var count int
	err := DB.QueryRowContext(ctx, query, fromDate, toDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчёта коробок: %w", err)
	}
	return count, nil
}

// GetProducedAmountByMaterial возвращает количество произведённых деталей по материалам за период
func GetProducedAmountByMaterial(fromDate, toDate time.Time) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			m.MaterialCode,
			SUM(h.Amount) as TotalAmount
		FROM HU h
		JOIN HU_Status hs ON h.HUID = hs.HUID
		JOIN materials m ON h.MaterialID = m.MaterialID
		WHERE hs.Status = N'Произведена'
		AND hs.ChangedAt >= ?
		AND hs.ChangedAt <= ?
		GROUP BY m.MaterialCode
		ORDER BY m.MaterialCode`

	rows, err := DB.QueryContext(ctx, query, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса статистики: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var materialCode string
		var amount int
		if err := rows.Scan(&materialCode, &amount); err != nil {
			logger.Error("Ошибка сканирования статистики: %v", err)
			continue
		}
		result[materialCode] = amount
	}

	return result, nil
}

// GetBoxesGroupedByMaterial возвращает коробки сгруппированные по материалам (для склада ГП)
func GetBoxesGroupedByMaterial() ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			m.MaterialCode,
			m.Description,
			COUNT(h.HUID) as BoxCount,
			SUM(h.Amount) as TotalAmount
		FROM HU h
		JOIN HU_Status hs ON h.HUID = hs.HUID
		JOIN materials m ON h.MaterialID = m.MaterialID
		WHERE hs.Status = N'Произведена'
		AND hs.ChangedAt = (
			SELECT MAX(ChangedAt) 
			FROM HU_Status 
			WHERE HUID = h.HUID
		)
		GROUP BY m.MaterialCode, m.Description
		ORDER BY m.MaterialCode`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка группировки коробок: %w", err)
	}
	defer rows.Close()

	var groups []map[string]interface{}
	for rows.Next() {
		var materialCode, description string
		var boxCount, totalAmount int

		err := rows.Scan(&materialCode, &description, &boxCount, &totalAmount)
		if err != nil {
			logger.Error("Ошибка сканирования группы: %v", err)
			continue
		}

		group := map[string]interface{}{
			"materialCode": strings.TrimSpace(materialCode),
			"description":  strings.TrimSpace(description),
			"boxCount":     boxCount,
			"totalAmount":  totalAmount,
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetBoxesByShipment возвращает коробки, привязанные к отгрузке
func GetBoxesByShipment(shipmentID int) ([]BoxWithStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			h.HUID,
			h.MaterialID,
			h.HUNumber,
			h.Amount,
			h.ShipmentID,
			m.MaterialCode,
			hs.Status as CurrentStatus
		FROM HU h
		JOIN materials m ON h.MaterialID = m.MaterialID
		JOIN HU_Status hs ON h.HUID = hs.HUID
		WHERE h.ShipmentID = ?
		AND hs.ChangedAt = (
			SELECT MAX(ChangedAt) 
			FROM HU_Status 
			WHERE HUID = h.HUID
		)
		ORDER BY hs.ChangedAt DESC`

	rows, err := DB.QueryContext(ctx, query, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса коробок в отгрузке: %w", err)
	}
	defer rows.Close()

	var boxes []BoxWithStatus
	for rows.Next() {
		var b BoxWithStatus
		var huNumber sql.NullString
		var shipmentIDVal sql.NullInt32

		err := rows.Scan(
			&b.HUID,
			&b.MaterialID,
			&huNumber,
			&b.Amount,
			&shipmentIDVal,
			&b.MaterialCode,
			&b.CurrentStatus,
		)
		if err != nil {
			logger.Error("Ошибка сканирования коробки: %v", err)
			continue
		}

		if huNumber.Valid {
			b.HUNumber = &huNumber.String
		}
		if shipmentIDVal.Valid {
			id := int(shipmentIDVal.Int32)
			b.ShipmentID = &id
		}

		boxes = append(boxes, b)
	}

	return boxes, nil
}
