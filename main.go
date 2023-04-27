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
	loggerTime = 6000        // time in between server status check, in seconds.
	// defining shell code used to set terminal string colors.
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

var (
	branding = figure.NewColorFigure("JeanServ 23", "nancyj-fancy", "Blue", true)
	greeting = map[string]bool{
		"hello": true,
		"Hello": true,
		"hi":    true,
		"Hi":    true,
		"Hey":   true,
		"hey":   true,
	}
)

// Defines state for an individual connection.
type connection struct {
	messageHistory [][]byte  // Message Histort
	connectionId   string    // connection identifier. Just the connections Socket for now.
	Conn           net.Conn  // connection objct
	startTime      time.Time // Time of connection starting
	LastMessage    struct {
		Message []byte    // Actual Last Message
		Time    time.Time // time last message was sent
	}
}

type state struct {
	connections []connection
}

func (s state) ActiveConnections() int {
	return len(s.connections)
}

func (c connection) Init(conn net.Conn) {
	// initial connectionid is local Addr Socket
	c.connectionId = c.Conn.LocalAddr().String()
	c.Conn = conn
	c.startTime = time.Now()
}

func main() {
	for _, v := range branding.Slicify() {
		fmt.Println(colorWrap(Blue, v))
		time.Sleep(100 * time.Millisecond)
	}

	// creating channels for various modes of communication
	errc := make(chan error)   // error channel - Red text
	logc := make(chan string)  // general logging channel - Blue text
	sessc := make(chan string) // session specific channel - Yellow text
	var wg sync.WaitGroup
	wg.Add(2) // adding two goroutines
	go func() {
		eventHandler(sessc, errc, logc) // starting the Event Handler go routine
		wg.Done()                       // decrementing the counter when done
	}()
	go func() {
		connListener(sessc, errc, logc)
		wg.Done() // decrementing the counter when done
	}()
	wg.Wait() // waiting for all goroutines to finish
}

// Connection Listener accepts and passes connections off to Connection Handler
func connListener(sessc chan string, errc chan error, logc chan string) error {
	// Create Listener bound to socket.
	listener, err := net.Listen(netp, net.JoinHostPort(ip, port))
	if err != nil {
		log.Fatalf("Failed to create listener: %q", err)
	}

	// defer closing of listener until we escape from connection handler.
	defer func() { logc <- fmt.Sprintf("closing connectionHandler"); listener.Close() }()

	// logs what socket the listener is bound to.
	logc <- fmt.Sprintf("binding Listener on socket %v", listener.Addr().String())

	// handles incoming connectons.
	for {
		logc <- fmt.Sprintf("Starting new Connection handler")
		// routine will hang here until a connection is accepted.
		conn, err := listener.Accept()
		if err != nil {
			errc <- err
		}
		// hands accepted connection off to a connection handler go routine, and starts loop again.
		go connHandler(sessc, errc, logc, conn)
	}

}

// Connection Handler takes connections from listener, and processes read/writes
func connHandler(sessc chan string, errc chan error, logc chan string, c connection) {
	c.conn.Write([]byte(branding.ColorString()))
	// splits client address into IP Addr, and Port list.
	cAddr := strings.Split(c.connectionid, ":")
	cIp := cAddr[0]                                                // isolate Client IP
	cPort := cAddr[1]                                              // isolate Client Port.
	sessc <- fmt.Sprintf("starting new session:%v:%v", cIp, cPort) // logs start of new session
	buf := make([]byte, buffersize)                                // Create buffer
	// defering closing function until we eescape from session handler.
	defer func() {
		logc <- fmt.Sprintf("closing %v:%v session", cPort, cIp)
		c.conn.Close()
	}()
	for {
		// read from connection, into buffer.
		r, err := c.Read(buf)
		if err != nil {
			if err == io.EOF {
				sessc <- fmt.Sprintf("Received EOF from %v .", cPort)
				return
			} else {
				errc <- err
				return
			}
		}
		// Logs message received
		sessc <- fmt.Sprintf("(%v)Received message: "+colorWrap(Purple, "%v"), cPort, string(buf[:r-1]))

		// Decision tree for handling individual messages
		// Most functionality regarding handling user messages should be placed here.
		// In the future this may be it's own function.
		m := string(buf[:r-1])
		switch {
		case greeting[m]:
			func() {
				sessc <- fmt.Sprintf("(%v)sending: "+colorWrap(Gray, "Hi!"), cPort)
				c.Write([]byte(colorWrap(Purple, "Hi!\n")))
			}()
		case m == "ping":
			func() {
				sessc <- fmt.Sprintf("(%v)sending: "+colorWrap(Gray, "pong"), cPort)
				c.Write([]byte(colorWrap(Purple, "pong\n")))
			}()
		// They know what they did.
		case m == "pene holes":
			func() {
				sessc <- fmt.Sprintf("(%v)sending: A secret message.", cPort)
				c.Write([]byte(colorWrap(Red, "Get back to Rocket League. Sucks to Suck sucker.")))
			}()
		// Takes any message after "ascii:" and converts it to fancy ascii art.
		case strings.Split(m, ":")[0] == "ascii":
			sessc <- fmt.Sprintf("(%v)Returning Ascii Art.", cPort)
			c.Write([]byte(figure.NewColorFigure(strings.Split(m, ":")[1], "", "Blue", true).String() + "\n"))
		}
	}
}

// Event Handler handles events such as connection shutdowns and error logging.
func eventHandler(sessc <-chan string, errc <-chan error, logc <-chan string) {
	// Create a custom logger
	logger := log.New(os.Stdout, "", log.LstdFlags)
	mwrap := ""
	// defering exit routine for eventHandler.
	defer func() { logger.Printf(colorWrap(Red, "Exiting Error Logger")) }()
	for {
		// Use select to read from the channel with a timeout or a quit signal
		select {
		// Wraps sessc, logc, or errc channel messages in their individual colors.
		// log = blue, sess = yellow, and err = red, server status messages = green.
		case log := <-logc:
			mwrap = colorWrap(Blue, log)
		case sess := <-sessc:
			mwrap = colorWrap(Yellow, sess)
		case err := <-errc:
			mwrap = colorWrap(Red, err.Error())
		case <-time.After(loggerTime * time.Second):
			// Log a message that no errors have occurred for loggerTime seconds
			mwrap = colorWrap(Green, fmt.Sprintf("No errors for %v seconds", loggerTime))
		}
		// Logs messages, with appropriate colors based on channel.
		logger.Println(mwrap)
	}
}

// wraps strings in colors.
func colorWrap(c, m string) string {
	const Reset = "\033[0m"
	return c + m + Reset
}
