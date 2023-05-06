package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/common-nighthawk/go-figure"
)

const (
	ip         = "127.0.0.1" // IP address
	netp       = "tcp"       // network protocol
	port       = "6000"      // Port to listen on
	buffersize = 1024        // Message Buffer size.
	loggerTime = 120         // time in between server status check, in seconds.
)

// create a channel type with blank interface
type ch chan message

// Define our three global log channels
//
//	client - Logs from individual connections.
//	error  - Error logs.
//	sys		- System logs.
var (
	clientChan, errorChan, sysChan ch
)

// Defining type used to define a message route and purpose
type MsgEnumType int64

const (
	Client MsgEnumType = iota
	Error
	System
)

// Returns msg type string
func (m MsgEnumType) Type() string {
	switch m {
	case Client:
		return "cli"
	case Error:
		return "err"
	case System:
		return "sys"
	}
	return "sys"
}

func (m MsgEnumType) GetChannel() ch {
	switch m {
	case Client:
		return clientChan
	case Error:
		return errorChan
	case System:
		return sysChan
	}
	return sysChan
}

// TODO(jeanhaley) - Wrap incoming messages in a struct that includes the message type.
// Writes value to the appropriate channel. Takes in value as a string and converts to []byte.
func (m MsgEnumType) WriteToChannel(value string) {
	msg := message{
		msg:     []byte(value),
		t:       time.Now(),
		msgType: m,
	}
	m.GetChannel() <- msg
}

// Reads from Channel.
func (m MsgEnumType) ReadFromChannel() interface{} {
	return <-m.GetChannel()
}

// defining Color Enums
type Color int64

const (
	Red Color = iota
	Green
	Yellow
	Blue
	Purple
	Cyan
	Gray
	White
)

func (c Color) Color() string {
	switch c {
	case Red:
		return "\033[31m"
	case Green:
		return "\033[32m"
	case Yellow:
		return "\033[33m"
	case Blue:
		return "\033[34m"
	case Purple:
		return "\033[35m"
	case Cyan:
		return "\033[36m"
	case Gray:
		return "\033[37m"
	case White:
		return "\033[97m"
	}
	return ""
}

var (
	branding     = figure.NewColorFigure("JeanServ 23.6 #pending update", "nancyj-fancy", "Blue", true)
	currentstate state
)

// Defines state for an individual connection.
type connection struct {
	messageHistory []message // Message History
	connectionId   string    // connection identifier. Just the connections Socket for now.
	conn           net.Conn  // connection objct
	startTime      time.Time // Time of connection starting
}

// Returns last message bundled in messageHistory
func (c connection) LastMessage() message {
	return c.messageHistory[len(c.messageHistory)-1]
}

// Exposes net.Conn Read method
func (c connection) Read(buf *[]byte) (int, error) {
	return c.conn.Read(*buf)
}

// Exposes net.Conn Write method
func (c *connection) Write(buf *[]byte) (int, error) {
	return c.conn.Write(*buf)
}

// Exposes net.Conn Close method
func (c *connection) Close() error {
	return c.conn.Close()
}

// Appends message to message history
func (c *connection) AppendHistory(m message) {
	c.messageHistory = append(c.messageHistory, m)
}

// exposes ConnectionId
func (c connection) ConnectionId() string {
	return c.connectionId
}

// Defines interface needed for connection handler
type ConnectionHandler interface {
	Read(buf *[]byte) (n int, err error)
	Write(buf *[]byte) (n int, err error)
	Close() error
	LastMessage() message
	AppendHistory(message)
	ConnectionId() string
}

// State is used to derive over-all state of connections
type state struct {
	connections []*connection
}

func (s *state) ActiveConnections() int {
	return len(s.connections)
}

func (s *state) RemoveConnection(cn string) {
	for i, c := range s.connections {
		if c.connectionId == cn {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
		}
	}
}

func (s *state) AddConnection(c *connection) {
	s.connections = append(s.connections, c)
}

// Message "object"
// individual message received from connection.
type message struct {
	msg     []byte      // Single message as a list of bytes
	t       time.Time   // Time Message was received
	msgType MsgEnumType // Message type. Used to define message route.
}

// Returns message payload as a string
func (m message) String() string {
	return string(m.msg)
}

// // return 'unix time' timestamp from message receipt
func (m message) Timestamp() int64 {
	return m.t.Unix()
}

func main() {
	// instantiating global channels.
	clientChan = make(chan message)
	errorChan = make(chan message)
	sysChan = make(chan message)
	for _, v := range branding.Slicify() {
		fmt.Println(colorWrap(Blue, v))
		time.Sleep(100 * time.Millisecond)
	}
	var wg sync.WaitGroup
	wg.Add(2) // adding two goroutines
	go func() {
		eventHandler() // starting the Event Handler go routine
		wg.Done()      // decrementing the counter when done
	}()
	go func() {
		connListener()
		wg.Done() // decrementing the counter when done
	}()
	wg.Wait() // waiting for all goroutines to finish
}

// Connection Listener accepts and passes connections off to Connection Handler
func connListener() error {
	// Create Listener bound to socket.
	listener, err := net.Listen(netp, net.JoinHostPort(ip, port))
	if err != nil {
		log.Fatalf("Failed to create listener: %q", err)
	}

	// defer closing of listener until we escape from connection handler.
	defer func() {
		System.WriteToChannel("closing Listener")
		if err := listener.Close(); err != nil {
			Error.WriteToChannel(err.Error())
		}
	}()

	// logs what socket the listener is bound to.
	System.WriteToChannel(fmt.Sprintf("binding Listener on socket %v", listener.Addr().String()))
	// handles incoming connectons.
	for {
		System.WriteToChannel("Starting new Connection handler")
		// routine will hang here until a connection is accepted.
		conn, err := listener.Accept()
		if err != nil {
			Error.WriteToChannel(err.Error())
		}
		newConn := connection{
			conn:         conn,
			connectionId: conn.RemoteAddr().String(),
			startTime:    time.Now(),
		}
		currentstate.AddConnection(&newConn)
		// Kicking off a fresh connection handler, and passing connection to it.
		go connHandler(&newConn)
	}

}

// Connection Handler takes connections from listener, and processes read/writes
func connHandler(conn ConnectionHandler) {
	branding := []byte(branding.ColorString())
	conn.Write(&branding)
	Client.WriteToChannel(fmt.Sprintf("starting new session:%v", conn.ConnectionId())) // logs start of new session
	buf := make([]byte, buffersize)                                                    // Create buffer
	// defering closing function until we escape from session handler.
	defer func() {
		System.WriteToChannel(fmt.Sprintf("closing %v session", conn.ConnectionId()))
		// TODO(JeanHaley) Create a state handler that can close this for us.
		// we should send a signal through an explicit connection channel to
		// the state handler that then tells it to close this connection and
		// pops it from the list of active connections.
		currentstate.RemoveConnection(conn.ConnectionId())
		conn.Close()
	}()
	for {
		r, err := conn.Read(&buf) // Write Client message to buffer
		if err != nil {
			if err == io.EOF {
				Client.WriteToChannel(fmt.Sprintf("Received EOF from %v .", conn.ConnectionId()))
				return
			} else {
				Error.WriteToChannel(err.Error())
				return
			}
		}
		conn.AppendHistory(message{msg: buf[:r-1], t: time.Now()}) // saves client messgae to message history

		// Logs message received
		Client.WriteToChannel(fmt.Sprintf("(%v)Received message: "+colorWrap(Purple, "%v"), conn.ConnectionId(), string(conn.LastMessage().msg)))
		cmsg := []byte("")
		// Respond to message object
		switch {
		case string(conn.LastMessage().msg) == "ping":
			Client.WriteToChannel(fmt.Sprintf("(%v)sending: "+colorWrap(Gray, "pong"), conn.ConnectionId))
			cmsg = []byte(colorWrap(Purple, "pong\n"))
		// Catches "ascii:" and makes that ascii art.
		case strings.Split(string(conn.LastMessage().msg), ":")[0] == "ascii":
			Client.WriteToChannel(fmt.Sprintf("(%v)Returning Ascii Art.", port)) // Logs ascii art message to server
			cmsg = []byte(
				figure.NewColorFigure(
					strings.Split(string(conn.LastMessage().msg), ":")[1],
					"", "Blue", true).String() +
					"\n") // Sends an Ascii art version of user's message back to user.
		default:
			cmsg = []byte("Message Received")
		}
		cmsg = append(cmsg, []byte("\n")...)
		conn.Write(&cmsg)
	}
}

// Event Handler handles events such as connection shutdowns and error logging.
func eventHandler() {
	// Create a custom logger
	logger := log.New(os.Stdout, "", log.LstdFlags)
	mwrap := ""
	// defering exit routine for eventHandler.
	defer func() { logger.Printf(colorWrap(Red, "Exiting Error Logger")) }()
	for {
		select {
		case msg := <-clientChan:
			mwrap = colorWrap(Blue, msg.String())
		case msg := <-sysChan:
			mwrap = colorWrap(Yellow, msg.String())
		case msg := <-errorChan:
			mwrap = colorWrap(Red, msg.String())
		case <-time.After(loggerTime * time.Second):
			// Log a message that no errors have occurred for loggerTime seconds
			mwrap = colorWrap(Green, fmt.Sprintf(
				"No errors for %v seconds, %v active connections",
				loggerTime,
				currentstate.ActiveConnections()))
		}
		// Logs messages, with appropriate colors based on channel.
		logger.Println(mwrap)
	}
}

// wraps strings in colors.
func colorWrap(c Color, m string) string {
	const Reset = "\033[0m"
	return c.Color() + m + Reset
}
