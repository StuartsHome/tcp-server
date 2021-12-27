package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

var connections = *flag.Int("conn", 4, "number of connections")

func main() {
	flag.Parse()

	// Create connections.
	var sliceOfConnections []net.Conn
	for i := 0; i < connections; i++ {
		conn, err := net.Dial("tcp", "localhost:4000")
		if err != nil {
			log.Fatal(err)
		}

		defer conn.Close()
		sliceOfConnections = append(sliceOfConnections, conn)
	}

	// A nine digit number.
	counter := 100000000
	for {
		for _, conn := range sliceOfConnections {
			clientReader := bufio.NewReader(os.Stdin)
			serverReader := bufio.NewReader(conn)
			_, err := clientReader.ReadBytes('\n')

			switch err {
			case nil:
				counter++
				if _, err = conn.Write([]byte(fmt.Sprintf("%d\n", counter))); err != nil {
					log.Printf("failed to send the client request: %v\n", err)
				}
			case io.EOF:
				log.Println("client closed the connection")
				return
			default:
				log.Printf("client error: %v\n", err)
				return
			}

			serverResponse, err := serverReader.ReadBytes('\n')
			switch err {
			case nil:
				log.Println(strings.TrimSpace(string(serverResponse)))
			case io.EOF:
				log.Println("server closed the connection")
				return
			default:
				log.Printf("server error: %v\n", err)
				return
			}

		}

	}
}
