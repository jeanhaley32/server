package main

// TODO(jeanhaley) - The following items need to be addressed:
//		- Rethink the "state handler" and how it is going to be used. maybe make it a "connection manager".
//		- Create a flow chart that shows the flow of how a client []byte is wrapped in a "msg", and routed through
//	          the system. I feel that at the moment there is no real rhyme or reason for this, and this needs to be
//		  codified.
// 		- enable the acceptance of CLI arguments to set the IP, Port, and Buffer size.
//		- re-factor code. Breakdown individual, independent functions into their own files.
//              - More Long term
//			- Break off individual components into seperate "micro-services" using GRPC for communication.
// 			- use Bubbtletea to create a CLI interface for the server.

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
	"github.com/google/uuid"
)

const (
	ip         = "127.0.0.1" // IP address
	netp       = "tcp"       // network protocol
	port       = "6000"      // Port to listen on
	buffersize = 1024        // Message Buffer size.
	loggerTime = 120         // time in between server status check, in seconds.
	Banner     = "JeanServ 23.6#"
)

// ___ Global Channel Variables ___
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

// Writes message to channel,
func (m MsgEnumType) WriteToChannel(a msg) {
	a.setTime()
	m.GetChannel() <- a
}

// Reads from Channel.
func (m MsgEnumType) ReadFromChannel() interface{} {
	return <-m.GetChannel()
}

// ___ End Global Channel Variables ___

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

// Returns color as a string
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
	branding     = figure.NewColorFigure(Banner, "nancyj-fancy", "Blue", true)
	currentstate state
)

// UID is used to identify individual connections.
type (
	UID       uint32
	timestamp string
)

// Define Enum for noClient UID
const (
	noClient UID = 0
)

// Defines state for an individual connection.
type connection struct {
	messageHistory []message // Message History
	connectionId   UID       // Unique Identifier for connection
	conn           net.Conn  // connection objct
	startTime      time.Time // Time of connection starting
}

// initializes connection object
func (c *connection) initConnection(conn net.Conn) {
	c.conn = conn
	c.startTime = time.Now()
	c.generateUid()
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
func (c connection) ConnectionId() UID {
	if c.connectionId == 0 {
		return UID(0)
	}
	return c.connectionId
}

// generates a unique connection id
func (c *connection) generateUid() {
	c.connectionId = UID(uuid.New().ID())
}

// Defines interface needed for connection handler
type ConnectionHandler interface {
	Read(buf *[]byte) (n int, err error)
	Write(buf *[]byte) (n int, err error)
	Close() error
	LastMessage() message
	AppendHistory(message)
	ConnectionId() UID
}

// State is used to derive over-all state of connections
type state struct {
	connections []*connection
}

func (s *state) ActiveConnections() int {
	return len(s.connections)
}

func (s *state) RemoveConnection(cn UID) {
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
type msg struct {
	destination UID         // destination of message
	source      UID         // source of message
	Id          UID         // Unique Identifier for message
	payload     []byte      // msg payload as a byte array
	t           time.Time   // Time Message was received
	msgType     MsgEnumType // Message type. Used to define message route.
}

// sets t to current time
func (m *msg) setTime() {
	m.t = time.Now()
}

// Returns message payload as a string
func (m msg) GetPayload() string {
	return string(m.payload)
}

// Returns Timestamp in Month/Day/Year Hour:Minute:Second format
func (m msg) Timestamp() timestamp {
	return t.Format("2006-01-02T15:04:05Z")

// return message Id
func (m msg) GetId() UID {
	return m.Id
}

// return message type
func (m msg) GetMsgType() MsgEnumType {
	return m.msgType
}

// Defines interface needed for message handler
type message interface {
	GetPayload() string
	Timestamp() timestamp
	GetId() UID
	GetMsgType() MsgEnumType
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
		// if we fail to create listener, we log the error and exit. This is a fatal error.
		log.Fatalf("Failed to create listener: %q", err)
	}
	// defer closing of listener until we escape from connection handler.
	defer func() {
		m := msg{
			payload: []byte("Listener Closed"),
		}
		if err := listener.Close(); err != nil {
			m.payload = []byte(fmt.Sprintf("Failed to Close Listener %q", err.Error()))
			m.msgType = Error
		}
		m.msgType.WriteToChannel(m)
	}()
	// logs what socket the listener is bound to.
	System.WriteToChannel(msg{payload: []byte(fmt.Sprintf("Listener bound to %v", listener.Addr()))})
	// handles incoming connectons.
	for {
		// logs that we are waiting for a connection.
		System.WriteToChannel(msg{payload: []byte("Waiting for connection"), msgType: System})
		// routine will hang here until a connection is accepted.
		conn, err := listener.Accept()
		if err != nil {
			// if we fail to accept connection, we log the error and continue.
			Error.WriteToChannel(msg{payload: []byte(fmt.Sprintf("Failed to accept connection: %q", err.Error()))})
		}
		// instantiating new connection
		newConn := connection{}
		// initializing connection
		newConn.initConnection(conn)
		// Add connection directory to state. This is used to track active connections.
		currentstate.AddConnection(&newConn)
		// kicking off connection handler
		go connHandler(&newConn)
	}

}

// Connection Handler takes connections from listener, and processes read/writes
func connHandler(conn ConnectionHandler) {
	branding := []byte(branding.ColorString()) // branding as a byte array
	conn.Write(&branding)                      // writes branding to connection
	Client.WriteToChannel(msg{
		payload: []byte(fmt.Sprintf("New connection from %v", conn.ConnectionId())),
		msgType: Client,
	}) // logs start of new session
	buf := make([]byte, buffersize) // Create buffer
	// defering closing function until we escape from session handler.
	defer func() {
		System.WriteToChannel(msg{payload: []byte(fmt.Sprintf("Closing connection from %v", conn.ConnectionId()))})
		// TODO(JeanHaley) Create a state handler(manager?) that can close this for us.
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
				Client.WriteToChannel(msg{
					payload: []byte(fmt.Sprintf("Received EOF from %v .", conn.ConnectionId())),
				})
				return
			} else {
				Error.WriteToChannel(msg{payload: []byte(err.Error())})
				return
			}
		}
		conn.AppendHistory(msg{payload: buf[:r-1], t: time.Now()}) // saves client messgae to message history

		// Logs message received
		Client.WriteToChannel(msg{
			payload: []byte(fmt.Sprintf("(%v)Received message: "+colorWrap(Purple, "%v"), conn.ConnectionId(), string(conn.LastMessage().GetPayload()))),
		})
		cmsg := []byte("")
		// Respond to message object
		switch {
		case string(conn.LastMessage().GetPayload()) == "ping":
			Client.WriteToChannel(msg{
				payload: []byte(fmt.Sprintf("(%v)sending: "+colorWrap(Gray, "pong"), conn.ConnectionId)),
			})
			cmsg = []byte(colorWrap(Purple, "pong\n"))
		// Catches "ascii:" and makes that ascii art.
		case strings.Split(string(conn.LastMessage().GetPayload()), ":")[0] == "ascii":
			Client.WriteToChannel(msg{
				payload: []byte(fmt.Sprintf("(%v)Returning Ascii Art.", port)),
			}) // Logs ascii art message to server
			cmsg = []byte(
				figure.NewColorFigure(
					strings.Split(string(conn.LastMessage().GetPayload()), ":")[1],
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
			mwrap = colorWrap(Blue, msg.GetPayload())
		case msg := <-sysChan:
			mwrap = colorWrap(Yellow, msg.GetPayload())
		case msg := <-errorChan:
			mwrap = colorWrap(Red, msg.GetPayload())
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
