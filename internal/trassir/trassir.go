// internal/trassir/trassir.go
package trassir

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"EMC_MES/internal/config"
	"EMC_MES/internal/logger"
)

// Инициализируем кастомный HTTP-клиент с правильными настройками TLS
var httpClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS12, // Явно только TLS 1.2
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			},
		},
	},
}

// LoginResponse структура для парсинга SID
type LoginResponse struct {
	Sid string `json:"sid"`
}

// ExportTaskRequest структура запроса на создание задачи экспорта
type ExportTaskRequest struct {
	ResourceGuid    string `json:"resource_guid"`
	StartTs         int64  `json:"start_ts"`
	EndTs           int64  `json:"end_ts"`
	IsHardware      int    `json:"is_hardware"`
	PreferSubstream int    `json:"prefer_substream"`
}

// ExportTaskResponse структура ответа на создание задачи экспорта
type ExportTaskResponse struct {
	TaskId string `json:"task_id"`
}

// GetVideoStream возвращает io.ReadCloser с видео для указанной камеры и момента времени
func GetVideoStream(cameraGuid string, moment time.Time) (io.ReadCloser, error) {
	cfg := config.GetConfig()
	if cfg == nil || cfg.Trassir.Address == "" {
		return nil, fmt.Errorf("Trassir не настроен")
	}

	baseURL := strings.TrimSuffix(cfg.Trassir.Address, "/") + "/"

	// Шаг 1: Получаем session ID (sid)
	loginURL := fmt.Sprintf("%slogin?password=%s", baseURL, cfg.Trassir.Password)

	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса авторизации: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("авторизация вернула статус %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil || loginResp.Sid == "" {
		return nil, fmt.Errorf("не удалось распарсить session ID: %s", string(body))
	}
	sid := loginResp.Sid

	// Шаг 2: Вычисляем временные границы (минус 60 секунд, плюс 30 секунд в микросекундах Unix)
	unixMicrosecondsPerSecond := int64(1000000)
	startTs := moment.Add(-60*time.Second).Unix() * unixMicrosecondsPerSecond
	endTs := moment.Add(30*time.Second).Unix() * unixMicrosecondsPerSecond

	// Шаг 3: Создание задачи экспорта видео
	taskReq := ExportTaskRequest{
		ResourceGuid:    cameraGuid,
		StartTs:         startTs,
		EndTs:           endTs,
		IsHardware:      0,
		PreferSubstream: 0,
	}

	jsonBytes, _ := json.Marshal(taskReq)
	createTaskURL := fmt.Sprintf("%sjit-export-create-task?sid=%s", baseURL, sid)
	reqTask, err := http.NewRequest("POST", createTaskURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса задачи: %w", err)
	}
	reqTask.Header.Set("Content-Type", "application/json")
	reqTask.Header.Set("User-Agent", "Mozilla/5.0")

	respTask, err := httpClient.Do(reqTask)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания задачи экспорта: %w", err)
	}
	defer respTask.Body.Close()

	bodyTask, _ := io.ReadAll(respTask.Body)

	if respTask.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("создание задачи вернуло статус %d: %s", respTask.StatusCode, string(bodyTask))
	}

	var taskResp ExportTaskResponse
	if err := json.Unmarshal(bodyTask, &taskResp); err != nil || taskResp.TaskId == "" {
		return nil, fmt.Errorf("не удалось получить task_id: %s", string(bodyTask))
	}
	taskId := taskResp.TaskId

	// Шаг 4: Скачиваем видеофайл
	downloadURL := fmt.Sprintf("%sjit-export-download?sid=%s&task_id=%s", baseURL, sid, taskId)
	logger.Info("[VIDEO] Скачивание: %s", downloadURL)

	reqDownload, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса скачивания: %w", err)
	}
	reqDownload.Header.Set("User-Agent", "Mozilla/5.0")

	respDownload, err := httpClient.Do(reqDownload)
	if err != nil {
		return nil, fmt.Errorf("ошибка скачивания видеофайла: %w", err)
	}

	if respDownload.StatusCode != http.StatusOK {
		bodyErr, _ := io.ReadAll(respDownload.Body)
		respDownload.Body.Close()
		return nil, fmt.Errorf("скачивание вернуло статус %d: %s", respDownload.StatusCode, string(bodyErr))
	}

	// Проверяем, что тело не пустое
	// Читаем первые 256 байт для диагностики
	peekData := make([]byte, 256)
	n, err := respDownload.Body.Read(peekData)
	if err != nil && err != io.EOF {
		respDownload.Body.Close()
		return nil, fmt.Errorf("ошибка чтения видео: %w", err)
	}

	peekReader := io.MultiReader(bytes.NewReader(peekData[:n]), respDownload.Body)

	return &closableReader{
		Reader: peekReader,
		Closer: respDownload.Body,
	}, nil
}

// closableReader обёртка для io.Reader с io.Closer
type closableReader struct {
	io.Reader
	io.Closer
}
