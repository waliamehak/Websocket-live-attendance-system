// internal/websocket/websocket.go
package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/database"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/models"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/session"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ClientInfo struct {
	UserID string
	Role   string
}

// track all connected clients (guarded)
var (
	clients   = make(map[*websocket.Conn]ClientInfo)
	clientsMu sync.RWMutex
)

type WSMessage struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

func HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(401, gin.H{"error": "token missing"})
		return
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade failed:", err)
		return
	}

	clientsMu.Lock()
	clients[conn] = ClientInfo{UserID: claims.UserID, Role: claims.Role}
	clientsMu.Unlock()

	log.Printf("client connected: %s (%s)", claims.UserID, claims.Role)

	// Broadcast new peer to all existing clients
	broadcastPeerJoined(claims.UserID, claims.Role)

	go handleMessages(conn)
}

func handleMessages(conn *websocket.Conn) {
	defer func() {
		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()

		conn.Close()
		log.Println("client disconnected")
	}()

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			break
		}

		switch msg.Event {
		case "ATTENDANCE_MARKED":
			handleAttendanceMarked(conn, msg)
		case "TODAY_SUMMARY":
			handleTodaySummary(conn, msg)
		case "MY_ATTENDANCE":
			handleMyAttendance(conn, msg)
		case "DONE":
			handleDone(conn, msg)
		case "WEBRTC_OFFER":
			handleWebRTCSignal(conn, msg)
		case "WEBRTC_ANSWER":
			handleWebRTCSignal(conn, msg)
		case "WEBRTC_ICE_CANDIDATE":
			handleWebRTCSignal(conn, msg)
		default:
			sendError(conn, "unknown event type")
		}
	}
}

func handleAttendanceMarked(conn *websocket.Conn, msg WSMessage) {
	client := getClient(conn)

	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	s := session.Get()
	if s == nil {
		sendError(conn, "No active attendance session")
		return
	}

	studentID, ok := msg.Data["studentId"].(string)
	if !ok || studentID == "" {
		sendError(conn, "invalid studentId")
		return
	}

	status, ok := msg.Data["status"].(string)
	if !ok || (status != "present" && status != "absent") {
		sendError(conn, "invalid status")
		return
	}

	session.WithWrite(func(s *session.ActiveSession) {
		s.Attendance[studentID] = status
	})

	broadcast(WSMessage{
		Event: "ATTENDANCE_MARKED",
		Data: map[string]interface{}{
			"studentId": studentID,
			"status":    status,
		},
	})
}

func handleTodaySummary(conn *websocket.Conn, msg WSMessage) {
	client := getClient(conn)

	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	s := session.Get()
	if s == nil {
		sendError(conn, "No active attendance session")
		return
	}

	present, absent := 0, 0
	for _, st := range s.Attendance {
		if st == "present" {
			present++
		} else if st == "absent" {
			absent++
		}
	}
	total := present + absent

	broadcast(WSMessage{
		Event: "TODAY_SUMMARY",
		Data: map[string]interface{}{
			"present": present,
			"absent":  absent,
			"total":   total,
		},
	})
}

func handleMyAttendance(conn *websocket.Conn, msg WSMessage) {
	client := getClient(conn)

	if client.Role != "student" {
		sendError(conn, "Forbidden, student event only")
		return
	}

	s := session.Get()
	if s == nil {
		sendError(conn, "No active attendance session")
		return
	}

	status, found := s.Attendance[client.UserID]
	if !found {
		status = "not yet updated"
	}

	sendToClient(conn, WSMessage{
		Event: "MY_ATTENDANCE",
		Data: map[string]interface{}{
			"status": status,
		},
	})
}

func handleDone(conn *websocket.Conn, msg WSMessage) {
	client := getClient(conn)

	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	s := session.Get()
	if s == nil {
		sendError(conn, "No active attendance session")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	classID, err := primitive.ObjectIDFromHex(s.ClassID)
	if err != nil {
		sendError(conn, "invalid class id in session")
		return
	}

	// fetch class
	classCollection := database.DB.Collection("classes")
	var class models.Class
	err = classCollection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		sendError(conn, "failed to fetch class")
		return
	}

	// Clear active room
	classCollection.UpdateOne(ctx, bson.M{"_id": classID}, bson.M{
		"$unset": bson.M{"activeRoomId": ""},
	})

	// ensure absent for unmarked students
	session.WithWrite(func(s *session.ActiveSession) {
		for _, studentID := range class.StudentIDs {
			sidHex := studentID.Hex()
			if _, exists := s.Attendance[sidHex]; !exists {
				s.Attendance[sidHex] = "absent"
			}
		}
	})

	attendanceCollection := database.DB.Collection("attendance")

	present, absent := 0, 0

	// snapshot to iterate without holding session lock for DB ops
	att := map[string]string{}
	for k, v := range s.Attendance {
		att[k] = v
	}

	for studentIDStr, status := range att {
		studentObjID, err := primitive.ObjectIDFromHex(studentIDStr)
		if err != nil {
			continue
		}

		attendanceCollection.DeleteOne(ctx, bson.M{
			"classId":   classID,
			"studentId": studentObjID,
		})

		rec := models.Attendance{
			ID:        primitive.NewObjectID(),
			ClassID:   classID,
			StudentID: studentObjID,
			Status:    status,
		}

		if _, err := attendanceCollection.InsertOne(ctx, rec); err != nil {
			log.Println("failed to save attendance:", err)
		}

		if status == "present" {
			present++
		} else {
			absent++
		}
	}

	total := present + absent

	broadcast(WSMessage{
		Event: "DONE",
		Data: map[string]interface{}{
			"message": "Attendance persisted",
			"present": present,
			"absent":  absent,
			"total":   total,
		},
	})

	session.Clear()
}

func getClient(conn *websocket.Conn) ClientInfo {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	return clients[conn]
}

func sendToClient(conn *websocket.Conn, msg WSMessage) {
	if err := conn.WriteJSON(msg); err != nil {
		log.Println("write error:", err)
	}
}

func broadcast(msg WSMessage) {
	clientsMu.RLock()
	conns := make([]*websocket.Conn, 0, len(clients))
	for c := range clients {
		conns = append(conns, c)
	}
	clientsMu.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteJSON(msg); err != nil {
			log.Println("broadcast error:", err)
			conn.Close()
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
		}
	}
}

func sendError(conn *websocket.Conn, message string) {
	sendToClient(conn, WSMessage{
		Event: "ERROR",
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// handleWebRTCSignal relays WebRTC signaling messages between peers
func handleWebRTCSignal(conn *websocket.Conn, msg WSMessage) {
	client := getClient(conn)

	// Get target peer ID from message
	targetID, ok := msg.Data["targetId"].(string)
	if !ok || targetID == "" {
		sendError(conn, "missing targetId in WebRTC message")
		return
	}

	// Forward signal to target peer
	clientsMu.RLock()
	for c, info := range clients {
		if info.UserID == targetID {
			// Add sender info
			msg.Data["fromId"] = client.UserID
			msg.Data["fromRole"] = client.Role

			c.WriteJSON(msg)
			clientsMu.RUnlock()
			return
		}
	}
	clientsMu.RUnlock()

	sendError(conn, "target peer not connected")
}

func broadcastPeerJoined(newUserID, newUserRole string) {
	userName := getUserName(newUserID)
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	for conn, info := range clients {
		if info.UserID != newUserID {
			conn.WriteJSON(WSMessage{
				Event: "PEER_JOINED",
				Data: map[string]interface{}{
					"userId": newUserID,
					"role":   newUserRole,
					"name":   userName,
				},
			})
		}
	}
}

func getUserName(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Unknown"
	}

	var user struct {
		Name string `bson:"name"`
	}

	err = database.DB.Collection("users").FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return "Unknown"
	}

	return user.Name
}
