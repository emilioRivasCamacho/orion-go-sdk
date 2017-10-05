package ologger

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
