package backend

import (
	"encoding/json"
	"time"
)

// Message types
const (
	// Actor -> Backend
	ActionRegister      = "actor.register"
	ActionUnregister    = "actor.unregister"
	ActionStatusUpdate  = "actor.status_update"
	ActionHeartbeat     = "actor.heartbeat"
	ActionCommandResult = "actor.command_result"

	// Backend -> Actor
	ActionStart  = "actor.start"
	ActionStop   = "actor.stop"
	ActionStatus = "actor.status"
	ActionConfig = "actor.config"
	ActionCreate = "actor.create"
	ActionDelete = "actor.delete"
)

// Message represents a message sent to/from backend
type Message struct {
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

// Response represents a response from backend
type Response struct {
	Success   bool                   `json:"success"`
	Code      int                    `json:"code"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

// Command represents a command from backend to actor
type Command struct {
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Result    interface{} `json:"result,omitempty"`
}

// RegisterData contains actor registration information
type RegisterData struct {
	RobotID   string `json:"robot_id"`
	Exchange  string `json:"exchange"`
	Version   string `json:"version"`
	TenantID  uint32 `json:"tenant_id,omitempty"`
}

// StatusData contains actor status information
type StatusData struct {
	RobotID   string   `json:"robot_id"`
	Status    string   `json:"status"` // running, stopped, error
	Balance   float64  `json:"balance,omitempty"`
	Positions []string `json:"positions,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// NewMessage creates a new message
func NewMessage(action string, data map[string]interface{}) *Message {
	return &Message{
		Action:    action,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// NewMessageWithRequestID creates a new message with request ID
func NewMessageWithRequestID(action string, data map[string]interface{}, requestID string) *Message {
	return &Message{
		Action:    action,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
}

// ToJSON converts message to JSON bytes
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage parses JSON bytes to message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseResponse parses JSON bytes to response
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NewRegisterMessage creates a registration message
func NewRegisterMessage(robotID, exchange, version string, tenantID uint32) *Message {
	return NewMessage(ActionRegister, map[string]interface{}{
		"robot_id":  robotID,
		"exchange":  exchange,
		"version":   version,
		"tenant_id": tenantID,
	})
}

// NewStatusUpdateMessage creates a status update message
func NewStatusUpdateMessage(robotID, status string, balance float64) *Message {
	return NewMessage(ActionStatusUpdate, map[string]interface{}{
		"robot_id": robotID,
		"status":   status,
		"balance":  balance,
	})
}

// NewHeartbeatMessage creates a heartbeat message
func NewHeartbeatMessage(robotID string) *Message {
	return NewMessage(ActionHeartbeat, map[string]interface{}{
		"robot_id": robotID,
	})
}

// NewCommandResultMessage creates a command result message
func NewCommandResultMessage(requestID string, success bool, result interface{}, errMsg string) *Message {
	data := map[string]interface{}{
		"request_id": requestID,
		"success":    success,
	}
	if result != nil {
		data["result"] = result
	}
	if errMsg != "" {
		data["error"] = errMsg
	}
	return NewMessage(ActionCommandResult, data)
}
