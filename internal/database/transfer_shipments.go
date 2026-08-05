package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// TransferShipment представляет фактическую отгрузку по заказу на перемещение
type TransferShipment struct {
	ShipmentID int
	TransferID int
	MaterialID int
	Quantity   int
	CreatedAt  time.Time
	CreatedBy  string
}

// TransferShipmentWithDetails объединяет отгрузку с информацией о материале
type TransferShipmentWithDetails struct {
	TransferShipment
	MaterialCode string
	MaterialDesc string
}

// CreateTransferShipment создаёт запись о фактической отгрузке
func CreateTransferShipment(transferID, materialID, quantity int, createdBy string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заказ существует и не завершён
	var status string
	err := DB.QueryRowContext(ctx, "SELECT Status FROM Transfers WHERE TransferID = ?", transferID).Scan(&status)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("заказ с ID %d не найден", transferID)
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if status == "Завершена" {
		return 0, fmt.Errorf("нельзя добавить отгрузку в завершённый заказ")
	}

	// Проверяем, что материал существует
	var materialExists bool
	err = DB.QueryRowContext(ctx, "SELECT 1 FROM materials WHERE MaterialID = ?", materialID).Scan(&materialExists)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("материал с ID %d не найден", materialID)
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка проверки материала: %w", err)
	}

	// Начинаем транзакцию
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Вставляем отгрузку
	query := `
		INSERT INTO TransferShipments (TransferID, MaterialID, Quantity, CreatedAt, CreatedBy)
		VALUES (?, ?, ?, GETDATE(), ?);
		SELECT SCOPE_IDENTITY();
	`

	var shipmentID int
	err = tx.QueryRowContext(ctx, query, transferID, materialID, quantity, createdBy).Scan(&shipmentID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания отгрузки: %w", err)
	}

	// Обновляем ShippedQuantity в Transfers (если поле есть)
	_, err = tx.ExecContext(ctx, `
		UPDATE Transfers 
		SET ShippedQuantity = ISNULL(ShippedQuantity, 0) + ?
		WHERE TransferID = ?
	`, quantity, transferID)
	if err != nil {
		// Не критично, если поле отсутствует
		logger.Debug("Не удалось обновить ShippedQuantity: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Создана отгрузка по заказу %d: материал %d, %d шт.", transferID, materialID, quantity)
	return shipmentID, nil
}

// GetTransferShipmentsByTransferID возвращает все отгрузки по заказу
func GetTransferShipmentsByTransferID(transferID int) ([]TransferShipmentWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			ts.ShipmentID,
			ts.TransferID,
			ts.MaterialID,
			ts.Quantity,
			ts.CreatedAt,
			ts.CreatedBy,
			m.MaterialCode,
			ISNULL(m.Description, '') as MaterialDesc
		FROM TransferShipments ts
		JOIN materials m ON ts.MaterialID = m.MaterialID
		WHERE ts.TransferID = ?
		ORDER BY ts.CreatedAt DESC
	`

	rows, err := DB.QueryContext(ctx, query, transferID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса отгрузок: %w", err)
	}
	defer rows.Close()

	var shipments []TransferShipmentWithDetails
	for rows.Next() {
		var s TransferShipmentWithDetails
		var createdBy sql.NullString
		err := rows.Scan(
			&s.ShipmentID,
			&s.TransferID,
			&s.MaterialID,
			&s.Quantity,
			&s.CreatedAt,
			&createdBy,
			&s.MaterialCode,
			&s.MaterialDesc,
		)
		if err != nil {
			logger.Error("Ошибка сканирования отгрузки: %v", err)
			continue
		}
		if createdBy.Valid {
			s.CreatedBy = createdBy.String
		}
		shipments = append(shipments, s)
	}

	return shipments, nil
}

// GetTransferShipmentsByTransferIDAndMaterial возвращает отгрузки по заказу и материалу
func GetTransferShipmentsByTransferIDAndMaterial(transferID, materialID int) ([]TransferShipmentWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			ts.ShipmentID,
			ts.TransferID,
			ts.MaterialID,
			ts.Quantity,
			ts.CreatedAt,
			ts.CreatedBy,
			m.MaterialCode,
			ISNULL(m.Description, '') as MaterialDesc
		FROM TransferShipments ts
		JOIN materials m ON ts.MaterialID = m.MaterialID
		WHERE ts.TransferID = ? AND ts.MaterialID = ?
		ORDER BY ts.CreatedAt DESC
	`

	rows, err := DB.QueryContext(ctx, query, transferID, materialID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса отгрузок: %w", err)
	}
	defer rows.Close()

	var shipments []TransferShipmentWithDetails
	for rows.Next() {
		var s TransferShipmentWithDetails
		var createdBy sql.NullString
		err := rows.Scan(
			&s.ShipmentID,
			&s.TransferID,
			&s.MaterialID,
			&s.Quantity,
			&s.CreatedAt,
			&createdBy,
			&s.MaterialCode,
			&s.MaterialDesc,
		)
		if err != nil {
			logger.Error("Ошибка сканирования отгрузки: %v", err)
			continue
		}
		if createdBy.Valid {
			s.CreatedBy = createdBy.String
		}
		shipments = append(shipments, s)
	}

	return shipments, nil
}

// GetTransferShipmentsByTransferIDGrouped возвращает отгрузки сгруппированные по материалам (для панели)
func GetTransferShipmentsByTransferIDGrouped(transferID int) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			m.MaterialCode,
			m.MaterialID,
			SUM(ts.Quantity) as TotalShipped
		FROM TransferShipments ts
		JOIN materials m ON ts.MaterialID = m.MaterialID
		WHERE ts.TransferID = ?
		GROUP BY m.MaterialCode, m.MaterialID
		ORDER BY m.MaterialCode
	`

	rows, err := DB.QueryContext(ctx, query, transferID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса группировки: %w", err)
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var materialCode string
		var materialID int
		var totalShipped int
		err := rows.Scan(&materialCode, &materialID, &totalShipped)
		if err != nil {
			logger.Error("Ошибка сканирования группировки: %v", err)
			continue
		}
		result = append(result, map[string]interface{}{
			"materialCode": materialCode,
			"materialId":   materialID,
			"totalShipped": totalShipped,
		})
	}

	return result, nil
}

// DeleteTransferShipment удаляет отгрузку (только если заказ не завершён)
func DeleteTransferShipment(shipmentID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заказ не завершён
	var transferID int
	var status string
	err := DB.QueryRowContext(ctx, `
		SELECT ts.TransferID, t.Status
		FROM TransferShipments ts
		JOIN Transfers t ON ts.TransferID = t.TransferID
		WHERE ts.ShipmentID = ?
	`, shipmentID).Scan(&transferID, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("отгрузка с ID %d не найдена", shipmentID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки отгрузки: %w", err)
	}
	if status == "Завершена" {
		return fmt.Errorf("нельзя удалить отгрузку из завершённого заказа")
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Получаем количество для обновления ShippedQuantity
	var quantity int
	err = tx.QueryRowContext(ctx, "SELECT Quantity FROM TransferShipments WHERE ShipmentID = ?", shipmentID).Scan(&quantity)
	if err != nil {
		return fmt.Errorf("ошибка получения количества: %w", err)
	}

	// Удаляем отгрузку
	_, err = tx.ExecContext(ctx, "DELETE FROM TransferShipments WHERE ShipmentID = ?", shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка удаления отгрузки: %w", err)
	}

	// Обновляем ShippedQuantity в Transfers
	_, err = tx.ExecContext(ctx, `
		UPDATE Transfers 
		SET ShippedQuantity = ISNULL(ShippedQuantity, 0) - ?
		WHERE TransferID = ?
	`, quantity, transferID)
	if err != nil {
		logger.Debug("Не удалось обновить ShippedQuantity: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Удалена отгрузка ID=%d", shipmentID)
	return nil
}
