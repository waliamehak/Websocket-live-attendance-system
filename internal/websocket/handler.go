package websocket

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/database"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/models"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for now
	},
}

// track all connected clients
var clients = make(map[*websocket.Conn]ClientInfo)

type ClientInfo struct {
	UserID string
	Role   string
}

// session state - same as in handlers/attendance.go but need it here too
type ActiveSession struct {
	ClassID    string
	StartedAt  string
	Attendance map[string]string
}

var activeSession *ActiveSession

type WSMessage struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

func HandleWebSocket(c *gin.Context) {
	// grab token from query
	token := c.Query("token")
	if token == "" {
		c.JSON(401, gin.H{"error": "token missing"})
		return
	}

	// verify jwt
	claims, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid token"})
		return
	}

	// upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade failed:", err)
		return
	}

	// store client info
	clients[conn] = ClientInfo{
		UserID: claims.UserID,
		Role:   claims.Role,
	}

	log.Printf("client connected: %s (%s)", claims.UserID, claims.Role)

	// handle messages
	go handleMessages(conn)
}

func handleMessages(conn *websocket.Conn) {
	defer func() {
		delete(clients, conn)
		conn.Close()
		log.Println("client disconnected")
	}()

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			break
		}

		// route to correct handler
		switch msg.Event {
		case "ATTENDANCE_MARKED":
			handleAttendanceMarked(conn, msg)
		case "TODAY_SUMMARY":
			handleTodaySummary(conn, msg)
		case "MY_ATTENDANCE":
			handleMyAttendance(conn, msg)
		case "DONE":
			handleDone(conn, msg)
		default:
			sendError(conn, "unknown event type")
		}
	}
}

func handleAttendanceMarked(conn *websocket.Conn, msg WSMessage) {
	client := clients[conn]

	// only teachers can mark
	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	if activeSession == nil {
		sendError(conn, "No active attendance session")
		return
	}

	studentID, ok := msg.Data["studentId"].(string)
	if !ok {
		sendError(conn, "invalid studentId")
		return
	}

	status, ok := msg.Data["status"].(string)
	if !ok || (status != "present" && status != "absent") {
		sendError(conn, "invalid status")
		return
	}

	// update in memory
	activeSession.Attendance[studentID] = status

	// broadcast to everyone
	broadcast(WSMessage{
		Event: "ATTENDANCE_MARKED",
		Data: map[string]interface{}{
			"studentId": studentID,
			"status":    status,
		},
	})
}

func handleTodaySummary(conn *websocket.Conn, msg WSMessage) {
	client := clients[conn]

	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	if activeSession == nil {
		sendError(conn, "No active attendance session")
		return
	}

	// count present/absent
	present := 0
	absent := 0

	for _, status := range activeSession.Attendance {
		if status == "present" {
			present++
		} else if status == "absent" {
			absent++
		}
	}

	total := present + absent

	// broadcast summary
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
	client := clients[conn]

	if client.Role != "student" {
		sendError(conn, "Forbidden, student event only")
		return
	}

	if activeSession == nil {
		sendError(conn, "No active attendance session")
		return
	}

	// check if student's attendance is marked
	status, found := activeSession.Attendance[client.UserID]
	if !found {
		status = "not yet updated"
	}

	// send only to this student
	sendToClient(conn, WSMessage{
		Event: "MY_ATTENDANCE",
		Data: map[string]interface{}{
			"status": status,
		},
	})
}

func handleDone(conn *websocket.Conn, msg WSMessage) {
	client := clients[conn]

	if client.Role != "teacher" {
		sendError(conn, "Forbidden, teacher event only")
		return
	}

	if activeSession == nil {
		sendError(conn, "No active attendance session")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	classID, _ := primitive.ObjectIDFromHex(activeSession.ClassID)

	// get all students in class
	classCollection := database.DB.Collection("classes")
	var class models.Class
	err := classCollection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		sendError(conn, "failed to fetch class")
		return
	}

	// mark absent students who weren't marked
	for _, studentID := range class.StudentIDs {
		sidHex := studentID.Hex()
		if _, exists := activeSession.Attendance[sidHex]; !exists {
			activeSession.Attendance[sidHex] = "absent"
		}
	}

	// persist to db
	attendanceCollection := database.DB.Collection("attendance")

	present := 0
	absent := 0

	for studentIDStr, status := range activeSession.Attendance {
		studentObjID, _ := primitive.ObjectIDFromHex(studentIDStr)

		// delete existing record if any
		attendanceCollection.DeleteOne(ctx, bson.M{
			"classId":   classID,
			"studentId": studentObjID,
		})

		// insert new record
		attendance := models.Attendance{
			ID:        primitive.NewObjectID(),
			ClassID:   classID,
			StudentID: studentObjID,
			Status:    status,
		}

		_, err := attendanceCollection.InsertOne(ctx, attendance)
		if err != nil {
			log.Println("failed to save attendance:", err)
		}

		if status == "present" {
			present++
		} else {
			absent++
		}
	}

	total := present + absent

	// broadcast done message
	broadcast(WSMessage{
		Event: "DONE",
		Data: map[string]interface{}{
			"message": "Attendance persisted",
			"present": present,
			"absent":  absent,
			"total":   total,
		},
	})

	// clear session
	activeSession = nil
}

// helper: send to single client
func sendToClient(conn *websocket.Conn, msg WSMessage) {
	err := conn.WriteJSON(msg)
	if err != nil {
		log.Println("write error:", err)
	}
}

// helper: broadcast to all clients
func broadcast(msg WSMessage) {
	for conn := range clients {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Println("broadcast error:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

// helper: send error message
func sendError(conn *websocket.Conn, message string) {
	sendToClient(conn, WSMessage{
		Event: "ERROR",
		Data: map[string]interface{}{
			"message": message,
		},
	})
}
