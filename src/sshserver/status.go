package sshserver

const (
	StatusContinue = "continue"
	StatusStop     = "stop"
)

const (
	StatusReady int32 = iota
	StatusRunning
	StatusStopping
	StatusFinished
)
