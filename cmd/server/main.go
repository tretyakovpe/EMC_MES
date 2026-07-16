package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"EMC_MES/internal/api"
	"EMC_MES/internal/config"
	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"

	"github.com/kardianos/service"
	_ "github.com/microsoft/go-mssqldb"
)

var (
	version   = "1.0.0"
	buildTime = "development"
)

// program структура для реализации service.Handler
type program struct {
	configPath string
	server     *http.Server
}

// initialize инициализирует конфигурацию, БД и создаёт HTTP сервер
// Используется обоими режимами (Service и Interactive)
func (p *program) initialize() error {
	logger.Info("Инициализация EMC MES v%s", version)

	// Загружаем конфигурацию
	if err := config.LoadConfig(p.configPath); err != nil {
		logger.Error("Ошибка загрузки конфигурации: %v", err)
		return err
	}

	cfg := config.GetConfig()
	logger.Info("Конфигурация загружена. Режим отладки: %v", cfg.Debug)

	// Подключаемся к базе данных
	if err := database.Init(); err != nil {
		logger.Error("Ошибка подключения к БД: %v", err)
		return err
	}

	logger.Info("Подключение к БД установлено: %s", cfg.DbName)

	// Настраиваем HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	mux := api.SetupRoutes()
	router := api.LoggingMiddleware(mux)

	p.server = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return nil
}

// Start запускает программу как Windows Service
// Вызывается service.Run() когда приложение работает как служба
func (p *program) Start(s service.Service) error {
	logger.Info("Запуск EMC MES в режиме Service")

	if err := p.initialize(); err != nil {
		return err
	}

	// Запускаем сервер в горутине
	go func() {
		logger.Info("HTTP сервер запущен на http://0.0.0.0%s", p.server.Addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Ошибка запуска сервера: %v", err)
		}
	}()

	return nil
}

// Stop останавливает программу (вызывается Windows Service Manager)
func (p *program) Stop(s service.Service) error {
	logger.Info("Получен сигнал завершения, останавливаю сервер...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при остановке сервера: %v", err)
	}

	database.Close()
	logger.Info("Сервер остановлен. До свидания!")

	return nil
}

// runInteractive запускает приложение в интерактивном режиме (для отладки)
// Ждёт сигнала OS для корректной остановки
func (p *program) runInteractive() {
	logger.Info("Запуск EMC MES в интерактивном режиме")

	if err := p.initialize(); err != nil {
		logger.Error("Ошибка инициализации: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	// Запускаем сервер в горутине
	go func() {
		logger.Info("HTTP сервер запущен на http://0.0.0.0%s", p.server.Addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Получен сигнал завершения, останавливаю сервер...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при остановке сервера: %v", err)
	}

	logger.Info("Сервер остановлен. До свидания!")
}

func main() {
	// Парсим флаги командной строки
	var showVersion bool
	var configPath string
	var installService bool
	var uninstallService bool

	// Устанавливаем рабочую директорию в папку с исполняемым файлом
	exePath, err := os.Executable()
	if err == nil {
		os.Chdir(filepath.Dir(exePath))
	}

	flag.BoolVar(&showVersion, "version", false, "Показать версию")
	flag.StringVar(&configPath, "config", "", "Путь к config.json (по умолчанию ./config/config.json)")
	flag.BoolVar(&installService, "install", false, "Установить как Windows службу")
	flag.BoolVar(&uninstallService, "uninstall", false, "Удалить Windows службу")
	flag.Parse()

	// Показываем версию и выходим
	if showVersion {
		fmt.Printf("EMC MES v%s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Создаём программу с переданным путём конфи��а
	prg := &program{configPath: configPath}

	// Настройки для Windows Service
	svcConfig := &service.Config{
		Name:        "EMC_MES",
		DisplayName: "EMC MES",
		Description: "Веб-сервер для MES системы (планы, отгрузки, статистика)",
		Arguments:   []string{"-config", configPath},
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Error("Ошибка создания службы: %v", err)
		os.Exit(1)
	}

	// Обработка установки службы
	if installService {
		err = svc.Install()
		if err != nil {
			logger.Error("Ошибка установки службы: %v", err)
			os.Exit(1)
		}
		logger.Info("Служба EMC_MES успешно установлена")
		logger.Info("Запустите её командой: sc start EMC_MES")
		return
	}

	// Обработка удаления службы
	if uninstallService {
		err = svc.Uninstall()
		if err != nil {
			logger.Error("Ошибка удаления службы: %v", err)
			os.Exit(1)
		}
		logger.Info("Служба EMC_MES успешно удалена")
		return
	}

	// ✅ ЕДИНЫЙ ПУТЬ ВЫПОЛНЕНИЯ ДЛЯ ОБОИХ РЕЖИМОВ

	// Проверяем, запущено ли приложение как служба или в консоли
	if service.Interactive() {
		// Интерактивный режим (двойной клик, запуск из IDE, консоль)
		prg.runInteractive()
	} else {
		// Режим Windows Service (управляется Services.msc)
		err = svc.Run()
		if err != nil {
			logger.Error("Ошибка запуска службы: %v", err)
			os.Exit(1)
		}
	}
}
