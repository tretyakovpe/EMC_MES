package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// Shipment представляет отгрузочную накладную
type Shipment struct {
	ShipmentID int
	Number     *int
	Date       time.Time
	Completed  bool
	Done       bool
}

// ShipmentDetail представляет строку отгрузки
type ShipmentDetail struct {
	ShipmentDetailID int
	ShipmentID       int
	MaterialID       int
	MaterialCode     string
	Boxes            int
	Amount           int
	ScannedBoxes     int
}

// ShipmentWithDetails объединяет отгрузку и её детали
type ShipmentWithDetails struct {
	Shipment
	Details []ShipmentDetail
}

// GetShipments возвращает список всех отгрузок
func GetShipments(completed, done *bool, fromDate, toDate *time.Time) ([]Shipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT ShipmentID, Number, Date, Completed, Done
		FROM Shipments
		WHERE 1=1
	`
	args := []interface{}{}

	if completed != nil {
		query += " AND Completed = ?"
		args = append(args, *completed)
	}
	if done != nil {
		query += " AND Done = ?"
		args = append(args, *done)
	}
	if fromDate != nil {
		query += " AND Date >= ?"
		args = append(args, *fromDate)
	}
	if toDate != nil {
		query += " AND Date <= ?"
		args = append(args, *toDate)
	}

	query += " ORDER BY Date DESC, ShipmentID DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса отгрузок: %w", err)
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var s Shipment
		var number sql.NullInt32

		err := rows.Scan(&s.ShipmentID, &number, &s.Date, &s.Completed, &s.Done)
		if err != nil {
			logger.Error("Ошибка сканирования Shipment: %v", err)
			continue
		}
		if number.Valid {
			n := int(number.Int32)
			s.Number = &n
		}
		shipments = append(shipments, s)
	}

	return shipments, nil
}

// GetShipmentByID возвращает отгрузку с деталями
func GetShipmentByID(shipmentID int) (*ShipmentWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем заголовок
	var shipment Shipment
	var number sql.NullInt32

	err := DB.QueryRowContext(ctx, `
		SELECT ShipmentID, Number, Date, Completed, Done
		FROM Shipments
		WHERE ShipmentID = ?
	`, shipmentID).Scan(
		&shipment.ShipmentID, &number, &shipment.Date, &shipment.Completed, &shipment.Done,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения отгрузки: %w", err)
	}
	if number.Valid {
		n := int(number.Int32)
		shipment.Number = &n
	}

	// Получаем детали
	rows, err := DB.QueryContext(ctx, `
		SELECT 
			sd.ShipmentDetailID,
			sd.ShipmentID,
			sd.MaterialID,
			m.MaterialCode,
			sd.Boxes,
			sd.Amount,
			sd.ScannedBoxes
		FROM ShipmentDetails sd
		JOIN materials m ON sd.MaterialID = m.MaterialID
		WHERE sd.ShipmentID = ?
		ORDER BY m.MaterialCode
	`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения деталей отгрузки: %w", err)
	}
	defer rows.Close()

	var details []ShipmentDetail
	for rows.Next() {
		var d ShipmentDetail
		err := rows.Scan(
			&d.ShipmentDetailID,
			&d.ShipmentID,
			&d.MaterialID,
			&d.MaterialCode,
			&d.Boxes,
			&d.Amount,
			&d.ScannedBoxes,
		)
		if err != nil {
			logger.Error("Ошибка сканирования ShipmentDetail: %v", err)
			continue
		}
		details = append(details, d)
	}

	return &ShipmentWithDetails{
		Shipment: shipment,
		Details:  details,
	}, nil
}

// CreateShipment создаёт новую отгрузку
func CreateShipment(number *int, date time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var numberParam interface{} = nil
	if number != nil {
		numberParam = *number
	}

	query := `
		INSERT INTO Shipments (Number, Date, Completed, Done)
		VALUES (?, ?, 0, 0);
		SELECT SCOPE_IDENTITY();
	`

	var shipmentID int
	err := DB.QueryRowContext(ctx, query, numberParam, date).Scan(&shipmentID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания отгрузки: %w", err)
	}

	logger.Info("[DB] Создана отгрузка ID=%d, Number=%v", shipmentID, number)
	return shipmentID, nil
}

// AddShipmentDetail добавляет строку в отгрузку
func AddShipmentDetail(shipmentID, materialID, boxes, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO ShipmentDetails (ShipmentID, MaterialID, Boxes, Amount, ScannedBoxes)
		VALUES (?, ?, ?, ?, 0)
	`

	_, err := DB.ExecContext(ctx, query, shipmentID, materialID, boxes, amount)
	if err != nil {
		return fmt.Errorf("ошибка добавления детали отгрузки: %w", err)
	}

	logger.Info("[DB] Добавлена строка в отгрузку %d: MaterialID=%d, Boxes=%d, Amount=%d",
		shipmentID, materialID, boxes, amount)
	return nil
}

// ScanBoxForShipment сканирует коробку в отгрузке
func ScanBoxForShipment(shipmentID, huID, materialID int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// 1. Обновляем ShipmentDetails
	result, err := tx.ExecContext(ctx, `
		UPDATE ShipmentDetails 
		SET ScannedBoxes = ScannedBoxes + 1
		WHERE ShipmentID = ? AND MaterialID = ?
	`, shipmentID, materialID)
	if err != nil {
		return false, fmt.Errorf("ошибка обновления ScannedBoxes: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return false, fmt.Errorf("материал %d не найден в отгрузке %d", materialID, shipmentID)
	}

	// 2. Обновляем HU
	_, err = tx.ExecContext(ctx, `
		UPDATE HU 
		SET ShipmentID = ? 
		WHERE HUID = ?
	`, shipmentID, huID)
	if err != nil {
		return false, fmt.Errorf("ошибка обновления HU: %w", err)
	}

	// 3. Добавляем статус
	_, err = tx.ExecContext(ctx, `
		INSERT INTO HU_Status (HUID, Status, ChangedAt)
		VALUES (?, N'Подготовлена к отгрузке', GETDATE())
	`, huID)
	if err != nil {
		return false, fmt.Errorf("ошибка добавления статуса: %w", err)
	}

	// 4. Проверяем, полностью ли отсканирована отгрузка
	var totalBoxes, totalScanned int
	err = tx.QueryRowContext(ctx, `
		SELECT 
			SUM(Boxes) as TotalBoxes,
			SUM(ScannedBoxes) as TotalScanned
		FROM ShipmentDetails
		WHERE ShipmentID = ?
	`, shipmentID).Scan(&totalBoxes, &totalScanned)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки статуса отгрузки: %w", err)
	}

	isCompleted := totalScanned >= totalBoxes

	// 5. Если полностью отсканирована, обновляем статус отгрузки
	if isCompleted {
		_, err = tx.ExecContext(ctx, `
			UPDATE Shipments 
			SET Completed = 1 
			WHERE ShipmentID = ?
		`, shipmentID)
		if err != nil {
			return false, fmt.Errorf("ошибка обновления статуса отгрузки: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Отсканирована коробка HU=%d в отгрузку %d", huID, shipmentID)
	return isCompleted, nil
}

// CompleteShipment завершает отгрузку (Done = true)
func CompleteShipment(shipmentID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// 1. Обновляем статус отгрузки
	result, err := tx.ExecContext(ctx, `
		UPDATE Shipments 
		SET Done = 1, Completed = 1
		WHERE ShipmentID = ? AND Done = 0
	`, shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка завершения отгрузки: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("отгрузка %d уже завершена или не существует", shipmentID)
	}

	// 2. Обновляем статус всех коробок в этой отгрузке
	_, err = tx.ExecContext(ctx, `
		INSERT INTO HU_Status (HUID, Status, ChangedAt)
		SELECT h.HUID, N'Отгружена', GETDATE()
		FROM HU h
		WHERE h.ShipmentID = ?
	`, shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса коробок: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Завершена отгрузка ID=%d", shipmentID)
	return nil
}

// DeleteShipment удаляет отгрузку
func DeleteShipment(shipmentID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что отгрузка не завершена
	var done bool
	err := DB.QueryRowContext(ctx, `
		SELECT Done FROM Shipments WHERE ShipmentID = ?
	`, shipmentID).Scan(&done)
	if err == sql.ErrNoRows {
		return fmt.Errorf("отгрузка %d не найдена", shipmentID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки отгрузки: %w", err)
	}
	if done {
		return fmt.Errorf("нельзя удалить завершённую отгрузку %d", shipmentID)
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Удаляем детали
	_, err = tx.ExecContext(ctx, `
		DELETE FROM ShipmentDetails WHERE ShipmentID = ?
	`, shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка удаления деталей отгрузки: %w", err)
	}

	// Удаляем заголовок
	_, err = tx.ExecContext(ctx, `
		DELETE FROM Shipments WHERE ShipmentID = ?
	`, shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка удаления отгрузки: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Удалена отгрузка ID=%d", shipmentID)
	return nil
}

// GetShipmentProgress возвращает прогресс отгрузки в процентах
func GetShipmentProgress(shipmentID int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var totalBoxes, totalScanned int
	err := DB.QueryRowContext(ctx, `
		SELECT 
			SUM(Boxes) as TotalBoxes,
			SUM(ScannedBoxes) as TotalScanned
		FROM ShipmentDetails
		WHERE ShipmentID = ?
	`, shipmentID).Scan(&totalBoxes, &totalScanned)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка получения прогресса: %w", err)
	}

	if totalBoxes == 0 {
		return 0, nil
	}
	return (totalScanned * 100) / totalBoxes, nil
}
