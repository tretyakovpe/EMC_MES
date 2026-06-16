package database

import (
	"EMC_MES/internal/logger"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// LineConfig конфигурация линии из таблицы plc
type LineConfig struct {
	Name       string
	IP         string
	Port       sql.NullInt64
	Printer    sql.NullString
	PrintLabel bool
	IsOnline   bool
	LastCheck  time.Time
	IsActive   bool
	Camera     sql.NullString
}

// GetActiveLines загружает активные линии из таблицы plc
func GetActiveLines() ([]LineConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			[name],
			[ip],
			[port],
			[printer],
			[print_label],
			[is_online],
			[last_check],
			[is_active],
			[camera]
		FROM [dbo].[plc]
		WHERE [is_active] = 1
		ORDER BY [name]`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса линий: %w", err)
	}
	defer rows.Close()

	var lines []LineConfig
	for rows.Next() {
		var line LineConfig
		var lastCheck sql.NullTime

		err := rows.Scan(
			&line.Name,
			&line.IP,
			&line.Port,
			&line.Printer,
			&line.PrintLabel,
			&line.IsOnline,
			&lastCheck,
			&line.IsActive,
			&line.Camera,
		)
		if err != nil {
			logger.Error("Ошибка сканирования строки линии: %v", err)
			continue
		}

		if lastCheck.Valid {
			line.LastCheck = lastCheck.Time
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// GetAllLines загружает все линии (активные и неактивные)
func GetAllLines() ([]LineConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			[name],
			[ip],
			[port],
			[printer],
			[print_label],
			[is_online],
			[last_check],
			[is_active],
			[camera]
		FROM [dbo].[plc]
		ORDER BY [name]`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса линий: %w", err)
	}
	defer rows.Close()

	var lines []LineConfig
	for rows.Next() {
		var line LineConfig
		var lastCheck sql.NullTime

		err := rows.Scan(
			&line.Name,
			&line.IP,
			&line.Port,
			&line.Printer,
			&line.PrintLabel,
			&line.IsOnline,
			&lastCheck,
			&line.IsActive,
			&line.Camera,
		)
		if err != nil {
			logger.Error("Ошибка сканирования строки линии: %v", err)
			continue
		}

		if lastCheck.Valid {
			line.LastCheck = lastCheck.Time
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// GetLineByName возвращает конфигурацию одной линии по имени
func GetLineByName(lineName string) (*LineConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			[name],
			[ip],
			[port],
			[printer],
			[print_label],
			[is_online],
			[last_check],
			[is_active],
			[camera]
		FROM [dbo].[plc]
		WHERE [name] = ?`

	var line LineConfig
	var lastCheck sql.NullTime

	err := DB.QueryRowContext(ctx, query, lineName).Scan(
		&line.Name,
		&line.IP,
		&line.Port,
		&line.Printer,
		&line.PrintLabel,
		&line.IsOnline,
		&lastCheck,
		&line.IsActive,
		&line.Camera,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска линии: %w", err)
	}

	if lastCheck.Valid {
		line.LastCheck = lastCheck.Time
	}

	return &line, nil
}

// GetLineOnlineStatus возвращает текущий статус линии (онлайн/оффлайн)
func GetLineOnlineStatus(lineName string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var isOnline bool
	query := "SELECT [is_online] FROM [dbo].[plc] WHERE [name] = ?"
	err := DB.QueryRowContext(ctx, query, lineName).Scan(&isOnline)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("ошибка получения статуса: %w", err)
	}
	return isOnline, nil
}

// UpdateLineOnlineStatus обновляет статус онлайн/оффлайн линии
func UpdateLineOnlineStatus(lineName string, isOnline bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE [dbo].[plc] 
		SET 
			[is_online] = ?,
			[last_check] = GETDATE()
		WHERE [name] = ?`

	_, err := DB.ExecContext(ctx, query, isOnline, lineName)
	if err != nil {
		logger.Error("[%s] Ошибка обновления статуса в plc: %v", lineName, err)
		return
	}

	status := "OFFLINE"
	if isOnline {
		status = "ONLINE"
	}
	logger.Info("[%s] Статус линии в БД изменён на: %s", lineName, status)
}

// UpdateLineActiveStatus обновляет активность линии (is_active)
func UpdateLineActiveStatus(lineName string, isActive bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE [dbo].[plc] 
		SET 
			[is_active] = ?
		WHERE [name] = ?`

	result, err := DB.ExecContext(ctx, query, isActive, lineName)
	if err != nil {
		return fmt.Errorf("ошибка обновления активности линии: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("линия %s не найдена", lineName)
	}

	status := "деактивирована"
	if isActive {
		status = "активирована"
	}
	logger.Info("[DB] Линия %s %s", lineName, status)
	// Отправляем событие через WebSocket
	if hub != nil {
		hub.Broadcast("line_active_status", map[string]interface{}{
			"line":     lineName,
			"isActive": isActive,
		})
	}
	return nil
}

// CreateLine создаёт новую линию
func CreateLine(name, ip string, port int, printer string, printLabel bool, camera string, createdBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO [dbo].[plc] (
			[name], [ip], [port], [printer], [print_label], 
			[is_online], [is_active], [camera], [last_check]
		) VALUES (?, ?, ?, ?, ?, 0, 1, ?, GETDATE())`

	_, err := DB.ExecContext(ctx, query, name, ip, port, printer, printLabel, camera)
	if err != nil {
		return fmt.Errorf("ошибка создания линии: %w", err)
	}

	logger.Info("[DB] Создана линия: %s (IP: %s, создал: %s)", name, ip, createdBy)
	return nil
}

// UpdateLine обновляет конфигурацию линии
func UpdateLine(name, ip string, port int, printer string, printLabel bool, camera string, updatedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE [dbo].[plc] 
		SET 
			[ip] = ?,
			[port] = ?,
			[printer] = ?,
			[print_label] = ?,
			[camera] = ?,
			[last_check] = GETDATE()
		WHERE [name] = ?`

	result, err := DB.ExecContext(ctx, query, ip, port, printer, printLabel, camera, name)
	if err != nil {
		return fmt.Errorf("ошибка обновления линии: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("линия %s не найдена", name)
	}

	logger.Info("[DB] Обновлена линия: %s (IP: %s, обновил: %s)", name, ip, updatedBy)
	return nil
}

// DeleteLine удаляет линию (мягкое удаление - деактивирует)
func DeleteLine(lineName string, updatedBy string) error {
	return UpdateLineActiveStatus(lineName, false)
}

// GetLinesStatusForAPI возвращает статус линий для API
func GetLinesStatusForAPI() ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			[name],
			[is_online],
			[last_check],
			[is_active],
			[ip],
			[printer]
		FROM [dbo].[plc]
		ORDER BY [name]`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса линий: %w", err)
	}
	defer rows.Close()

	var lines []map[string]interface{}
	for rows.Next() {
		var name, ip string
		var printer sql.NullString
		var isOnline, isActive bool
		var lastCheck sql.NullTime

		if err := rows.Scan(&name, &isOnline, &lastCheck, &isActive, &ip, &printer); err != nil {
			logger.Error("Ошибка сканирования строки линии: %v", err)
			continue
		}

		line := map[string]interface{}{
			"name":     strings.TrimSpace(name),
			"isOnline": isOnline,
			"isActive": isActive,
			"ip":       strings.TrimSpace(ip),
			"printer":  nil,
			"lastSeen": nil,
		}

		if printer.Valid {
			line["printer"] = strings.TrimSpace(printer.String)
		}

		if lastCheck.Valid {
			line["lastSeen"] = lastCheck.Time.Format("2006-01-02 15:04:05")
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// GetLineStats возвращает текущую статистику линии
func GetLineStats(lineName string) (counter, boxQuantity int, material string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        SELECT 
            ISNULL(last_counter, 0) as last_counter,
            ISNULL(last_box_quantity, 0) as last_box_quantity,
            ISNULL(RTRIM(last_material), '-') as last_material
        FROM [dbo].[plc]
        WHERE [name] = ?`

	err = DB.QueryRowContext(ctx, query, lineName).Scan(&counter, &boxQuantity, &material)
	if err == sql.ErrNoRows {
		return 0, 0, "-", nil
	}
	return
}
