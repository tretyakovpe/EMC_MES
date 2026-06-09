package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"EMC_MES/internal/logger"
)

// Material представляет структуру материала из БД
type Material struct {
	MaterialID   int
	MaterialCode string
	CustomerCode string
	Destination  string
	HU           string
	Netto        int
	Brutto       int
	QuantityInHU int
	Description  string
}

// GetMaterialByID возвращает материал по ID
func GetMaterialByID(materialID int) (*Material, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var m Material
	query := `
		SELECT 
			MaterialID,
			MaterialCode,
			CustomerCode,
			Destination,
			HU,
			Netto,
			Brutto,
			QuantityInHU,
			Description
		FROM [dbo].[materials]
		WHERE MaterialID = ?`

	err := DB.QueryRowContext(ctx, query, materialID).Scan(
		&m.MaterialID,
		&m.MaterialCode,
		&m.CustomerCode,
		&m.Destination,
		&m.HU,
		&m.Netto,
		&m.Brutto,
		&m.QuantityInHU,
		&m.Description,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска материала: %w", err)
	}

	return &m, nil
}

// GetMaterialByCode возвращает материал по коду
func GetMaterialByCode(materialCode string) (*Material, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var m Material
	query := `
		SELECT 
			MaterialID,
			MaterialCode,
			CustomerCode,
			Destination,
			HU,
			Netto,
			Brutto,
			QuantityInHU,
			Description
		FROM [dbo].[materials]
		WHERE MaterialCode = ?`

	err := DB.QueryRowContext(ctx, query, materialCode).Scan(
		&m.MaterialID,
		&m.MaterialCode,
		&m.CustomerCode,
		&m.Destination,
		&m.HU,
		&m.Netto,
		&m.Brutto,
		&m.QuantityInHU,
		&m.Description,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска материала: %w", err)
	}

	return &m, nil
}

// GetAllMaterials возвращает список всех материалов
func GetAllMaterials() ([]Material, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			MaterialID,
			MaterialCode,
			CustomerCode,
			Destination,
			HU,
			Netto,
			Brutto,
			QuantityInHU,
			Description
		FROM [dbo].[materials]
		ORDER BY MaterialCode`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса материалов: %w", err)
	}
	defer rows.Close()

	var materials []Material
	for rows.Next() {
		var m Material
		err := rows.Scan(
			&m.MaterialID,
			&m.MaterialCode,
			&m.CustomerCode,
			&m.Destination,
			&m.HU,
			&m.Netto,
			&m.Brutto,
			&m.QuantityInHU,
			&m.Description,
		)
		if err != nil {
			logger.Error("Ошибка сканирования материала: %v", err)
			continue
		}
		materials = append(materials, m)
	}

	return materials, nil
}

// GetMaterialQuantityInHU возвращает количество деталей в коробке для материала
func GetMaterialQuantityInHU(materialCode string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var quantity int
	query := "SELECT QuantityInHU FROM [dbo].[materials] WHERE MaterialCode = ?"
	err := DB.QueryRowContext(ctx, query, materialCode).Scan(&quantity)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка получения QuantityInHU: %w", err)
	}
	return quantity, nil
}

// GetMaterialID возвращает ID материала по коду
func GetMaterialID(materialCode string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var materialID int
	query := "SELECT MaterialID FROM [dbo].[materials] WHERE MaterialCode = ?"
	err := DB.QueryRowContext(ctx, query, materialCode).Scan(&materialID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка поиска материала: %w", err)
	}
	return materialID, nil
}

// GetMaterialsByCodePrefix возвращает материалы с определённым префиксом (для группировки)
func GetMaterialsByCodePrefix(prefix string) ([]Material, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			MaterialID,
			MaterialCode,
			CustomerCode,
			Destination,
			HU,
			Netto,
			Brutto,
			QuantityInHU,
			Description
		FROM [dbo].[materials]
		WHERE MaterialCode LIKE ?
		ORDER BY MaterialCode`

	rows, err := DB.QueryContext(ctx, query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса материалов: %w", err)
	}
	defer rows.Close()

	var materials []Material
	for rows.Next() {
		var m Material
		err := rows.Scan(
			&m.MaterialID,
			&m.MaterialCode,
			&m.CustomerCode,
			&m.Destination,
			&m.HU,
			&m.Netto,
			&m.Brutto,
			&m.QuantityInHU,
			&m.Description,
		)
		if err != nil {
			logger.Error("Ошибка сканирования материала: %v", err)
			continue
		}
		materials = append(materials, m)
	}

	return materials, nil
}

// CreateMaterial создаёт новый материал
func CreateMaterial(materialCode, customerCode, destination, hu string, netto, brutto, quantityInHU int, Description string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO [dbo].[materials] (
			MaterialCode, 
			CustomerCode, 
			Destination, 
			HU, 
			Netto, 
			Brutto, 
			QuantityInHU,
			Description
		) VALUES (?, ?, ?, ?, ?, ?, ?);
		SELECT SCOPE_IDENTITY();
	`

	var materialID int
	err := DB.QueryRowContext(
		ctx, query,
		materialCode, customerCode, destination, hu, netto, brutto, quantityInHU, Description,
	).Scan(&materialID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания материала: %w", err)
	}

	logger.Info("[DB] Создан материал: %s (ID=%d)", materialCode, materialID)
	return materialID, nil
}

// UpdateMaterial обновляет данные материала
func UpdateMaterial(materialID int, customerCode, destination, hu string, netto, brutto, quantityInHU int, description string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE [dbo].[materials] 
		SET 
			CustomerCode = ?,
			Destination = ?,
			HU = ?,
			Netto = ?,
			Brutto = ?,
			QuantityInHU = ?,
			Description = ?
		WHERE MaterialID = ?`

	result, err := DB.ExecContext(
		ctx, query,
		customerCode, destination, hu, netto, brutto, quantityInHU, materialID, description,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления материала: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("материал с ID %d не найден", materialID)
	}

	logger.Info("[DB] Обновлён материал ID=%d", materialID)
	return nil
}

// DeleteMaterial удаляет материал (если нет связей)
func DeleteMaterial(materialID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `DELETE FROM [dbo].[materials] WHERE MaterialID = ?`
	result, err := DB.ExecContext(ctx, query, materialID)
	if err != nil {
		return fmt.Errorf("ошибка удаления материала: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("материал с ID %d не найден", materialID)
	}

	logger.Info("[DB] Удалён материал ID=%d", materialID)
	return nil
}

// GetMaterialByCustomerCode возвращает материал по коду клиента
func GetMaterialByCustomerCode(customerCode string) (*Material, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var m Material
	query := `
		SELECT 
			MaterialID,
			MaterialCode,
			CustomerCode,
			Destination,
			HU,
			Netto,
			Brutto,
			QuantityInHU,
			ISNULL(Description, '') as Description
		FROM [dbo].[materials]
		WHERE RTRIM(CustomerCode) = ?`

	err := DB.QueryRowContext(ctx, query, strings.TrimSpace(customerCode)).Scan(
		&m.MaterialID,
		&m.MaterialCode,
		&m.CustomerCode,
		&m.Destination,
		&m.HU,
		&m.Netto,
		&m.Brutto,
		&m.QuantityInHU,
		&m.Description,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска материала: %w", err)
	}

	return &m, nil
}
