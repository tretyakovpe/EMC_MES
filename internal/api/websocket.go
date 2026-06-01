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

	// Только отправка сообщений клиенту
	go writePump(client)
	// Чтение сообщений от клиента (для поддержания соединения)
	go readPump(client)
}

// writePump отправляет сообщения клиенту
func writePump(client *events.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// Канал закрыт
				return
			}
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Ошибка отправки WebSocket сообщения: %v", err)
				return
			}
		}
	}
}

// readPump читает сообщения от клиента (просто читаем, чтобы соединение не закрывалось)
func readPump(client *events.Client) {
	defer func() {
		client.Hub.Unregister <- client
		client.Conn.Close()
		logger.Info("WebSocket клиент отключен")
	}()

	// Устанавливаем таймаут на чтение
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Обработчик Pong (обновляет таймаут)
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
