package irc

import (
	"io"
	"log"
	"strings"
	"sync"
)

// cmdResponseWriter is a simple writer that abstracts away the Msg struct.
type cmdResponseWriter struct {
	send     chan<- *Msg
	receiver string
}

// Compose a message to send back to the receiver (channel).
func (w cmdResponseWriter) Write(p []byte) (int, error) {
	w.send <- &Msg{Cmd: "PRIVMSG", Params: []string{w.receiver, string(p)}}
	return len(p), nil
}

// The Cmd interface responds to incoming chat commands.
type Cmd interface {
	Respond(body, source string, w io.Writer)
}

// A CmdFunc responds to incoming chat commands.
type CmdFunc func(body, source string, w io.Writer)

// Shim struct to allow users who don't need state to more easily register a
// CmdFunc while not modifying our handling code.
type cmd struct {
	cmdFunc CmdFunc
}

// Respond on our shim just passes through to the user func.
func (c cmd) Respond(body, source string, w io.Writer) {
	c.cmdFunc(body, source, w)
}

// A CmdHandler dispatches for a group of commands with a common prefix.
type CmdHandler struct {
	prefix  string
	cmdsMtx sync.Mutex
	cmds    map[string]Cmd
}

// NewCmdHandler creates a new CmdHandler with the given command prefix.
func NewCmdHandler(prefix string) *CmdHandler {
	return &CmdHandler{prefix: prefix, cmds: make(map[string]Cmd)}
}

// Accepts for a CmdHandler ensures the msg contains a chat command.
func (cmdHandler *CmdHandler) Accepts(msg *Msg) bool {
	isPrivmsg := msg.Cmd == "PRIVMSG"
	hasCmdPrefix := len(msg.Params) == 2 &&
		strings.HasPrefix(msg.Params[1], cmdHandler.prefix)
	return isPrivmsg && hasCmdPrefix
}

// Handle for a CmdHandler extracts the relevant parts of a command msg and
// dispatches to a Cmd, if one is found with the given name.
func (cmdHandler *CmdHandler) Handle(msg *Msg, send chan<- *Msg) {
	receiver, body, err := msg.ExtractPrivmsg()
	if err != nil {
		log.Println(err)
		return
	}
	nameAndBody := strings.SplitN(body, " ", 2)
	name := strings.TrimPrefix(nameAndBody[0], cmdHandler.prefix)
	if len(nameAndBody) > 1 {
		body = nameAndBody[1]
	} else {
		body = ""
	}
	source, err := msg.ExtractNick()
	if err != nil {
		log.Println(err)
		return
	}
	cmdHandler.cmdsMtx.Lock()
	ccmd, ok := cmdHandler.cmds[name]
	cmdHandler.cmdsMtx.Unlock()
	if ok {
		go ccmd.Respond(body, source,
			cmdResponseWriter{receiver: receiver, send: send})
	}
}

func (cmdHandler *CmdHandler) RegisteredNames() (names []string) {
	cmdHandler.cmdsMtx.Lock()
	defer cmdHandler.cmdsMtx.Unlock()
	for name := range cmdHandler.cmds {
		names = append(names, name)
	}
	return
}

// Register adds a Cmd to be executed when the given name is matched.
func (cmdHandler *CmdHandler) Register(name string, cmd Cmd) {
	cmdHandler.cmdsMtx.Lock()
	defer cmdHandler.cmdsMtx.Unlock()
	cmdHandler.cmds[name] = cmd
}

// RegisterFunc adds a CmdFunc to be executed when the given name is matched.
func (cmdHandler *CmdHandler) RegisterFunc(name string, cmdFunc CmdFunc) {
	cmdHandler.Register(name, cmd{cmdFunc: cmdFunc})
}
