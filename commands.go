package lirc

import "strconv"

// Command describes a command that can be sent to lirc.
type Command interface {
	// EncodeCommand encodes the command and arguments as a slice of strings.
	EncodeCommand() []string
}

// SendOnce tells lircd to send the IR signal associated with the given remote
// control and button name, and then repeat it repeats times. repeats is a
// decimal number between 0 and repeat_max. The latter can be given as a
// --repeat-max command line argument to lircd, and defaults to 600. If repeats
// is not specified or is less than the minimum number of repeats for the
// selected remote control, the minimum value will be used.
type SendOnce struct {
	RemoteControl string
	ButtonName    string
	Repeats       uint // optional
}

// EncodeCommand implements the [Command] interface.
func (s SendOnce) EncodeCommand() []string {
	if s.Repeats == 0 {
		return []string{"SEND_ONCE", s.RemoteControl, s.ButtonName}
	}
	return []string{"SEND_ONCE", s.RemoteControl, s.ButtonName, strconv.Itoa(int(s.Repeats))}
}

// SendStart tells lircd to start repeating the given button until it receives a
// [SendStop] command. However, the number of repeats is limited to repeat_max.
// lircd won't accept any new send commands while it is repeating.
type SendStart struct {
	RemoteControl string
	ButtonName    string
}

// EncodeCommand implements the [Command] interface.
func (s SendStart) EncodeCommand() []string {
	return []string{"SEND_START", s.RemoteControl, s.ButtonName}
}

// SendStop tells lircd to abort a [SendStart] command.
type SendStop struct {
	RemoteControl string
	ButtonName    string
}

// EncodeCommand implements the [Command] interface.
func (s SendStop) EncodeCommand() []string {
	return []string{"SEND_STOP", s.RemoteControl, s.ButtonName}
}

// List returns a list of all defined remote controls.
type List struct {
	RemoteControl string
}

// EncodeCommand implements the [Command] interface.
func (l List) EncodeCommand() []string {
	if l.RemoteControl == "" {
		return []string{"LIST"}
	}
	return []string{"LIST", l.RemoteControl}
}

// SetInputLog starts logging all received data on that file. The log is printable
// lines as defined in mode2(1) describing pulse/space durations.
type SetInputLog struct {
	Path string
}

// EncodeCommand implements the [Command] interface.
func (s SetInputLog) EncodeCommand() []string {
	if s.Path == "" {
		return []string{"SET_INPUTLOG"}
	}
	return []string{"SET_INPUTLOG", s.Path}
}

// DrvOption makes lircd invoke the drvctl_func(DRVCTL_SET_OPTION, option) with
// option being made up by the parsed key and value. The return package reflects
// the outcome of the drvctl_func call.
type DrvOption struct {
	Key   string
	Value string
}

// EncodeCommand implements the [Command] interface.
func (d DrvOption) EncodeCommand() []string {
	return []string{"DRV_OPTION", d.Key, d.Value}
}

// Simulate instructs lircd to send this to all clients i. e., to simulate that
// this key has been decoded. The key data must be formatted exactly as the packet
// described in [SOCKET BROADCAST MESSAGES FORMAT], notably is the number of digits
// in code and repeat count hardcoded. This command is only accepted if the
// --allow-simulate command line option is active.
type Simulate struct {
	Key  string
	Data string
}

// EncodeCommand implements the [Command] interface.
func (s Simulate) EncodeCommand() []string {
	return []string{"SIMULATE", s.Key, s.Data}
}

// SetTransmitters makes lircd invoke the drvctl_func(LIRC_SET_TRANSMITTER_MASK,
// &channels), where channels is the decoded value of transmitter mask. See lirc(4)
// for more information.
type SetTransmitters struct {
	TransmitterMask string
}

// EncodeCommand implements the [Command] interface.
func (s SetTransmitters) EncodeCommand() []string {
	return []string{"SET_TRANSMITTERS", s.TransmitterMask}
}

// Version tells lircd to send a version packet response.
type Version struct{}

// EncodeCommand implements the [Command] interface.
func (v Version) EncodeCommand() []string {
	return []string{"VERSION"}
}
