package logger

const (
	// EMERGENCY level - system is unusable
	EMERGENCY int = iota
	// ALERT level - action must be taken immediately
	ALERT
	// CRITICAL level - critical conditions
	CRITICAL
	// ERROR level - error conditions
	ERROR
	// WARNING level - warning conditions
	WARNING
	// NOTICE level - normal, but significant, condition
	NOTICE
	// INFO level - informational message
	INFO
	// DEBUG level
	DEBUG
)

func levelToNumber(level string) int {
	switch level {
	case "alert":
		return ALERT
	case "warning":
		return WARNING
	case "critical":
		return CRITICAL
	case "emergency":
		return EMERGENCY
	case "error":
		return ERROR
	case "info":
		return INFO
	case "notice":
		return NOTICE
	case "debug":
		return DEBUG
	default:
		return DEBUG
	}
}
