package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"EMC_MES/internal/api"
	"EMC_MES/internal/config"
	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"

	_ "github.com/microsoft/go-mssqldb"
)

var (
	version   = "1.0.0"
	buildTime = "development"
)

func main() {
	// Парсим флаги командной строки
	var showVersion bool
	var configPath string
	flag.BoolVar(&showVersion, "version", false, "Показать версию")
	flag.StringVar(&configPath, "config", "", "Путь к config.json (по умолчанию ./config/config.json)")
	flag.Parse()

	if showVersion {
		fmt.Printf("EMC MES Server v%s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Запуск EMC MES Server v%s", version)

	// Загружаем конфигурацию
	if err := config.LoadConfig(configPath); err != nil {
		logger.Error("Ошибка загрузки конфигурации: %v", err)
		os.Exit(1)
	}

	cfg := config.GetConfig()
	logger.Info("Конфигурация загружена. Режим отладки: %v", cfg.Debug)

	// Подключаемся к базе данных
	if err := database.Init(); err != nil {
		logger.Error("Ошибка подключения к БД: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	logger.Info("Подключение к БД установлено: %s", cfg.DbName)

	// Настраиваем и запускаем HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	router := api.SetupRoutes()

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		logger.Info("HTTP сервер запущен на http://0.0.0.0%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Получен сигнал завершения, останавливаю сервер...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при остановке сервера: %v", err)
	}

	logger.Info("Сервер остановлен. До свидания!")
}
