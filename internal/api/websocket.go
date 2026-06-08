package api

import (
	"net/http"
	"time"

	"EMC_MES/internal/events"
	"EMC_MES/internal/logger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// serveWs обрабатывает WebSocket подключения
func serveWs(hub *events.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Ошибка WebSocket upgrade: %v", err)
		return
	}

	client := &events.Client{
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	hub.Register <- client

	logger.Info("WebSocket клиент подключен: %s", r.RemoteAddr)

	// Запускаем горутины
	go writePump(client)
	go readPump(client)
}

// writePump отправляет сообщения клиенту и пинги
func writePump(client *events.Client) {
	ticker := time.NewTicker(30 * time.Second) // пинг каждые 30 секунд
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Ошибка отправки WebSocket сообщения: %v", err)
				return
			}
		case <-ticker.C:
			// Отправляем пинг
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Ошибка отправки пинга: %v", err)
				return
			}
		}
	}
}

// readPump читает сообщения от клиента
func readPump(client *events.Client) {
	defer func() {
		client.Hub.Unregister <- client
		client.Conn.Close()
		logger.Info("WebSocket клиент отключен")
	}()

	// Устанавливаем таймаут на чтение
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Обработчик пинга от клиента (обновляет таймаут)
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket ошибка: %v", err)
			}
			break
		}
	}
}
