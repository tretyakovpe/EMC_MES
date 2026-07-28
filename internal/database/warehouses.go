package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/logger"
)

// Warehouse представляет склад
type Warehouse struct {
	WarehouseID int
	Code        string
	Name        string
	Description *string
	IsActive    bool
}

// GetWarehouses возвращает список всех складов
func GetWarehouses(activeOnly bool) ([]Warehouse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			WarehouseID,
			Code,
			Name,
			ISNULL(Description, '') as Description,
			IsActive
		FROM Warehouses
	`
	if activeOnly {
		query += " WHERE IsActive = 1"
	}
	query += " ORDER BY Code"

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса складов: %w", err)
	}
	defer rows.Close()

	var warehouses []Warehouse
	for rows.Next() {
		var w Warehouse
		var description string
		err := rows.Scan(&w.WarehouseID, &w.Code, &w.Name, &description, &w.IsActive)
		if err != nil {
			logger.Error("Ошибка сканирования склада: %v", err)
			continue
		}
		if description != "" {
			w.Description = &description
		}
		warehouses = append(warehouses, w)
	}

	return warehouses, nil
}

// GetWarehouseByID возвращает склад по ID
func GetWarehouseByID(id int) (*Warehouse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			WarehouseID,
			Code,
			Name,
			ISNULL(Description, '') as Description,
			IsActive
		FROM Warehouses
		WHERE WarehouseID = ?
	`

	var w Warehouse
	var description string
	err := DB.QueryRowContext(ctx, query, id).Scan(
		&w.WarehouseID, &w.Code, &w.Name, &description, &w.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения склада: %w", err)
	}
	if description != "" {
		w.Description = &description
	}
	return &w, nil
}

// CreateWarehouse создаёт новый склад
func CreateWarehouse(code, name string, description *string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var desc interface{} = nil
	if description != nil && *description != "" {
		desc = *description
	}

	query := `
		INSERT INTO Warehouses (Code, Name, Description, IsActive)
		VALUES (?, ?, ?, 1);
		SELECT SCOPE_IDENTITY();
	`

	var warehouseID int
	err := DB.QueryRowContext(ctx, query, code, name, desc).Scan(&warehouseID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания склада: %w", err)
	}

	logger.Info("[DB] Создан склад: %s (%s)", code, name)
	return int(warehouseID), nil
}

// UpdateWarehouse обновляет данные склада
func UpdateWarehouse(id int, code, name string, description *string, isActive bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var desc interface{} = nil
	if description != nil && *description != "" {
		desc = *description
	}

	query := `
		UPDATE Warehouses
		SET Code = ?, Name = ?, Description = ?, IsActive = ?
		WHERE WarehouseID = ?
	`

	result, err := DB.ExecContext(ctx, query, code, name, desc, isActive, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления склада: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("склад с ID %d не найден", id)
	}

	logger.Info("[DB] Обновлён склад ID=%d", id)
	return nil
}

// DeleteWarehouse удаляет склад (мягкое удаление)
func DeleteWarehouse(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `UPDATE Warehouses SET IsActive = 0 WHERE WarehouseID = ?`
	result, err := DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления склада: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("склад с ID %d не найден", id)
	}

	logger.Info("[DB] Удалён склад ID=%d", id)
	return nil
}
