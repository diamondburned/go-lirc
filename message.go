package lirc

import "errors"

// ButtonPress represents the IR Remote Key Press ButtonPress
type ButtonPress struct {
	// Code is a 16 hexadecimal digits number encoding of the IR signal.
	// It's usage in applications is deprecated and it should be ignored.
	Code uint16
	// RepeatCount shows how long the user has been holding down a button.
	// The counter will start at 0 and increment each time a new IR signal has been received.
	RepeatCount uint
	// ButtonName is the name of a key defined in the lircd.conf file.
	ButtonName string
	// RemoteControlName is the mandatory name attribute in the lircd.conf config file.
	RemoteControlName string
}

// CommandReply is the message received after sending a command.
type CommandReply struct {
	// Command is the command that was sent to lircd.
	Command string
	// Success is whether the command was successful.
	Success bool
	// Data is the data received from lircd.
	Data []string
}

// ErrUnsuccessfulCommand is returned with a reply when a command was not successful.
var ErrUnsuccessfulCommand = errors.New("lirc: unsuccessful command")
