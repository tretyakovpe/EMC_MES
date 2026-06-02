package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ShiftConfig настройки смены
type ShiftConfig struct {
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// Config основная структура конфигурации
type Config struct {
	// Подключение к БД
	DbServer   string `json:"dbServer"`
	DbPort     int    `json:"dbPort"`
	DbUser     string `json:"dbUser"`
	DbPassword string `json:"dbPassword"`
	DbName     string `json:"dbName"`

	// Сервер
	ServerPort int `json:"serverPort"`

	// Режим отладки
	Debug bool `json:"debug"`

	// Настройки смен
	Shifts map[string]ShiftConfig `json:"shifts"`
}

var (
	globalConfig *Config
	once         sync.Once
)

// LoadConfig загружает конфигурацию из config.json
func LoadConfig(configPath ...string) error {
	var err error
	once.Do(func() {
		var path string
		if len(configPath) > 0 && configPath[0] != "" {
			path = configPath[0]
		}
		globalConfig, err = load(path)
	})
	return err
}

func load(configPath string) (*Config, error) {
	if configPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("не удалось получить путь к программе: %w", err)
		}
		appDir := filepath.Dir(exePath)

		configPath = filepath.Join(appDir, "config", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = filepath.Join(appDir, "config.json")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("config.json не найден")
			}
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения config.json: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга config.json: %w", err)
	}

	// Устанавливаем смены по умолчанию, если не заданы
	if cfg.Shifts == nil {
		cfg.Shifts = map[string]ShiftConfig{
			"1": {Name: "1 смена", Start: "06:00:00", End: "13:59:59"},
			"2": {Name: "2 смена", Start: "14:00:00", End: "21:59:59"},
			"3": {Name: "3 смена", Start: "22:00:00", End: "05:59:59"},
		}
	}

	return &cfg, nil
}

// GetConfig возвращает глобальную конфигурацию
func GetConfig() *Config {
	return globalConfig
}

// GetConnectionString возвращает строку подключения к MS SQL Server
func (c *Config) GetConnectionString() string {
	if c.DbPort == 0 {
		c.DbPort = 1433
	}
	return fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=disable",
		c.DbServer, c.DbUser, c.DbPassword, c.DbPort, c.DbName)
}

// GetShiftBounds возвращает границы смены
func (c *Config) GetShiftBounds(shift string) (start, end string) {
	if s, ok := c.Shifts[shift]; ok {
		return s.Start, s.End
	}
	// Значения по умолчанию
	switch shift {
	case "1":
		return "06:30:00", "14:59:59"
	case "2":
		return "15:00:00", "23:29:59"
	default:
		return "23:30:00", "06:29:59"
	}
}
