// handleVideoStream возвращает видео для partNok записи
package api

import (
	"io"
	"net/http"
	"strings"

	"EMC_MES/internal/database"
	"EMC_MES/internal/logger"
	"EMC_MES/internal/trassir"
)

func handleVideoStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	partNokID := r.URL.Query().Get("id")
	if partNokID == "" {
		http.Error(w, "Missing partNok id", http.StatusBadRequest)
		return
	}

	// Получаем запись partNok
	part, err := database.GetPartNokByID(partNokID)
	if err != nil {
		logger.Error("Ошибка получения partNok %s: %v", partNokID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if part == nil {
		http.Error(w, "PartNok not found", http.StatusNotFound)
		return
	}

	// Получаем камеру по линии
	lineCfg, err := database.GetLineByName(part.Line)
	if err != nil || lineCfg == nil || !lineCfg.Camera.Valid {
		http.Error(w, "Камера не настроена для этой линии", http.StatusNotFound)
		return
	}
	cameraGuid := strings.TrimSpace(lineCfg.Camera.String)

	// Момент события
	moment := part.Datetime

	// Пробуем получить видео из Trassir
	reader, err := trassir.GetVideoStream(cameraGuid, moment)
	if err != nil {
		logger.Error("Ошибка получения видео для partNok %s: %v", partNokID, err)
		// Если видео не найдено — возвращаем 404 с понятным сообщением
		http.Error(w, "Видео недоступно (возможно, архив удалён)", http.StatusNotFound)
		return
	}
	defer reader.Close()

	// Отдаём поток
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if _, err := io.Copy(w, reader); err != nil {
		logger.Error("Ошибка отправки видео: %v", err)
	}
}
