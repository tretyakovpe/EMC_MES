package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

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
}

var (
	globalConfig *Config
	once         sync.Once
)

// LoadConfig загружает конфигурацию из config.json
// configPath - опциональный параметр, путь к файлу конфигурации
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
	// Если путь не указан, ищем в стандартных местах
	if configPath == "" {
		// Получаем путь к исполняемому файлу
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("не удалось получить путь к программе: %w", err)
		}
		appDir := filepath.Dir(exePath)

		// Пробуем config/config.json
		configPath = filepath.Join(appDir, "config", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Пробуем config.json в корне
			configPath = filepath.Join(appDir, "config.json")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("config.json не найден")
			}
		}
	}

	// Читаем файл
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения config.json: %w", err)
	}

	// Парсим JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга config.json: %w", err)
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
