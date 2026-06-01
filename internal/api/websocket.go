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
		// Разрешаем все источники для разработки
		// В продакшене нужно ограничить
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

	// Регистрируем клиента
	hub.Register <- client

	logger.Info("WebSocket клиент подключен: %s", r.RemoteAddr)

	// Запускаем горутины для клиента
	go writePump(client)
	go readPump(client)
}

// writePump отправляет сообщения клиенту
func writePump(client *events.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// Канал закрыт, отправляем CloseMessage
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Ошибка отправки WebSocket сообщения: %v", err)
				return
			}

		case <-ticker.C:
			// Отправляем Ping для поддержания соединения
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	}()

	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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

	logger.Info("WebSocket клиент отключен")
}
