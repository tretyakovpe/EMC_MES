package database

import (
	"EMC_MES/internal/logger"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ================================================================
// СТРУКТУРЫ ДАННЫХ
// ================================================================

// TransferOrder представляет заказ на перемещение
type TransferOrder struct {
	TransferOrderID int        `json:"transferOrderId"`
	Number          int        `json:"number"`
	Date            time.Time  `json:"date"`
	FromWarehouseID int        `json:"fromWarehouseId"`
	ToWarehouseID   int        `json:"toWarehouseId"`
	PlannedDate     time.Time  `json:"plannedDate"`
	Status          string     `json:"status"` // Draft, InProgress, Ready, Completed
	StatusChangedAt *time.Time `json:"statusChangedAt,omitempty"`
	StatusChangedBy *string    `json:"statusChangedBy,omitempty"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	StartedBy       *string    `json:"startedBy,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	CompletedBy     *string    `json:"completedBy,omitempty"`
	CreatedBy       *string    `json:"createdBy,omitempty"`
	// Дополнительные поля (для отображения)
	FromWarehouseCode string `json:"fromWarehouseCode"`
	FromWarehouseName string `json:"fromWarehouseName"`
	ToWarehouseCode   string `json:"toWarehouseCode"`
	ToWarehouseName   string `json:"toWarehouseName"`
	// Детали заказа (для полной структуры)
	Details   []TransferOrderDetail   `json:"details,omitempty"`
	Shipments []TransferOrderShipment `json:"shipments,omitempty"`
}

// TransferOrderDetail представляет позицию заказа
type TransferOrderDetail struct {
	TransferOrderDetailID int    `json:"transferOrderDetailId,omitempty"`
	TransferOrderID       int    `json:"transferOrderId,omitempty"`
	MaterialID            int    `json:"materialId"`
	Quantity              int    `json:"quantity"`
	ShippedQuantity       int    `json:"shippedQuantity"`
	MaterialCode          string `json:"materialCode"`
	Description           string `json:"description"`
}

// TransferOrderShipment представляет фактическую отгрузку
type TransferOrderShipment struct {
	TransferOrderShipmentID int       `json:"transferOrderShipmentId"`
	TransferOrderID         int       `json:"transferOrderId"`
	MaterialID              int       `json:"materialId"`
	Quantity                int       `json:"quantity"`
	CreatedAt               time.Time `json:"createdAt"`
	CreatedBy               *string   `json:"createdBy,omitempty"`
	MaterialCode            string    `json:"materialCode"`
	MaterialDescription     string    `json:"materialDescription"`
}

// TransferOrderDetailInput используется при создании/обновлении заказа
type TransferOrderDetailInput struct {
	MaterialCode string `json:"materialCode"`
	Quantity     int    `json:"quantity"`
}

// ================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ================================================================

// getTransferOrderStatusLabel возвращает человекочитаемый статус
func getTransferOrderStatusLabel(status string) string {
	labels := map[string]string{
		"Draft":      "Создан",
		"InProgress": "В работе",
		"Ready":      "Готов",
		"Completed":  "Завершен",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

// ================================================================
// ОСНОВНЫЕ ФУНКЦИИ РАБОТЫ С ЗАКАЗАМИ
// ================================================================

// CreateTransferOrder создаёт новый заказ в статусе Draft
// Номер заказа (number) вводит логист вручную
func CreateTransferOrder(
	number int,
	fromWarehouseID, toWarehouseID int,
	plannedDate time.Time,
	details []TransferOrderDetailInput,
	createdBy string,
) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Проверяем, что номер заказа уникален
	var exists bool
	err := DB.QueryRowContext(ctx, "SELECT 1 FROM TransferOrders WHERE Number = ?", number).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("ошибка проверки номера заказа: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("заказ с номером %d уже существует", number)
	}

	// Проверяем, что склады существуют и активны
	if !warehouseExists(ctx, fromWarehouseID) {
		return 0, fmt.Errorf("склад отправителя (ID=%d) не найден или неактивен", fromWarehouseID)
	}
	if !warehouseExists(ctx, toWarehouseID) {
		return 0, fmt.Errorf("склад получателя (ID=%d) не найден или неактивен", toWarehouseID)
	}
	if fromWarehouseID == toWarehouseID {
		return 0, fmt.Errorf("склад отправителя и получателя не могут совпадать")
	}

	if len(details) == 0 {
		return 0, fmt.Errorf("заказ должен содержать хотя бы одну позицию")
	}

	// Проверяем материалы и получаем их ID
	materialIDs := make([]int, 0, len(details))
	for _, d := range details {
		material, err := GetMaterialByCode(d.MaterialCode)
		if err != nil {
			return 0, fmt.Errorf("ошибка проверки материала %s: %w", d.MaterialCode, err)
		}
		if material == nil {
			return 0, fmt.Errorf("материал с кодом %s не найден", d.MaterialCode)
		}
		if d.Quantity <= 0 {
			return 0, fmt.Errorf("количество для материала %s должно быть положительным", d.MaterialCode)
		}
		materialIDs = append(materialIDs, material.MaterialID)
	}

	// Начинаем транзакцию
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Создаём заказ
	insertOrderQuery := `
		INSERT INTO TransferOrders (
			Number, Date, FromWarehouseID, ToWarehouseID, PlannedDate,
			Status, StatusChangedAt, StatusChangedBy, CreatedBy
		) VALUES (?, GETDATE(), ?, ?, ?, 'Draft', GETDATE(), ?, ?);
		SELECT SCOPE_IDENTITY();
	`

	var orderID int
	err = tx.QueryRowContext(
		ctx, insertOrderQuery,
		number, fromWarehouseID, toWarehouseID, plannedDate,
		createdBy, createdBy,
	).Scan(&orderID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания заказа: %w", err)
	}

	// Добавляем детали заказа
	insertDetailQuery := `
		INSERT INTO TransferOrderDetails (
			TransferOrderID, MaterialID, Quantity, ShippedQuantity
		) VALUES (?, ?, ?, 0);
	`

	for i, d := range details {
		material, _ := GetMaterialByCode(d.MaterialCode)
		_, err = tx.ExecContext(
			ctx, insertDetailQuery,
			orderID, material.MaterialID, d.Quantity,
		)
		if err != nil {
			return 0, fmt.Errorf("ошибка добавления позиции %d (материал %s): %w", i+1, d.MaterialCode, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Создан заказ №%d (ID=%d) на перемещение с %d позициями",
		number, orderID, len(details))
	return orderID, nil
}

// GetTransferOrders возвращает список заказов с фильтрацией
func GetTransferOrders(
	status *string,
	fromWarehouseID *int,
	toWarehouseID *int,
	fromDate *time.Time,
	toDate *time.Time,
) ([]TransferOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT 
			t.TransferOrderID,
			t.Number,
			t.Date,
			t.FromWarehouseID,
			t.ToWarehouseID,
			t.PlannedDate,
			t.Status,
			t.StatusChangedAt,
			t.StatusChangedBy,
			t.StartedAt,
			t.StartedBy,
			t.CompletedAt,
			t.CompletedBy,
			t.CreatedBy,
			wf.Code as FromWarehouseCode,
			wf.Name as FromWarehouseName,
			wt.Code as ToWarehouseCode,
			wt.Name as ToWarehouseName
		FROM TransferOrders t
		JOIN Warehouses wf ON t.FromWarehouseID = wf.WarehouseID
		JOIN Warehouses wt ON t.ToWarehouseID = wt.WarehouseID
		WHERE 1=1
	`

	var args []interface{}
	var argCounter int = 1

	if status != nil && *status != "" {
		query += fmt.Sprintf(" AND t.Status = ?")
		args = append(args, *status)
		argCounter++
	}

	if fromWarehouseID != nil {
		query += fmt.Sprintf(" AND t.FromWarehouseID = ?")
		args = append(args, *fromWarehouseID)
		argCounter++
	}

	if toWarehouseID != nil {
		query += fmt.Sprintf(" AND t.ToWarehouseID = ?")
		args = append(args, *toWarehouseID)
		argCounter++
	}

	if fromDate != nil {
		query += fmt.Sprintf(" AND t.PlannedDate >= ?")
		args = append(args, fromDate.Format("2006-01-02"))
		argCounter++
	}

	if toDate != nil {
		query += fmt.Sprintf(" AND t.PlannedDate <= ?")
		args = append(args, toDate.Format("2006-01-02"))
		argCounter++
	}

	query += " ORDER BY t.Number DESC"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса заказов: %w", err)
	}
	defer rows.Close()

	var orders []TransferOrder
	for rows.Next() {
		var o TransferOrder
		var statusChangedAt, startedAt, completedAt sql.NullTime
		var statusChangedBy, startedBy, completedBy, createdBy sql.NullString

		err := rows.Scan(
			&o.TransferOrderID,
			&o.Number,
			&o.Date,
			&o.FromWarehouseID,
			&o.ToWarehouseID,
			&o.PlannedDate,
			&o.Status,
			&statusChangedAt,
			&statusChangedBy,
			&startedAt,
			&startedBy,
			&completedAt,
			&completedBy,
			&createdBy,
			&o.FromWarehouseCode,
			&o.FromWarehouseName,
			&o.ToWarehouseCode,
			&o.ToWarehouseName,
		)
		if err != nil {
			logger.Error("Ошибка сканирования заказа: %v", err)
			continue
		}

		if statusChangedAt.Valid {
			o.StatusChangedAt = &statusChangedAt.Time
		}
		if statusChangedBy.Valid {
			o.StatusChangedBy = &statusChangedBy.String
		}
		if startedAt.Valid {
			o.StartedAt = &startedAt.Time
		}
		if startedBy.Valid {
			o.StartedBy = &startedBy.String
		}
		if completedAt.Valid {
			o.CompletedAt = &completedAt.Time
		}
		if completedBy.Valid {
			o.CompletedBy = &completedBy.String
		}
		if createdBy.Valid {
			o.CreatedBy = &createdBy.String
		}

		orders = append(orders, o)
	}

	return orders, nil
}

// GetTransferOrderByID возвращает заказ с деталями и отгрузками
func GetTransferOrderByID(orderID int) (*TransferOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Основной запрос заказа
	queryOrder := `
		SELECT 
			t.TransferOrderID,
			t.Number,
			t.Date,
			t.FromWarehouseID,
			t.ToWarehouseID,
			t.PlannedDate,
			t.Status,
			t.StatusChangedAt,
			t.StatusChangedBy,
			t.StartedAt,
			t.StartedBy,
			t.CompletedAt,
			t.CompletedBy,
			t.CreatedBy,
			wf.Code as FromWarehouseCode,
			wf.Name as FromWarehouseName,
			wt.Code as ToWarehouseCode,
			wt.Name as ToWarehouseName
		FROM TransferOrders t
		JOIN Warehouses wf ON t.FromWarehouseID = wf.WarehouseID
		JOIN Warehouses wt ON t.ToWarehouseID = wt.WarehouseID
		WHERE t.TransferOrderID = ?
	`

	var o TransferOrder
	var statusChangedAt, startedAt, completedAt sql.NullTime
	var statusChangedBy, startedBy, completedBy, createdBy sql.NullString

	err := DB.QueryRowContext(ctx, queryOrder, orderID).Scan(
		&o.TransferOrderID,
		&o.Number,
		&o.Date,
		&o.FromWarehouseID,
		&o.ToWarehouseID,
		&o.PlannedDate,
		&o.Status,
		&statusChangedAt,
		&statusChangedBy,
		&startedAt,
		&startedBy,
		&completedAt,
		&completedBy,
		&createdBy,
		&o.FromWarehouseCode,
		&o.FromWarehouseName,
		&o.ToWarehouseCode,
		&o.ToWarehouseName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заказа: %w", err)
	}

	if statusChangedAt.Valid {
		o.StatusChangedAt = &statusChangedAt.Time
	}
	if statusChangedBy.Valid {
		o.StatusChangedBy = &statusChangedBy.String
	}
	if startedAt.Valid {
		o.StartedAt = &startedAt.Time
	}
	if startedBy.Valid {
		o.StartedBy = &startedBy.String
	}
	if completedAt.Valid {
		o.CompletedAt = &completedAt.Time
	}
	if completedBy.Valid {
		o.CompletedBy = &completedBy.String
	}
	if createdBy.Valid {
		o.CreatedBy = &createdBy.String
	}

	// Загружаем детали заказа с актуальными ShippedQuantity
	queryDetails := `
		SELECT 
			d.TransferOrderDetailID,
			d.TransferOrderID,
			d.MaterialID,
			d.Quantity,
			d.ShippedQuantity,
			m.MaterialCode,
			ISNULL(m.Description, '') as Description
		FROM TransferOrderDetails d
		JOIN materials m ON d.MaterialID = m.MaterialID
		WHERE d.TransferOrderID = ?
		ORDER BY m.MaterialCode
	`

	rows, err := DB.QueryContext(ctx, queryDetails, orderID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения деталей заказа: %w", err)
	}
	defer rows.Close()

	o.Details = make([]TransferOrderDetail, 0)
	for rows.Next() {
		var d TransferOrderDetail
		err := rows.Scan(
			&d.TransferOrderDetailID,
			&d.TransferOrderID,
			&d.MaterialID,
			&d.Quantity,
			&d.ShippedQuantity,
			&d.MaterialCode,
			&d.Description,
		)
		if err != nil {
			logger.Error("Ошибка сканирования детали заказа: %v", err)
			continue
		}
		o.Details = append(o.Details, d)
	}

	// Загружаем отгрузки
	queryShipments := `
		SELECT 
			s.TransferOrderShipmentID,
			s.TransferOrderID,
			s.MaterialID,
			s.Quantity,
			s.CreatedAt,
			s.CreatedBy,
			m.MaterialCode,
			ISNULL(m.Description, '') as MaterialDescription
		FROM TransferOrderShipments s
		JOIN materials m ON s.MaterialID = m.MaterialID
		WHERE s.TransferOrderID = ?
		ORDER BY s.CreatedAt DESC
	`

	rowsShip, err := DB.QueryContext(ctx, queryShipments, orderID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения отгрузок: %w", err)
	}
	defer rowsShip.Close()

	o.Shipments = make([]TransferOrderShipment, 0)
	for rowsShip.Next() {
		var s TransferOrderShipment
		var createdBy sql.NullString
		err := rowsShip.Scan(
			&s.TransferOrderShipmentID,
			&s.TransferOrderID,
			&s.MaterialID,
			&s.Quantity,
			&s.CreatedAt,
			&createdBy,
			&s.MaterialCode,
			&s.MaterialDescription,
		)
		if err != nil {
			logger.Error("Ошибка сканирования отгрузки: %v", err)
			continue
		}
		if createdBy.Valid {
			s.CreatedBy = &createdBy.String
		}
		o.Shipments = append(o.Shipments, s)
	}

	return &o, nil
}

// UpdateTransferOrder обновляет заказ (только в статусе Draft)
func UpdateTransferOrder(
	orderID int,
	number int,
	fromWarehouseID, toWarehouseID int,
	plannedDate time.Time,
	details []TransferOrderDetailInput,
	updatedBy string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Проверяем, что заказ существует и в статусе Draft
	var currentStatus string
	var currentNumber int
	err := DB.QueryRowContext(ctx, "SELECT Status, Number FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&currentStatus, &currentNumber)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if currentStatus != "Draft" {
		return fmt.Errorf("нельзя редактировать заказ в статусе '%s'", getTransferOrderStatusLabel(currentStatus))
	}

	// Если номер изменился, проверяем уникальность
	if number != currentNumber {
		var exists bool
		err = DB.QueryRowContext(ctx, "SELECT 1 FROM TransferOrders WHERE Number = ? AND TransferOrderID != ?", number, orderID).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("ошибка проверки номера заказа: %w", err)
		}
		if exists {
			return fmt.Errorf("заказ с номером %d уже существует", number)
		}
	}

	// Проверяем склады
	if !warehouseExists(ctx, fromWarehouseID) {
		return fmt.Errorf("склад отправителя (ID=%d) не найден или неактивен", fromWarehouseID)
	}
	if !warehouseExists(ctx, toWarehouseID) {
		return fmt.Errorf("склад получателя (ID=%d) не найден или неактивен", toWarehouseID)
	}
	if fromWarehouseID == toWarehouseID {
		return fmt.Errorf("склад отправителя и получателя не могут совпадать")
	}
	if len(details) == 0 {
		return fmt.Errorf("заказ должен содержать хотя бы одну позицию")
	}

	// Проверяем материалы
	materialIDs := make([]int, 0, len(details))
	for _, d := range details {
		material, err := GetMaterialByCode(d.MaterialCode)
		if err != nil {
			return fmt.Errorf("ошибка проверки материала %s: %w", d.MaterialCode, err)
		}
		if material == nil {
			return fmt.Errorf("материал с кодом %s не найден", d.MaterialCode)
		}
		if d.Quantity <= 0 {
			return fmt.Errorf("количество для материала %s должно быть положительным", d.MaterialCode)
		}
		materialIDs = append(materialIDs, material.MaterialID)
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Обновляем заголовок заказа
	updateOrderQuery := `
		UPDATE TransferOrders
		SET 
			Number = ?,
			FromWarehouseID = ?,
			ToWarehouseID = ?,
			PlannedDate = ?,
			StatusChangedAt = GETDATE(),
			StatusChangedBy = ?
		WHERE TransferOrderID = ? AND Status = 'Draft'
	`
	result, err := tx.ExecContext(ctx, updateOrderQuery, number, fromWarehouseID, toWarehouseID, plannedDate, updatedBy, orderID)
	if err != nil {
		return fmt.Errorf("ошибка обновления заказа: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ не найден или не в статусе Draft")
	}

	// Удаляем старые детали
	_, err = tx.ExecContext(ctx, "DELETE FROM TransferOrderDetails WHERE TransferOrderID = ?", orderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления старых деталей: %w", err)
	}

	// Добавляем новые детали
	insertDetailQuery := `
		INSERT INTO TransferOrderDetails (
			TransferOrderID, MaterialID, Quantity, ShippedQuantity
		) VALUES (?, ?, ?, 0)
	`
	for _, d := range details {
		material, _ := GetMaterialByCode(d.MaterialCode)
		_, err = tx.ExecContext(ctx, insertDetailQuery, orderID, material.MaterialID, d.Quantity)
		if err != nil {
			return fmt.Errorf("ошибка добавления детали для материала %s: %w", d.MaterialCode, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("[DB] Обновлён заказ ID=%d (№%d, %d позиций)", orderID, number, len(details))
	return nil
}

// DeleteTransferOrder удаляет заказ (только в статусе Draft)
func DeleteTransferOrder(orderID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем статус
	var status string
	err := DB.QueryRowContext(ctx, "SELECT Status FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if status != "Draft" {
		return fmt.Errorf("нельзя удалить заказ в статусе '%s'", getTransferOrderStatusLabel(status))
	}

	// Удаляем (каскадно удалятся детали и отгрузки из-за ON DELETE CASCADE)
	result, err := DB.ExecContext(ctx, "DELETE FROM TransferOrders WHERE TransferOrderID = ? AND Status = 'Draft'", orderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления заказа: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ не найден или не в статусе Draft")
	}

	logger.Info("[DB] Удалён заказ ID=%d", orderID)
	return nil
}

// ================================================================
// УПРАВЛЕНИЕ СТАТУСАМИ
// ================================================================

// StartTransferOrder переводит заказ из Draft в InProgress
func StartTransferOrder(orderID int, startedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заказ существует и в статусе Draft
	var status string
	var number int
	err := DB.QueryRowContext(ctx, "SELECT Status, Number FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&status, &number)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if status != "Draft" {
		return fmt.Errorf("нельзя начать сборку заказа в статусе '%s'", getTransferOrderStatusLabel(status))
	}

	query := `
		UPDATE TransferOrders
		SET 
			Status = 'InProgress',
			StatusChangedAt = GETDATE(),
			StatusChangedBy = ?,
			StartedAt = GETDATE(),
			StartedBy = ?
		WHERE TransferOrderID = ? AND Status = 'Draft'
	`
	result, err := DB.ExecContext(ctx, query, startedBy, startedBy, orderID)
	if err != nil {
		return fmt.Errorf("ошибка начала сборки: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ не найден или уже не в статусе Draft")
	}

	logger.Info("[DB] Заказ №%d (ID=%d) переведён в статус InProgress пользователем %s", number, orderID, startedBy)
	return nil
}

// ConfirmTransferOrder переводит заказ из Ready в Completed
func ConfirmTransferOrder(orderID int, confirmedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, что заказ существует и в статусе Ready
	var status string
	var number int
	err := DB.QueryRowContext(ctx, "SELECT Status, Number FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&status, &number)
	if err == sql.ErrNoRows {
		return fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if status != "Ready" {
		return fmt.Errorf("заказ можно подтвердить только в статусе 'Готов', текущий статус: '%s'", getTransferOrderStatusLabel(status))
	}

	query := `
		UPDATE TransferOrders
		SET 
			Status = 'Completed',
			StatusChangedAt = GETDATE(),
			StatusChangedBy = ?,
			CompletedAt = GETDATE(),
			CompletedBy = ?
		WHERE TransferOrderID = ? AND Status = 'Ready'
	`
	result, err := DB.ExecContext(ctx, query, confirmedBy, confirmedBy, orderID)
	if err != nil {
		return fmt.Errorf("ошибка подтверждения заказа: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заказ не найден или уже не в статусе Ready")
	}

	logger.Info("[DB] Заказ №%d (ID=%d) подтверждён пользователем %s", number, orderID, confirmedBy)
	return nil
}

// ================================================================
// РАБОТА С ОТГРУЗКАМИ
// ================================================================

// AddTransferShipment добавляет отгрузку по заказу
func AddTransferShipment(
	orderID int,
	materialCode string,
	quantity int,
	createdBy string,
) (*TransferOrderShipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Проверяем заказ
	var status string
	var number int
	err := DB.QueryRowContext(ctx, "SELECT Status, Number FROM TransferOrders WHERE TransferOrderID = ?", orderID).Scan(&status, &number)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("заказ с ID %d не найден", orderID)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки заказа: %w", err)
	}
	if status != "InProgress" {
		return nil, fmt.Errorf("отгрузки можно добавлять только в заказ в статусе 'В работе', текущий статус: '%s'", getTransferOrderStatusLabel(status))
	}

	// 2. Проверяем материал
	material, err := GetMaterialByCode(materialCode)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки материала: %w", err)
	}
	if material == nil {
		return nil, fmt.Errorf("материал с кодом %s не найден", materialCode)
	}

	// 3. Проверяем, есть ли материал в заказе
	var detailID int
	var plannedQuantity int
	var shippedQuantity int
	err = DB.QueryRowContext(ctx, `
		SELECT TransferOrderDetailID, Quantity, ShippedQuantity
		FROM TransferOrderDetails
		WHERE TransferOrderID = ? AND MaterialID = ?
	`, orderID, material.MaterialID).Scan(&detailID, &plannedQuantity, &shippedQuantity)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("материал %s не найден в заказе", materialCode)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки деталей заказа: %w", err)
	}

	// 4. Проверяем остаток
	remaining := plannedQuantity - shippedQuantity
	if quantity > remaining {
		return nil, fmt.Errorf("нельзя отгрузить больше %d шт. (план: %d, уже отгружено: %d)",
			remaining, plannedQuantity, shippedQuantity)
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("количество должно быть положительным")
	}

	// 5. Вставляем отгрузку
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO TransferOrderShipments (
			TransferOrderID, MaterialID, Quantity, CreatedAt, CreatedBy
		) VALUES (?, ?, ?, GETDATE(), ?);
		SELECT SCOPE_IDENTITY();
	`

	var shipmentID int
	err = tx.QueryRowContext(ctx, insertQuery, orderID, material.MaterialID, quantity, createdBy).Scan(&shipmentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания отгрузки: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	// 6. Получаем созданную отгрузку с данными
	shipment, err := GetTransferShipmentByID(shipmentID)
	if err != nil {
		logger.Warn("Отгрузка создана, но не удалось получить её данные: %v", err)
	}

	logger.Info("[DB] Добавлена отгрузка %d шт. материала %s в заказ №%d (ID=%d)",
		quantity, materialCode, number, orderID)
	return shipment, nil
}

// GetTransferShipmentByID возвращает отгрузку по ID
func GetTransferShipmentByID(shipmentID int) (*TransferOrderShipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			s.TransferOrderShipmentID,
			s.TransferOrderID,
			s.MaterialID,
			s.Quantity,
			s.CreatedAt,
			s.CreatedBy,
			m.MaterialCode,
			ISNULL(m.Description, '') as MaterialDescription
		FROM TransferOrderShipments s
		JOIN materials m ON s.MaterialID = m.MaterialID
		WHERE s.TransferOrderShipmentID = ?
	`

	var s TransferOrderShipment
	var createdBy sql.NullString
	err := DB.QueryRowContext(ctx, query, shipmentID).Scan(
		&s.TransferOrderShipmentID,
		&s.TransferOrderID,
		&s.MaterialID,
		&s.Quantity,
		&s.CreatedAt,
		&createdBy,
		&s.MaterialCode,
		&s.MaterialDescription,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения отгрузки: %w", err)
	}
	if createdBy.Valid {
		s.CreatedBy = &createdBy.String
	}
	return &s, nil
}

// DeleteTransferShipment удаляет отгрузку (только если заказ в InProgress)
func DeleteTransferShipment(shipmentID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем статус заказа
	var orderID int
	var status string
	err := DB.QueryRowContext(ctx, `
		SELECT s.TransferOrderID, t.Status
		FROM TransferOrderShipments s
		JOIN TransferOrders t ON s.TransferOrderID = t.TransferOrderID
		WHERE s.TransferOrderShipmentID = ?
	`, shipmentID).Scan(&orderID, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("отгрузка с ID %d не найдена", shipmentID)
	}
	if err != nil {
		return fmt.Errorf("ошибка проверки отгрузки: %w", err)
	}
	if status != "InProgress" {
		return fmt.Errorf("нельзя удалить отгрузку из заказа в статусе '%s'", getTransferOrderStatusLabel(status))
	}

	result, err := DB.ExecContext(ctx, "DELETE FROM TransferOrderShipments WHERE TransferOrderShipmentID = ?", shipmentID)
	if err != nil {
		return fmt.Errorf("ошибка удаления отгрузки: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("отгрузка не найдена")
	}

	logger.Info("[DB] Удалена отгрузка ID=%d из заказа ID=%d", shipmentID, orderID)
	return nil
}

// GetTransferShipmentsByOrderID возвращает все отгрузки по заказу
func GetTransferShipmentsByOrderID(orderID int) ([]TransferOrderShipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			s.TransferOrderShipmentID,
			s.TransferOrderID,
			s.MaterialID,
			s.Quantity,
			s.CreatedAt,
			s.CreatedBy,
			m.MaterialCode,
			ISNULL(m.Description, '') as MaterialDescription
		FROM TransferOrderShipments s
		JOIN materials m ON s.MaterialID = m.MaterialID
		WHERE s.TransferOrderID = ?
		ORDER BY s.CreatedAt DESC
	`

	rows, err := DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса отгрузок: %w", err)
	}
	defer rows.Close()

	var shipments []TransferOrderShipment
	for rows.Next() {
		var s TransferOrderShipment
		var createdBy sql.NullString
		err := rows.Scan(
			&s.TransferOrderShipmentID,
			&s.TransferOrderID,
			&s.MaterialID,
			&s.Quantity,
			&s.CreatedAt,
			&createdBy,
			&s.MaterialCode,
			&s.MaterialDescription,
		)
		if err != nil {
			logger.Error("Ошибка сканирования отгрузки: %v", err)
			continue
		}
		if createdBy.Valid {
			s.CreatedBy = &createdBy.String
		}
		shipments = append(shipments, s)
	}

	return shipments, nil
}

// ================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ (внутренние)
// ================================================================

// warehouseExists проверяет существование и активность склада
func warehouseExists(ctx context.Context, warehouseID int) bool {
	var exists bool
	query := "SELECT 1 FROM Warehouses WHERE WarehouseID = ? AND IsActive = 1"
	err := DB.QueryRowContext(ctx, query, warehouseID).Scan(&exists)
	return err == nil
}
