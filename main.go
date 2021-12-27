package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StuartsHome/number-server/logger"
	"golang.org/x/net/netutil"
)

const (
	port              = 4000
	maxNumConnections = 5
)

type counter int32

type Server struct {
	// Mutex to sync writes to the map.
	mu sync.Mutex
	// Map of structs as more performant than map of bools
	duplicateNums map[string]struct{}

	// New unique numbers for the 10 second run.
	new counter
	// Duplicate numbers for the 10 second run.
	duplicates counter
	// Total numbers for the full application run.
	total counter
	// An empty chan to allow a graceful shutdown.
	quit chan interface{}
	// Waitgroup for every connection.
	wg sync.WaitGroup
	l  net.Listener
}

func NewServer() *Server {
	server := Server{
		quit:          make(chan interface{}),
		duplicateNums: make(map[string]struct{}),
	}
	lc := net.ListenConfig{}
	l, err := lc.Listen(context.TODO(), "tcp", fmt.Sprintf("localhost:%d", port))

	// Limit the number of connections to 5.
	netl := netutil.LimitListener(l, maxNumConnections)
	if err != nil {
		log.Fatal(err)
	}
	server.l = netl

	return &server
}

// Stop is a graceful shutdown of the server. Not as strong as a SIGKILL.
// This method is called when the buffer contains 'terminate'.
func (s *Server) stop() {
	// s.quit channel will close once all sent values have been received.
	close(s.quit)
	// Wait for all waitgroups to be done before closing the listener.
	s.wg.Wait()
	s.l.Close()
}

func (c *counter) inc() {
	atomic.AddInt32((*int32)(c), 1)
}

func (c *counter) get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

func (c *counter) reset() {
	// Swap the current integer with 0.
	atomic.SwapInt32((*int32)(c), 0)
}

func main() {
	logger.InitLogger(true)
	NewServer().run(true)
}

func (s *Server) run(reportOn bool) {
	if reportOn {
		// Create a report after every 10 seconds.
		go func() {
			for range time.Tick(10 * time.Second) {
				s.createReport()
			}
		}()
	}

	for {
		c, err := s.l.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				logger.Fatalf("this is a fatal. Let's close all connections")
			}
		} else {
			s.wg.Add(1)
			go s.handleConnections(c)
		}
	}

}

func (s *Server) handleConnections(c net.Conn) {

	// Single buffer per connection, comprised of 9 digits and a new line char.
	buffer := make([]byte, 10)
	for {
		select {
		case <-s.quit:
			s.wg.Done()
			return
		default:
			// Gather the length of the current buffer.
			readBuffer, err := c.Read(buffer)
			if err != nil {
				c.Close()
				s.wg.Done()
				return
			}

			// If the length of the current buffer is not 10, close this connection without comment.
			if readBuffer != 10 {
				c.Close()
				s.wg.Done()
				return
			}

			// Buffer 9 digit sequence must be only comprised of decimal digits.
			if currNum := buffer[:readBuffer-1]; !checkForCharacters(currNum) {
				if checkForTerminate(currNum) {
					c.Close()
					s.wg.Done() // Additional s.wg.Done to close the server.
					s.stop()
					return
				}
				c.Close()
				s.wg.Done()
				return
			}

			// Once all checks have been performed, assign the current buffer to a variable.
			currWord := string(buffer[:len(buffer)-1])

			// Check the variable value is unique.
			if _, ok := s.duplicateNums[currWord]; !ok {
				// Lock the mutex when writing to the map.
				s.mu.Lock()
				s.duplicateNums[currWord] = struct{}{}
				s.mu.Unlock()

				trimmedString := strings.TrimLeft(currWord, "0")

				logger.Logf("%s\n", trimmedString)
				c.Write([]byte(fmt.Sprintf("message received: %s\n", trimmedString)))

				// Increment the current total of unique numbers for this 10 sec run.
				s.new.inc()

				// Increment the total of unique numbers for this run of the application.
				s.total.inc()
			} else {
				// If the variable value is not unique, increment the duplicate total for this 10 sec run.
				s.duplicates.inc()
			}

		}

	}
}

func checkForTerminate(buffer []byte) bool {
	// Assume 'terminate' is always lowercase.
	return string(buffer) == "terminate"
}

// Verify the byte slice only contains integers.
func checkForCharacters(r []byte) bool {
	sep := 0

	for i := range r {
		if r[i] >= 48 && r[i] <= 57 {
			continue
		}
		if r[i] == 46 {
			if sep > 0 {
				return false
			}
			sep++
			continue
		}
		return false
	}
	return true
}

func (s *Server) createReport() {
	fmt.Printf("\nReceived %d unique numbers, ", s.new.get())
	fmt.Printf("%d duplicate(s). ", s.duplicates.get())
	fmt.Printf("Unique total: %d\n\n", s.total.get())

	// Reset counters (except total counter) after every 10 second cycle.
	s.new.reset()
	s.duplicates.reset()
}
