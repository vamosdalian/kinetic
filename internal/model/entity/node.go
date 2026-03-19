package entity

import "time"

type NodeEntity struct {
	ID            string    // Unique identifier (UUID)
	Hostname      string    // Machine hostname
	IP            string    // IP address
	Controller    bool      // Is this node a controller
	Status        string    // Node status (e.g., "online", "offline", "busy")
	Labels        string    // JSON string storing tags/labels for sheduling (e.g. key-value pairs)
	LastHeartbeat time.Time // Timestamp of the last heartbeat received
}
