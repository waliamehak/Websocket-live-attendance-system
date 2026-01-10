# Live Attendance System

Real-time attendance tracking system with WebSocket support.

## Tech Stack

- **Backend:** Go + Gin
- **Database:** MongoDB
- **WebSocket:** Gorilla WebSocket
- **Auth:** JWT + bcrypt

## Setup

### Prerequisites

- Go 1.21+
- MongoDB (local or Atlas)

### Installation
```bash
# clone repo
git clone https://github.com/waliamehak/WebSocket-live-attendance-system.git
cd WebSocket-live-attendance-system

# install dependencies
go mod download

# setup env
cp .env.example .env
# edit .env with your MongoDB URI

# run server
go run cmd/server/main.go
```

### Environment Variables
```
PORT=3000
MONGODB_URI=mongodb://localhost:27017/attendance
JWT_SECRET=your_secret_key
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
- `GET /students` - List all students (teacher only)

### Attendance
- `POST /attendance/start` - Start attendance session (teacher only)
- `GET /class/:id/my-attendance` - Check my attendance (student only)

### WebSocket
- `ws://localhost:3000/ws?token=<JWT_TOKEN>`

#### Events
- `ATTENDANCE_MARKED` - Mark student attendance (teacher → broadcast)
- `TODAY_SUMMARY` - Get attendance summary (teacher → broadcast)
- `MY_ATTENDANCE` - Check your status (student → unicast)
- `DONE` - End session & persist to DB (teacher → broadcast)

## Project Structure
Inspiration to the folder structure (https://github.com/golang-standards/project-layout)

## Testing

Use Postman or Thunder Client to test HTTP endpoints.

For quick manual testing, this repo also includes a lightweight `static/index.html` test page built with AI assistance to exercise the REST APIs and WebSocket flow in a browser.

For WebSocket testing, use tools like:
- Postman WebSocket
- wscat: `wscat -c "ws://localhost:3000/ws?token=YOUR_TOKEN"`

