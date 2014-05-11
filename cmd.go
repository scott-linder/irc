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

func (f CmdFunc) Respond(body, source string, w io.Writer) {
	f(body, source, w)
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
func (c *CmdHandler) Accepts(msg *Msg) bool {
	isPrivmsg := msg.Cmd == "PRIVMSG"
	hasCmdPrefix := len(msg.Params) == 2 &&
		strings.HasPrefix(msg.Params[1], c.prefix)
	return isPrivmsg && hasCmdPrefix
}

// Handle for a CmdHandler extracts the relevant parts of a command msg and
// dispatches to a Cmd, if one is found with the given name.
func (c *CmdHandler) Handle(msg *Msg, send chan<- *Msg) {
	receiver, body, err := msg.ExtractPrivmsg()
	if err != nil {
		log.Println(err)
		return
	}
	nameAndBody := strings.SplitN(body, " ", 2)
	name := strings.TrimPrefix(nameAndBody[0], c.prefix)
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
	c.cmdsMtx.Lock()
	ccmd, ok := c.cmds[name]
	c.cmdsMtx.Unlock()
	if ok {
		go ccmd.Respond(body, source,
			cmdResponseWriter{receiver: receiver, send: send})
	}
}

func (c *CmdHandler) RegisteredNames() (names []string) {
	c.cmdsMtx.Lock()
	defer c.cmdsMtx.Unlock()
	for name := range c.cmds {
		names = append(names, name)
	}
	return
}

// Register adds a Cmd to be executed when the given name is matched.
func (c *CmdHandler) Register(name string, cmd Cmd) {
	c.cmdsMtx.Lock()
	defer c.cmdsMtx.Unlock()
	c.cmds[name] = cmd
}

// RegisterFunc adds a CmdFunc to be executed when the given name is matched.
func (c *CmdHandler) RegisterFunc(name string, cmdFunc CmdFunc) {
	c.Register(name, Cmd(cmdFunc))
}
