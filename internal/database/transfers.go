package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// Transfer представляет заявку на перемещение
type Transfer struct {
	TransferID      int
	TransferNumber  string
	FromWarehouseID int
	ToWarehouseID   int
	MaterialID      int
	Quantity        int
	Status          string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

// TransferWithDetails объединяет заявку с информацией о складах и материале
type TransferWithDetails struct {
	Transfer
	FromWarehouseCode string
	FromWarehouseName string
	ToWarehouseCode   string
	ToWarehouseName   string
	MaterialCode      string
	MaterialDesc      string
}

// GetTransfers возвращает список заявок с фильтрацией
func GetTransfers(status string, fromDate, toDate *time.Time) ([]TransferWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			t.TransferID,
			t.TransferNumber,
			t.FromWarehouseID,
			fw.Code as FromWarehouseCode,
			fw.Name as FromWarehouseName,
			t.ToWarehouseID,
			tw.Code as ToWarehouseCode,
			tw.Name as ToWarehouseName,
			t.MaterialID,
			m.MaterialCode,
			m.Description as MaterialDesc,
			t.Quantity,
			t.Status,
			t.CreatedAt,
			t.CompletedAt
		FROM Transfers t
		JOIN Warehouses fw ON t.FromWarehouseID = fw.WarehouseID
		JOIN Warehouses tw ON t.ToWarehouseID = tw.WarehouseID
		JOIN materials m ON t.MaterialID = m.MaterialID
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		query += fmt.Sprintf(" AND t.Status = ?")
		args = append(args, status)
		argIdx++
	}
	if fromDate != nil {
		query += fmt.Sprintf(" AND t.CreatedAt >= ?")
		args = append(args, *fromDate)
		argIdx++
	}
	if toDate != nil {
		query += fmt.Sprintf(" AND t.CreatedAt <= ?")
		args = append(args, *toDate)
		argIdx++
	}

	query += " ORDER BY t.CreatedAt DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса заявок: %w", err)
	}
	defer rows.Close()

	var transfers []TransferWithDetails
	for rows.Next() {
		var t TransferWithDetails
		var materialDesc sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&t.TransferID,
			&t.TransferNumber,
			&t.FromWarehouseID,
			&t.FromWarehouseCode,
			&t.FromWarehouseName,
			&t.ToWarehouseID,
			&t.ToWarehouseCode,
			&t.ToWarehouseName,
			&t.MaterialID,
			&t.MaterialCode,
			&materialDesc,
			&t.Quantity,
			&t.Status,
			&t.CreatedAt,
			&completedAt,
		)
		if err != nil {
			logger.Error("Ошибка сканирования заявки: %v", err)
			continue
		}
		if materialDesc.Valid {
			t.MaterialDesc = materialDesc.String
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		transfers = append(transfers, t)
	}

	return transfers, nil
}

// GetTransferByID возвращает заявку по ID
func GetTransferByID(id int) (*TransferWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			t.TransferID,
			t.TransferNumber,
			t.FromWarehouseID,
			fw.Code as FromWarehouseCode,
			fw.Name as FromWarehouseName,
			t.ToWarehouseID,
			tw.Code as ToWarehouseCode,
			tw.Name as ToWarehouseName,
			t.MaterialID,
			m.MaterialCode,
			m.Description as MaterialDesc,
			t.Quantity,
			t.Status,
			t.CreatedAt,
			t.CompletedAt
		FROM Transfers t
		JOIN Warehouses fw ON t.FromWarehouseID = fw.WarehouseID
		JOIN Warehouses tw ON t.ToWarehouseID = tw.WarehouseID
		JOIN materials m ON t.MaterialID = m.MaterialID
		WHERE t.TransferID = ?
	`

	var t TransferWithDetails
	var materialDesc sql.NullString
	var completedAt sql.NullTime

	err := DB.QueryRowContext(ctx, query, id).Scan(
		&t.TransferID,
		&t.TransferNumber,
		&t.FromWarehouseID,
		&t.FromWarehouseCode,
		&t.FromWarehouseName,
		&t.ToWarehouseID,
		&t.ToWarehouseCode,
		&t.ToWarehouseName,
		&t.MaterialID,
		&t.MaterialCode,
		&materialDesc,
		&t.Quantity,
		&t.Status,
		&t.CreatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заявки: %w", err)
	}
	if materialDesc.Valid {
		t.MaterialDesc = materialDesc.String
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}

	return &t, nil
}

// CreateTransfer создаёт новую заявку на перемещение
func CreateTransfer(transferNumber string, fromWarehouseID, toWarehouseID, materialID, quantity int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO Transfers (
			TransferNumber,
			FromWarehouseID,
			ToWarehouseID,
			MaterialID,
			Quantity,
			Status,
			CreatedAt
		) VALUES (?, ?, ?, ?, ?, 'Создана', GETDATE());
		SELECT SCOPE_IDENTITY();
	`

	var transferID int
	err := DB.QueryRowContext(
		ctx, query,
		transferNumber,
		fromWarehouseID,
		toWarehouseID,
		materialID,
		quantity,
	).Scan(&transferID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания заявки: %w", err)
	}

	logger.Info("[DB] Создана заявка %s (ID=%d)", transferNumber, transferID)
	return int(transferID), nil
}

// UpdateTransferStatus обновляет статус заявки
func UpdateTransferStatus(transferID int, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем допустимые статусы
	validStatuses := map[string]bool{
		"Создана":   true,
		"В работе":  true,
		"Готова":    true,
		"Завершена": true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("недопустимый статус: %s", status)
	}

	var completedAt interface{} = nil
	if status == "Завершена" {
		completedAt = time.Now()
	}

	query := `
		UPDATE Transfers
		SET Status = ?,
		    CompletedAt = ?
		WHERE TransferID = ?
	`

	result, err := DB.ExecContext(ctx, query, status, completedAt, transferID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса заявки: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заявка с ID %d не найдена", transferID)
	}

	logger.Info("[DB] Обновлён статус заявки ID=%d -> %s", transferID, status)
	return nil
}

// DeleteTransfer удаляет заявку (только если статус "Создана")
func DeleteTransfer(transferID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заявка в статусе "Создана"
	var status string
	err := DB.QueryRowContext(ctx, "SELECT Status FROM Transfers WHERE TransferID = ?", transferID).Scan(&status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заявка с ID %d не найдена", transferID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки статуса: %w", err)
	}
	if status != "Создана" {
		return fmt.Errorf("нельзя удалить заявку в статусе '%s'", status)
	}

	query := `DELETE FROM Transfers WHERE TransferID = ?`
	result, err := DB.ExecContext(ctx, query, transferID)
	if err != nil {
		return fmt.Errorf("ошибка удаления заявки: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заявка с ID %d не найдена", transferID)
	}

	logger.Info("[DB] Удалена заявка ID=%d", transferID)
	return nil
}

// GenerateTransferNumber генерирует номер заявки
func GenerateTransferNumber() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	datePrefix := now.Format("2006-01-02")

	var count int
	err := DB.QueryRowContext(ctx, `
		SELECT COUNT(*) + 1 
		FROM Transfers 
		WHERE CAST(CreatedAt AS DATE) = CAST(GETDATE() AS DATE)
	`).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации номера: %w", err)
	}

	return fmt.Sprintf("TR-%s-%03d", datePrefix, count), nil
}
