/*
   Package irc provides a framework for writing IRC clients, specifically bots.
*/
package irc

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
)

// MsgHandler handles messages and optionally sends responses on chan send.
type Handler interface {
	Accept(msg *Msg) bool
	Handle(msg *Msg, send chan<- *Msg)
}

// Client is an IRC connection which handles message dispatch to a MsgHandler.
type Client struct {
	conn     io.ReadWriteCloser
	writer   *textproto.Writer
	reader   *textproto.Reader
	send     chan *Msg
	recv     chan *Msg
	handlers []Handler
}

// Dial connects to an IRC host.
func Dial(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	client := Client{
		conn:     conn,
		writer:   textproto.NewWriter(bufio.NewWriter(conn)),
		reader:   textproto.NewReader(bufio.NewReader(conn)),
		send:     make(chan *Msg, 5),
		recv:     make(chan *Msg, 5),
		handlers: make([]Handler, 0),
	}
	return &client, nil
}

// Register registers a handler for dispatch.
func (client *Client) Register(handler Handler) {
	client.handlers = append(client.handlers, handler)
}

//
func (client *Client) Nick(user string) {
	client.writer.PrintfLine("NICK %v", user)
	client.writer.PrintfLine("USER %v %v %v :%v", user)
}

func (client *Client) Join(channel string) {
	client.writer.PrintfLine("JOIN %v", channel)
}

// Listen puts an irc connection into a loop, parsing and dispatching recieved
// messages to a handler, as well as sending outgoing messages.
func (client *Client) Listen() {
	go func() {
		for {
			line, err := client.reader.ReadLine()
			if err != nil {
				log.Println(err)
				continue
			}
			msg, err := ParseMsg(line)
			if err != nil {
				log.Println(err)
				continue
			}
			client.recv <- msg
		}
	}()
	for {
		select {
		case msg := <-client.recv:
			fmt.Printf("[log:recv] %v\n", msg)
			for _, handler := range client.handlers {
				if handler.Accept(msg) {
					go handler.Handle(msg, client.send)
				}
			}
		case msg := <-client.send:
			fmt.Printf("[log:send] %v\n", msg)
			client.writer.PrintfLine("%v", msg)
		}
	}
}
