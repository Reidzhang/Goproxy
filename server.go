package main

import (
	"fmt"
	"os"
	"net"
	"time"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "8081"
)

func main ()  {
	listener, err := net.Listen("tcp", CONN_HOST+":"+CONN_PORT)
	checkError(err)
	code := 0
	defer func() {
		listener.Close()
		os.Exit(code)
	}()
	for {
		conn, err := listener.Accept()
		fmt.Println(time.Now(), " - Proxy is listening on " + conn.RemoteAddr().String())
		if err != nil {
			// Temporary handler for accept errors
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			code = 1
		}
		go handleRequest(conn)
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}