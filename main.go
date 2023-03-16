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
)

const (
	ip = "127.0.0.1"
	// network protocol
	netp       = "tcp"
	port       = "3000"
	buffersize = 1024
	loggerTime = 30
	Reset      = "\033[0m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Purple     = "\033[35m"
	Cyan       = "\033[36m"
	Gray       = "\033[37m"
	White      = "\033[97m"
)

func main() {
	// creating channel to communicate errors in goroutines.
	errc := make(chan error)
	// Creating channel to log abitary messages.
	logc := make(chan string)
	// Create Channel for session messages.
	sessc := make(chan string)

	var wg sync.WaitGroup
	wg.Add(2) // adding two goroutines
	go func() {
		// Print a message that the function has started
		eventHandler(sessc, errc, logc)
		wg.Done() // decrementing the counter when done
	}()
	go func() {
		connectionsHandler(sessc, errc, logc)
		wg.Done() // decrementing the counter when done
	}()
	wg.Wait() // waiting for all goroutines to finish
}

// Connections handler spins off individual connections into their own session GoRoutines.
func connectionsHandler(sessc chan string, errc chan error, logc chan string) error {
	// Create Listener bound to socket.
	listener, err := net.Listen(netp, net.JoinHostPort(ip, port))

	if err != nil {
		log.Fatalf("Failed to create listener: %q", err)
	}

	defer listener.Close()

	logc <- fmt.Sprintf("binding Listener on socket %v", listener.Addr().String())

	for {
		logc <- fmt.Sprintf("Starting new Connection handler")
		conn, err := listener.Accept()
		if err != nil {
			errc <- err
		}
		defer func() {
			conn.Close()
			errc <- nil
			logc <- fmt.Sprintf("Closing server")
		}()
		go sessionHandler(sessc, errc, logc, conn)
	}

}

// Session Handler handles individual tcp connections.
func sessionHandler(sessc chan string, errc chan error, logc chan string, c net.Conn) {
	// split client address into IP Addr, and Port.
	cAddr := strings.Split(c.RemoteAddr().String(), ":")
	cIp := cAddr[0]                                                // isolate Client IP
	cPort := cAddr[1]                                              // isolate Client Port.
	sessc <- fmt.Sprintf("starting new session:%v:%v", cIp, cPort) // log begining of new session
	buf := make([]byte, buffersize)
	for {
		r, err := c.Read(buf)
		if err != nil {
			if err != io.EOF {
				sessc <- fmt.Sprintf("Client on port %v terminated connection.", cPort)
				c.Close()
				return
			} else {
				errc <- err
				c.Close()
			}
		}
		if string(buf[:r-1]) == "ping" {
			logc <- fmt.Sprintf("(%v)sending: "+Gray+"pong"+Reset, cPort)
			c.Write([]byte(Purple + fmt.Sprintf("pong\n") + Reset))
		}
		logc <- fmt.Sprintf("(%v)Received message: "+Purple+
			"%v"+Reset, cPort, string(buf[:r-1]))
	}
}

// Event Handler handles events such as connection shutdowns and error logging.
func eventHandler(sessc <-chan string, errc <-chan error, logc <-chan string) {
	// Create a custom logger
	logger := log.New(os.Stdout, "", log.LstdFlags)
	mwrap := ""
	// Use defer to print a message that the function has finished
	defer func() { logger.Printf(Red + "Finished Error Logger"); return }()
	for {
		// Use select to read from the channel with a timeout or a quit signal
		select {
		// Wraps sessc, logc, or errc channel messages in their individual colors.
		// log = blue, sess = yellow, and err = red, server status messages = green.
		case log := <-logc:
			mwrap = fmt.Sprintf(Blue + log + Reset)
		case sess := <-sessc:
			mwrap = fmt.Sprintf(Yellow + sess + Reset)
		case err := <-errc:
			mwrap = fmt.Sprintf(Red + err.Error() + Reset)
		case <-time.After(loggerTime * time.Second):
			// Log a message that no errors have occurred for 10 seconds
			mwrap = fmt.Sprintf(Green+"No errors for %v seconds"+Reset, loggerTime)
		}
		// prints appropriate message.
		logger.Println(mwrap)
	}
}
