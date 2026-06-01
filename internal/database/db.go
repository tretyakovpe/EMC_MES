package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"EMC_MES/internal/config"
	"EMC_MES/internal/events"
	"EMC_MES/internal/logger"

	_ "github.com/microsoft/go-mssqldb"
)

var DB *sql.DB
var hub *events.Hub

// Init инициирует подключение к MS SQL Server
func Init() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("конфигурация не загружена")
	}

	connString := cfg.GetConnectionString()

	var err error
	DB, err = sql.Open("mssql", connString)
	if err != nil {
		return fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	// Настройки пула соединений
	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(30 * time.Minute)

	// Проверяем физическое наличие связи с сервером БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err != nil {
		return fmt.Errorf("база данных недоступна: %w", err)
	}

	logger.Info("Успешное подключение к базе данных MS SQL Server: %s", cfg.DbName)
	return nil
}

// Close закрывает пул соединений при остановке приложения
func Close() {
	if DB != nil {
		DB.Close()
		logger.Info("Соединение с базой данных закрыто")
	}
}

// Ping проверяет соединение с БД
func Ping() error {
	if DB == nil {
		return fmt.Errorf("база данных не инициализирована")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return DB.PingContext(ctx)
}

// BeginTx начинает транзакцию с таймаутом
func BeginTx(ctx context.Context, timeout time.Duration) (*sql.Tx, context.CancelFunc, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		tx, err := DB.BeginTx(ctx, nil)
		if err != nil {
			cancel()
			return nil, nil, err
		}
		return tx, cancel, nil
	}
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	return tx, func() {}, nil
}

func SetHub(h *events.Hub) {
	hub = h
}
