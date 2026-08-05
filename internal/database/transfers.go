package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// TransferOrder представляет заказ на перемещение
type TransferOrder struct {
	TransferOrderID int
	Number          int
	Date            time.Time
	PlannedDate     time.Time
	FromWarehouseID int
	ToWarehouseID   int
	Completed       bool
}

// TransferOrderDetail представляет деталь заказа на перемещение
type TransferOrderDetail struct {
	TransferOrderDetailID int
	TransferOrderID       int
	MaterialID            int
	Quantity              int
}

// TransferOrderWithDetails объединяет заказ с информацией о складах и материалах
type TransferOrderWithDetails struct {
	TransferOrder
	FromWarehouseCode string
	FromWarehouseName string
	ToWarehouseCode   string
	ToWarehouseName   string
	Details           []TransferOrderDetailWithMaterial
}

// TransferOrderDetailWithMaterial деталь с информацией о материале
type TransferOrderDetailWithMaterial struct {
	TransferOrderDetail
	MaterialCode string
	Description  string
}

// GetTransferOrders возвращает список заказов на перемещение
func GetTransferOrders(completed *bool, fromDate, toDate *time.Time) ([]TransferOrderWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			tor.TransferOrderID,
			tor.Number,
			tor.Date,
			tor.PlannedDate,
			tor.FromWarehouseID,
			fw.Code as FromWarehouseCode,
			fw.Name as FromWarehouseName,
			tor.ToWarehouseID,
			tw.Code as ToWarehouseCode,
			tw.Name as ToWarehouseName,
			tor.Completed
		FROM TransferOrders tor
		JOIN Warehouses fw ON tor.FromWarehouseID = fw.WarehouseID
		JOIN Warehouses tw ON tor.ToWarehouseID = tw.WarehouseID
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if completed != nil {
		query += fmt.Sprintf(" AND tor.Completed = ?")
		args = append(args, *completed)
		argIdx++
	}
	if fromDate != nil {
		query += fmt.Sprintf(" AND tor.PlannedDate >= ?")
		args = append(args, *fromDate)
		argIdx++
	}
	if toDate != nil {
		query += fmt.Sprintf(" AND tor.PlannedDate <= ?")
		args = append(args, *toDate)
		argIdx++
	}

	query += " ORDER BY tor.PlannedDate ASC, tor.Date DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса заказов: %w %s", err, query)
	}
	defer rows.Close()

	var orders []TransferOrderWithDetails
	for rows.Next() {
		var order TransferOrderWithDetails
		err := rows.Scan(
			&order.TransferOrderID,
			&order.Number,
			&order.Date,
			&order.PlannedDate,
			&order.FromWarehouseID,
			&order.FromWarehouseCode,
			&order.FromWarehouseName,
			&order.ToWarehouseID,
			&order.ToWarehouseCode,
			&order.ToWarehouseName,
			&order.Completed,
		)
		if err != nil {
			logger.Error("Ошибка сканирования заказа: %v", err)
			continue
		}

		// Загружаем детали для каждого заказа
		details, err := GetTransferOrderDetails(order.TransferOrderID)
		if err != nil {
			logger.Error("Ошибка загрузки деталей для заказа %d: %v", order.TransferOrderID, err)
			continue
		}
		order.Details = details
		orders = append(orders, order)
	}

	return orders, nil
}

// GetTransferOrderDetails возвращает детали заказа
func GetTransferOrderDetails(orderID int) ([]TransferOrderDetailWithMaterial, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			tod.TransferOrderDetailID,
			tod.TransferOrderID,
			tod.MaterialID,
			tod.Quantity,
			m.MaterialCode,
			m.Description
		FROM TransferOrderDetails tod
		JOIN materials m ON tod.MaterialID = m.MaterialID
		WHERE tod.TransferOrderID = ?
	`

	rows, err := DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса деталей: %w", err)
	}
	defer rows.Close()

	var details []TransferOrderDetailWithMaterial
	for rows.Next() {
		var d TransferOrderDetailWithMaterial
		var desc sql.NullString
		err := rows.Scan(
			&d.TransferOrderDetailID,
			&d.TransferOrderID,
			&d.MaterialID,
			&d.Quantity,
			&d.MaterialCode,
			&desc,
		)
		if err != nil {
			logger.Error("Ошибка сканирования детали: %v", err)
			continue
		}
		if desc.Valid {
			d.Description = desc.String
		}
		details = append(details, d)
	}

	return details, nil
}

// GetTransferOrderByID возвращает заказ по ID
func GetTransferOrderByID(id int) (*TransferOrderWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			tor.TransferOrderID,
			tor.Number,
			tor.Date,
			tor.PlannedDate,
			tor.FromWarehouseID,
			fw.Code as FromWarehouseCode,
			fw.Name as FromWarehouseName,
			tor.ToWarehouseID,
			tw.Code as ToWarehouseCode,
			tw.Name as ToWarehouseName,
			tor.Completed
		FROM TransferOrders tor
		JOIN Warehouses fw ON tor.FromWarehouseID = fw.WarehouseID
		JOIN Warehouses tw ON tor.ToWarehouseID = tw.WarehouseID
		WHERE tor.TransferOrderID = ?
	`

	var order TransferOrderWithDetails
	err := DB.QueryRowContext(ctx, query, id).Scan(
		&order.TransferOrderID,
		&order.Number,
		&order.Date,
		&order.PlannedDate,
		&order.FromWarehouseID,
		&order.FromWarehouseCode,
		&order.FromWarehouseName,
		&order.ToWarehouseID,
		&order.ToWarehouseCode,
		&order.ToWarehouseName,
		&order.Completed,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заказа: %w", err)
	}

	// Загружаем детали
	details, err := GetTransferOrderDetails(id)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки деталей: %w", err)
	}
	order.Details = details

	return &order, nil
}

// CreateTransferOrder создаёт новый заказ на перемещение
func CreateTransferOrder(number int, fromWarehouseID, toWarehouseID int, plannedDate time.Time, details []TransferOrderDetail) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Начинаем транзакцию
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Вставляем заказ
	query := `
		INSERT INTO TransferOrders (
			Number,
			Date,
			PlannedDate,
			FromWarehouseID,
			ToWarehouseID,
			Completed
		) VALUES (?, GETDATE(), ?, ?, ?, 0);
		SELECT SCOPE_IDENTITY();
	`

	var orderID int
	err = tx.QueryRowContext(ctx, query, number, plannedDate, fromWarehouseID, toWarehouseID).Scan(&orderID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания заказа: %w", err)
	}

	// Вставляем детали
	for _, detail := range details {
		detail.TransferOrderID = orderID
		query := `
			INSERT INTO TransferOrderDetails (
				TransferOrderID,
				MaterialID,
				Quantity
			) VALUES (?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, query, detail.TransferOrderID, detail.MaterialID, detail.Quantity)
		if err != nil {
			return 0, fmt.Errorf("ошибка создания детали заказа: %w", err)
		}
	}

	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Создан заказ на перемещение №%d (ID=%d)", number, orderID)
	return int(orderID), nil
}

// UpdateTransferOrder обновляет заказ на перемещение
func UpdateTransferOrder(orderID int, number int, fromWarehouseID, toWarehouseID int, plannedDate time.Time, details []TransferOrderDetail) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Проверяем, что заказ не завершен
	var completed bool
	err := DB.QueryRowContext(ctx, "SELECT Completed FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&completed)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки статуса: %w", err)
	}
	if completed {
		return fmt.Errorf("нельзя редактировать завершённый заказ")
	}

	// Начинаем транзакцию
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Обновляем заказ
	query := `
		UPDATE TransferOrders
		SET Number = ?,
		    FromWarehouseID = ?,
		    ToWarehouseID = ?,
		    PlannedDate = ?
		WHERE TransferOrderID = ?
	`
	_, err = tx.ExecContext(ctx, query, number, fromWarehouseID, toWarehouseID, plannedDate, orderID)
	if err != nil {
		return fmt.Errorf("ошибка обновления заказа: %w", err)
	}

	// Удаляем старые детали
	_, err = tx.ExecContext(ctx, "DELETE FROM TransferOrderDetails WHERE TransferOrderID = ?", orderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления старых деталей: %w", err)
	}

	// Вставляем новые детали
	for _, detail := range details {
		detail.TransferOrderID = orderID
		query := `
			INSERT INTO TransferOrderDetails (
				TransferOrderID,
				MaterialID,
				Quantity
			) VALUES (?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, query, detail.TransferOrderID, detail.MaterialID, detail.Quantity)
		if err != nil {
			return fmt.Errorf("ошибка создания детали заказа: %w", err)
		}
	}

	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Обновлён заказ на перемещение ID=%d", orderID)
	return nil
}

// CompleteTransferOrder завершает заказ
func CompleteTransferOrder(orderID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE TransferOrders
		SET Completed = 1
		WHERE TransferOrderID = ?
	`

	result, err := DB.ExecContext(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("ошибка завершения заказа: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}

	logger.Info("[DB] Завершён заказ ID=%d", orderID)
	return nil
}

// DeleteTransferOrder удаляет заказ (только если не завершён)
func DeleteTransferOrder(orderID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заказ не завершен
	var completed bool
	err := DB.QueryRowContext(ctx, "SELECT Completed FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&completed)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки статуса: %w", err)
	}
	if completed {
		return fmt.Errorf("нельзя удалить завершённый заказ")
	}

	// Удаляем заказ (детали удалятся каскадно)
	query := `DELETE FROM TransferOrders WHERE TransferOrderID = ?`
	result, err := DB.ExecContext(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления заказа: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}

	logger.Info("[DB] Удалён заказ ID=%d", orderID)
	return nil
}

// GenerateTransferOrderNumber генерирует номер заказа
func GenerateTransferOrderNumber() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var maxNumber int
	err := DB.QueryRowContext(ctx, "SELECT ISNULL(MAX(Number), 0) FROM TransferOrders").Scan(&maxNumber)
	if err != nil {
		return 0, fmt.Errorf("ошибка генерации номера: %w", err)
	}

	return maxNumber + 1, nil
}
