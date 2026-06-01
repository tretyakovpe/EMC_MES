package database

import (
	"EMC_MES/internal/logger"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PlannedVolume представляет запись из таблицы PlannedVolumes
type PlannedVolume struct {
	VolumeID       int
	MaterialID     int
	MaterialCode   string
	Shift          *string // "A", "B", "C" или nil для суток
	PlannedPerHour int
	MaxPerShift    int
	IsActive       bool
	ValidFrom      time.Time
	ValidTo        *time.Time
	CreatedAt      time.Time
	CreatedBy      *string
	UpdatedAt      *time.Time
	UpdatedBy      *string
}

// Plan представляет запись из таблицы Plans
type Plan struct {
	PlanID        int
	PlanDate      time.Time
	Shift         *string // "A", "B", "C" или nil для суток
	MaterialID    int
	MaterialCode  string
	PlannedAmount int
	ActualAmount  int
	Status        string // "Активен", "Выполнен", "Отменён"
	CreatedAt     time.Time
	CreatedBy     *string
	UpdatedAt     *time.Time
	UpdatedBy     *string
}

// GetPlannedVolumes возвращает список всех активных плановых объёмов
func GetPlannedVolumes() ([]PlannedVolume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			pv.VolumeID,
			pv.MaterialID,
			m.MaterialCode,
			pv.Shift,
			pv.PlannedPerHour,
			pv.MaxPerShift,
			pv.IsActive,
			pv.ValidFrom,
			pv.ValidTo,
			pv.CreatedAt,
			pv.CreatedBy,
			pv.UpdatedAt,
			pv.UpdatedBy
		FROM PlannedVolumes pv
		JOIN materials m ON pv.MaterialID = m.MaterialID
		WHERE pv.IsActive = 1
		ORDER BY m.MaterialCode, pv.Shift
	`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса плановых объёмов: %w", err)
	}
	defer rows.Close()

	var volumes []PlannedVolume
	for rows.Next() {
		var v PlannedVolume
		var validTo sql.NullTime
		var shift sql.NullString
		var createdBy sql.NullString
		var updatedAt sql.NullTime
		var updatedBy sql.NullString

		err := rows.Scan(
			&v.VolumeID,
			&v.MaterialID,
			&v.MaterialCode,
			&shift,
			&v.PlannedPerHour,
			&v.MaxPerShift,
			&v.IsActive,
			&v.ValidFrom,
			&validTo,
			&v.CreatedAt,
			&createdBy,
			&updatedAt,
			&updatedBy,
		)
		if err != nil {
			logger.Error("Ошибка сканирования PlannedVolume: %v", err)
			continue
		}

		if shift.Valid {
			v.Shift = &shift.String
		}
		if validTo.Valid {
			v.ValidTo = &validTo.Time
		}
		if createdBy.Valid {
			v.CreatedBy = &createdBy.String
		}
		if updatedAt.Valid {
			v.UpdatedAt = &updatedAt.Time
		}
		if updatedBy.Valid {
			v.UpdatedBy = &updatedBy.String
		}

		volumes = append(volumes, v)
	}

	return volumes, nil
}

// GetPlans возвращает список планов с фильтрацией
func GetPlans(planDateFrom, planDateTo *time.Time, materialID *int, shift *string) ([]Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			p.PlanID,
			p.PlanDate,
			p.Shift,
			p.MaterialID,
			m.MaterialCode,
			p.PlannedAmount,
			p.Status,
			p.CreatedAt,
			p.CreatedBy,
			p.UpdatedAt,
			p.UpdatedBy,
			dbo.GetActualProduction(p.MaterialID, p.PlanDate, p.Shift) as ActualAmount
		FROM Plans p
		JOIN materials m ON p.MaterialID = m.MaterialID
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if planDateFrom != nil {
		query += fmt.Sprintf(" AND p.PlanDate >= @p%d", argIdx)
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), *planDateFrom))
		argIdx++
	}
	if planDateTo != nil {
		query += fmt.Sprintf(" AND p.PlanDate <= @p%d", argIdx)
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), *planDateTo))
		argIdx++
	}
	if materialID != nil {
		query += fmt.Sprintf(" AND p.MaterialID = @p%d", argIdx)
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), *materialID))
		argIdx++
	}
	if shift != nil && *shift != "" {
		if *shift == "total" {
			query += " AND p.Shift IS NULL"
		} else {
			query += fmt.Sprintf(" AND p.Shift = @p%d", argIdx)
			args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), *shift))
			argIdx++
		}
	}

	query += " ORDER BY p.PlanDate, m.MaterialCode, p.Shift"

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса планов: %w", err)
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		var shift sql.NullString
		var createdBy sql.NullString
		var updatedAt sql.NullTime
		var updatedBy sql.NullString

		err := rows.Scan(
			&p.PlanID,
			&p.PlanDate,
			&shift,
			&p.MaterialID,
			&p.MaterialCode,
			&p.PlannedAmount,
			&p.Status,
			&p.CreatedAt,
			&createdBy,
			&updatedAt,
			&updatedBy,
			&p.ActualAmount,
		)
		if err != nil {
			logger.Error("Ошибка сканирования Plan: %v", err)
			continue
		}

		if shift.Valid {
			p.Shift = &shift.String
		}
		if createdBy.Valid {
			p.CreatedBy = &createdBy.String
		}
		if updatedAt.Valid {
			p.UpdatedAt = &updatedAt.Time
		}
		if updatedBy.Valid {
			p.UpdatedBy = &updatedBy.String
		}

		plans = append(plans, p)
	}

	return plans, nil
}

// GetPlanByID возвращает план по ID
func GetPlanByID(planID int) (*Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			p.PlanID,
			p.PlanDate,
			p.Shift,
			p.MaterialID,
			m.MaterialCode,
			p.PlannedAmount,
			p.Status,
			p.CreatedAt,
			p.CreatedBy,
			p.UpdatedAt,
			p.UpdatedBy,
			dbo.GetActualProduction(p.MaterialID, p.PlanDate, p.Shift) as ActualAmount
		FROM Plans p
		JOIN materials m ON p.MaterialID = m.MaterialID
		WHERE p.PlanID = @planID
	`

	var p Plan
	var shift sql.NullString
	var createdBy sql.NullString
	var updatedAt sql.NullTime
	var updatedBy sql.NullString

	err := DB.QueryRowContext(ctx, query, sql.Named("planID", planID)).Scan(
		&p.PlanID,
		&p.PlanDate,
		&shift,
		&p.MaterialID,
		&p.MaterialCode,
		&p.PlannedAmount,
		&p.Status,
		&p.CreatedAt,
		&createdBy,
		&updatedAt,
		&updatedBy,
		&p.ActualAmount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения плана: %w", err)
	}

	if shift.Valid {
		p.Shift = &shift.String
	}
	if createdBy.Valid {
		p.CreatedBy = &createdBy.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}
	if updatedBy.Valid {
		p.UpdatedBy = &updatedBy.String
	}

	return &p, nil
}

// CreatePlan создаёт новый план
func CreatePlan(planDate time.Time, shift *string, materialID int, plannedAmount int, createdBy string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO Plans (PlanDate, Shift, MaterialID, PlannedAmount, Status, CreatedBy)
		VALUES (@planDate, @shift, @materialID, @plannedAmount, 'Активен', @createdBy);
		SELECT SCOPE_IDENTITY();
	`

	var shiftParam interface{} = nil
	if shift != nil {
		shiftParam = *shift
	}

	var planID int
	err := DB.QueryRowContext(ctx, query,
		sql.Named("planDate", planDate),
		sql.Named("shift", shiftParam),
		sql.Named("materialID", materialID),
		sql.Named("plannedAmount", plannedAmount),
		sql.Named("createdBy", createdBy),
	).Scan(&planID)

	if err != nil {
		return 0, fmt.Errorf("ошибка создания плана: %w", err)
	}

	logger.Info("[DB] Создан план: MaterialID=%d, Date=%s, Amount=%d", materialID, planDate.Format("2006-01-02"), plannedAmount)
	return int(planID), nil
}

// UpdatePlan обновляет существующий план (перезапись)
func UpdatePlan(planID int, plannedAmount int, updatedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE Plans 
		SET PlannedAmount = @plannedAmount,
		    UpdatedAt = GETDATE(),
		    UpdatedBy = @updatedBy,
		    Status = CASE 
		        WHEN dbo.GetActualProduction(MaterialID, PlanDate, Shift) >= @plannedAmount THEN 'Выполнен'
		        WHEN dbo.GetActualProduction(MaterialID, PlanDate, Shift) > 0 THEN 'В работе'
		        ELSE 'Активен'
		    END
		WHERE PlanID = @planID
	`

	result, err := DB.ExecContext(ctx, query,
		sql.Named("planID", planID),
		sql.Named("plannedAmount", plannedAmount),
		sql.Named("updatedBy", updatedBy),
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления плана: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("план с ID %d не найден", planID)
	}

	logger.Info("[DB] Обновлён план ID=%d, NewAmount=%d", planID, plannedAmount)
	return nil
}

// DeletePlan удаляет план (мягкое удаление — меняет статус на Отменён)
func DeletePlan(planID int, updatedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE Plans 
		SET Status = 'Отменён',
		    UpdatedAt = GETDATE(),
		    UpdatedBy = @updatedBy
		WHERE PlanID = @planID AND Status IN ('Активен', 'В работе')
	`

	result, err := DB.ExecContext(ctx, query,
		sql.Named("planID", planID),
		sql.Named("updatedBy", updatedBy),
	)
	if err != nil {
		return fmt.Errorf("ошибка отмены плана: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("план с ID %d не найден или уже выполнен/отменён", planID)
	}

	logger.Info("[DB] Отменён план ID=%d", planID)
	return nil
}

// GetPlannedVolumeByMaterial возвращает плановый объём для материала и смены
func GetPlannedVolumeByMaterial(materialID int, shift *string, date time.Time) (*PlannedVolume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			VolumeID,
			MaterialID,
			Shift,
			PlannedPerHour,
			MaxPerShift,
			IsActive,
			ValidFrom,
			ValidTo
		FROM PlannedVolumes
		WHERE MaterialID = @materialID
		  AND (Shift = @shift OR (Shift IS NULL AND @shift IS NULL))
		  AND IsActive = 1
		  AND ValidFrom <= @date
		  AND (ValidTo IS NULL OR ValidTo >= @date)
	`

	var shiftParam interface{} = nil
	if shift != nil {
		shiftParam = *shift
	}

	var v PlannedVolume
	var dbShift sql.NullString
	var validTo sql.NullTime

	err := DB.QueryRowContext(ctx, query,
		sql.Named("materialID", materialID),
		sql.Named("shift", shiftParam),
		sql.Named("date", date),
	).Scan(
		&v.VolumeID,
		&v.MaterialID,
		&dbShift,
		&v.PlannedPerHour,
		&v.MaxPerShift,
		&v.IsActive,
		&v.ValidFrom,
		&validTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения планового объёма: %w", err)
	}

	if dbShift.Valid {
		v.Shift = &dbShift.String
	}
	if validTo.Valid {
		v.ValidTo = &validTo.Time
	}

	return &v, nil
}

// UpdatePlansStatus обновляет статусы всех активных планов
func UpdatePlansStatus(planDate *time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var query string
	var args []interface{}

	if planDate != nil {
		query = `EXEC dbo.UpdatePlansStatus @planDate`
		args = []interface{}{sql.Named("planDate", *planDate)}
	} else {
		query = `EXEC dbo.UpdatePlansStatus @planDate = NULL`
	}

	_, err := DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления статусов планов: %w", err)
	}

	logger.Info("[DB] Обновлены статусы планов")
	return nil
}
