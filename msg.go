package irc

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrBadMsg = errors.New("Invalid IRC message.")
)

// Msg represents the essential components identifying a received for
// to-be-sent IRC message.
type Msg struct {
	Raw    string
	Prefix string
	Cmd    string
	Params []string
}

// paramsString formats the parameter list, handling the trailing edge case.
func (msg *Msg) paramsString() (str string) {
	for i, param := range msg.Params {
		if i == len(msg.Params)-1 {
			// Edge case for trailing (last) parameter.
			str += ":" + param
		} else {
			str += param + " "
		}
	}
	return
}

// String converts the Msg struct to an IRC message string.
func (msg Msg) String() string {
	return fmt.Sprintf(":%v %v %v",
		msg.Prefix, msg.Cmd, msg.paramsString())
}

// ParseMsg accepts a raw IRC message string and parses it into a Msg struct.
func ParseMsg(raw string) (*Msg, error) {
	msg := &Msg{Raw: strings.TrimSpace(raw)}
	if msg.Raw == "" {
		return nil, ErrBadMsg
	}
	// Prefix is optional, but must be of the form ':prefix rest'.
	if strings.HasPrefix(raw, ":") {
		// Break the string up into a prefix and "the rest".
		prefixAndRest := strings.SplitN(raw, " ", 2)
		msg.Prefix = strings.TrimLeft(prefixAndRest[0], ":")
		raw = prefixAndRest[1]
	}
	trailing := ""
	// Trailing part is also optional, but is always identified by " :".
	if strings.Contains(raw, " :") {
		// Break the string up into "the rest" and the trailing.
		restAndTrailing := strings.SplitN(raw, " :", 2)
		raw = restAndTrailing[0]
		trailing = restAndTrailing[1]
	}
	// We now have the command and non-trailing parameters.
	cmdAndParams := strings.Split(raw, " ")
	msg.Cmd = cmdAndParams[0]
	msg.Params = cmdAndParams[1:]
	if trailing != "" {
		msg.Params = append(msg.Params, trailing)
	}
	return msg, nil
}

// ExtractPrivmsg attempts to extract the relevant parts of a privmessage.
func (msg *Msg) ExtractPrivmsg() (source string, body string, err error) {
	if msg.Cmd == "PRIVMSG" && len(msg.Params) == 2 {
		source = msg.Params[0]
		body = msg.Params[1]
	} else {
		err = errors.New("Malformed PRIVMSG")
	}
	return
}

// ExtractNick attempts to extract the sender nick from the message prefix.
func (msg *Msg) ExtractNick() (nick string, err error) {
	if strings.Contains(msg.Prefix, "!") && msg.Prefix != "" {
		nick = strings.Split(msg.Prefix, "!")[0]
	} else {
		err = errors.New("Unable to extract nick from prefix.")
	}
	return
}
