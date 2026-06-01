package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	fileLogger *log.Logger
	logFile    *os.File
	mu         sync.Mutex
)

// Init инициализирует логирование в файл внутри папки logs/
func Init() error {
	mu.Lock()
	defer mu.Unlock()

	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось получить путь к программе: %w", err)
	}
	appDir := filepath.Dir(exePath)

	// Создаем папку logs
	logsDir := filepath.Join(appDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать папку logs: %w", err)
	}

	// Формируем имя файла с текущей датой и временем
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("log_%s.txt", timestamp))

	// Открываем файл на запись
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл лога: %w", err)
	}

	logFile = file

	// Создаем логгер с выводом в файл и консоль (через MultiWriter)
	multiWriter := io.MultiWriter(file, os.Stdout)
	fileLogger = log.New(multiWriter, "", log.LstdFlags)

	Info("Система логирования успешно запущена. Файл: %s", logPath)
	return nil
}

// Close закрывает файл лога при выходе из программы
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
	}
}

// formatMessage форматирует сообщение с уровнем и контекстом
func formatMessage(level string, format string, v ...interface{}) string {
	msg := fmt.Sprintf(format, v...)

	// Получаем информацию о caller (файл и строка)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Берём только имя файла, без полного пути
		fileName := filepath.Base(file)
		return fmt.Sprintf("[%s] %s (%s:%d)", level, msg, fileName, line)
	}
	return fmt.Sprintf("[%s] %s", level, msg)
}

// Info пишет информационное сообщение
func Info(format string, v ...interface{}) {
	msg := formatMessage("INFO", format, v...)
	if fileLogger != nil {
		fileLogger.Println(msg)
	}
}

// Error пишет сообщение об ошибке
func Error(format string, v ...interface{}) {
	msg := formatMessage("ERROR", format, v...)
	if fileLogger != nil {
		fileLogger.Println(msg)
	}
}

// Warn пишет предупреждение
func Warn(format string, v ...interface{}) {
	msg := formatMessage("WARN", format, v...)
	if fileLogger != nil {
		fileLogger.Println(msg)
	}
}

// Debug пишет отладочное сообщение (только если включён режим отладки)
func Debug(format string, v ...interface{}) {
	// Проверяем переменную окружения DEBUG
	if os.Getenv("DEBUG") == "true" {
		msg := formatMessage("DEBUG", format, v...)
		if fileLogger != nil {
			fileLogger.Println(msg)
		}
	}
}

// Fatal пишет критическое сообщение и завершает программу
func Fatal(format string, v ...interface{}) {
	msg := formatMessage("FATAL", format, v...)
	if fileLogger != nil {
		fileLogger.Println(msg)
	}
	os.Exit(1)
}

// Printf совместимость со стандартным логгером
func Printf(format string, v ...interface{}) {
	Info(format, v...)
}

// Println совместимость со стандартным логгером
func Println(v ...interface{}) {
	Info("%s", fmt.Sprint(v...))
}
