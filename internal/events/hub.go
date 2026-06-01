package events

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage структура сообщения
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Ts   int64       `json:"ts"`
}

// Client представляет одно WebSocket-подключение
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte
}

// Hub управляет всеми WebSocket-клиентами
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

// NewHub создаёт новый Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run запускает основной цикл Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast отправляет сообщение всем клиентам
func (h *Hub) Broadcast(msgType string, data interface{}) {
	msg := WebSocketMessage{
		Type: msgType,
		Data: data,
		Ts:   time.Now().Unix(),
	}
	jsonMsg, _ := json.Marshal(msg)
	h.broadcast <- jsonMsg
}

// BroadcastBoxClosed отправляет событие о закрытии коробки
func (h *Hub) BroadcastBoxClosed(line, material, labelCode string, amount int) {
	h.Broadcast("box_closed", map[string]interface{}{
		"line":      line,
		"material":  material,
		"labelCode": labelCode,
		"amount":    amount,
	})
}

// BroadcastPartProduced отправляет событие о производстве детали
func (h *Hub) BroadcastPartProduced(line, material string, counter int, isGood bool) {
	eventType := "part_ok"
	if !isGood {
		eventType = "part_nok"
	}
	h.Broadcast(eventType, map[string]interface{}{
		"line":     line,
		"material": material,
		"counter":  counter,
	})
}

// BroadcastLineStatus отправляет событие об изменении статуса линии
func (h *Hub) BroadcastLineStatus(line string, isOnline bool) {
	h.Broadcast("line_status", map[string]interface{}{
		"line":     line,
		"isOnline": isOnline,
	})
}

// BroadcastLineStatus отправляет событие об включении/отключении линии
func (h *Hub) BroadcastLineActive(line string, isActive bool) {
	h.Broadcast("line_active", map[string]interface{}{
		"line":     line,
		"isActive": isActive,
	})
}

// BroadcastShipmentCompleted отправляет событие о завершении отгрузки
func (h *Hub) BroadcastShipmentCompleted(shipmentID int, number *int) {
	shipmentNumber := ""
	if number != nil {
		shipmentNumber = string(rune(*number))
	}
	h.Broadcast("shipment_completed", map[string]interface{}{
		"shipmentId": shipmentID,
		"number":     shipmentNumber,
	})
}

// BroadcastPlanUpdated отправляет событие об обновлении плана
func (h *Hub) BroadcastPlanUpdated(planID int, materialCode string, planDate string) {
	h.Broadcast("plan_updated", map[string]interface{}{
		"planId":       planID,
		"materialCode": materialCode,
		"planDate":     planDate,
	})
}

// GetClientCount возвращает количество подключённых клиентов
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
