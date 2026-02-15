# Real-Time Video Classroom with Attendance Tracking

Real-time video conferencing platform for online classrooms with integrated WebSocket-based attendance tracking.

## Tech Stack

- **Backend:** Go + Gin
- **Database:** MongoDB
- **WebRTC:** Peer-to-peer video streaming
- **WebSocket:** Gorilla WebSocket for real-time signaling
- **Auth:** Auth0 + RS256 JWT


## Features

- **Live Video Conferencing:** WebRTC-powered video classroom with:
  - Automatic peer discovery and connection
  - Camera/microphone controls
  - Multi-participant support
  - Real-time connection status
  - Low-latency peer-to-peer streaming
- **Real-time Attendance Tracking:** WebSocket-based attendance with broadcast and per-user messaging
- **Class Management:** Teachers create classes and enroll students
- **Authentication:** Auth0-based authentication with RS256 JWT and role-based access control (RBAC)

## Setup

### Prerequisites

- Go 1.21+
- MongoDB (local or Atlas)
- Project structure inspired by [golang-standards/project-layout](https://github.com/golang-standards/project-layout)


### Installation
```bash
# Clone repo
git clone https://github.com/waliamehak/WebSocket-live-attendance-system.git
cd WebSocket-live-attendance-system

# Install dependencies
go mod download

# Setup environment
cp .env.example .env
# Edit .env with your MongoDB URI

# Run server
go run cmd/server/main.go
```

### Environment Variables
```
PORT=3000
MONGODB_URI=mongodb://localhost:27017/attendance
AUTH0_DOMAIN=your-tenant.us.auth0.com
AUTH0_AUDIENCE=https://liveclassroom.api
AUTH0_NAMESPACE=https://liveclassroom.app
```

## API Endpoints

### Auth
- `POST /auth/signup` - Create account
- `POST /auth/login` - Login
- `GET /auth/me` - Get current user (requires auth)

### Classes
- `POST /class` - Create class (teacher only)
- `POST /class/:id/add-student` - Add student to class (teacher only)
- `GET /class/:id` - Get class details
- `GET /class/:id/room` - Get active video room status
- `GET /students` - List all students (teacher only)

### Attendance & Video Sessions
- `POST /attendance/start` - Start attendance session & create video room (teacher only)
- `POST /attendance/end` - End session & save to database (teacher only)
- `GET /class/:id/my-attendance` - Check my attendance (student only)

### Video Classroom
- Access via: `/static/classroom.html`
- Login with class credentials
- Requires active attendance session

## WebSocket

Connect to: `ws://localhost:3000/ws?token=<JWT_TOKEN>`

### Events

**WebRTC Signaling:**
- `PEER_JOINED` - New peer connected (broadcast)
- `WEBRTC_OFFER` - WebRTC offer signal
- `WEBRTC_ANSWER` - WebRTC answer signal
- `WEBRTC_ICE_CANDIDATE` - ICE candidate exchange

**Attendance:**
- `ATTENDANCE_MARKED` - Mark student attendance (teacher → broadcast)
- `TODAY_SUMMARY` - Get attendance summary (teacher → broadcast)
- `MY_ATTENDANCE` - Check your status (student → unicast)
- `DONE` - End session & persist to DB (teacher → broadcast)

## Testing

### HTTP Endpoints
Use Postman, Thunder Client, or the included `static/index.html` test page.

### WebSocket
- Postman WebSocket
- wscat: `wscat -c "ws://localhost:3000/ws?token=YOUR_TOKEN"`

### Video Classroom
1. Start server: `go run cmd/server/main.go`
2. Login as teacher → Create class → Start attendance session
3. Open `http://localhost:3000/static/classroom.html`
4. Login and join with class ID
5. Open second browser window as student and join same class
6. Video connections establish automatically via WebRTC

## How It Works

1. **Teacher** creates a class and enrolls students
2. **Teacher** starts an attendance session (creates a video room)
3. **Students** join the video classroom via WebSocket
4. **WebRTC** establishes peer-to-peer video connections automatically
5. **Attendance** is tracked in real-time via WebSocket events
6. **Teacher** ends session, data persists to MongoDB