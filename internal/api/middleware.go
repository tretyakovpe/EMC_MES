package api

import (
	"EMC_MES/internal/logger"
	"bytes"
	"io"
	"net/http"
	"strings"
)

// LoggingMiddleware логирует запросы с параметрами
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Логируем только API-запросы
		if strings.HasPrefix(r.URL.Path, "/api/") {

			// 1. Параметры из URL (query parameters)
			queryParams := r.URL.RawQuery
			queryInfo := ""
			if queryParams != "" {
				queryInfo = "?" + queryParams
			}

			// 2. Параметры из динамического пути (например /api/lines/123)
			// Они уже видны в r.URL.Path

			// 3. Пытаемся прочитать тело запроса (только для POST/PUT/PATCH)
			bodyInfo := ""
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil && len(bodyBytes) > 0 {
					// Ограничиваем длину лога (первые 500 символов)
					bodyStr := string(bodyBytes)
					if len(bodyStr) > 500 {
						bodyStr = bodyStr[:500] + "..."
					}
					bodyInfo = " | Body: " + bodyStr

					// Важно: восстанавливаем тело для дальнейшего использования
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			// Итоговый лог
			logger.Info("➡️ %s %s%s%s (from %s)",
				r.Method,
				r.URL.Path,
				queryInfo,
				bodyInfo,
				r.RemoteAddr,
			)
		}

		next.ServeHTTP(w, r)
	})
}
