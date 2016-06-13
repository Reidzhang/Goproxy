package main

import (
	"net"
	"bufio"
	"strings"
	"strconv"
	"time"
	"fmt"
	"io"
)
type header struct {
	destPort int
	firstLine string
	contentLen int64
	methods []string
	host string
	isConnect bool
	version string
}

// Constants
const CONNECT_METHOD string = "CONNECT"
const HTTPS string = "https"
const CONTION string = "Connection"
const PCONTION string = "Proxy-connection"
const CL string = "Content-Length"
const HT string = "Host"
const CR_LT string = "\t\n"

func handleRequest(conn net.Conn) {
	// close this connection
	defer conn.Close()

	var reqHeader header
	reqHeader.parseHeader(conn)
	timeout := time.Duration(120) * time.Second
	serConn, err := net.DialTimeout("tcp", header.host+":"+header.destPort, timeout)
	defer serConn.Close()
	if err != nil {
		// check error and deal with it
		fmt.Println(err)
		if reqHeader.isConnect {
			// write bad gateway back to client
			tmp := strings.Join([]string{reqHeader.version, "502 Bad Gateway\t\n\t\n"}, " ")
			s := []byte(tmp)
			conn.Write(s)
		}
		return
	}

	if reqHeader.isConnect {
		// this is CONNECT-TUNNELING
		tmp := strings.Join([]string{reqHeader.version, "200 OK\t\n\t\n"}, " ")
		s := []byte(tmp)
		conn.Write(s)
		Pipe(conn, serConn)
	} else {
		s := []byte(reqHeader.firstLine + CR_LT)
		serConn.Write(s)
		for i := 0; i < len(reqHeader.methods); i++ {
			s = []byte(reqHeader.methods[i] + CR_LT)
			serConn.Write(s)
		}
		// finish writing
		serConn.Write([]byte(CR_LT))
		b := make([]byte, 1024)
		for {
			n, err := serConn.Read(b)

			if err != nil {
				if err == io.EOF {
					fmt.Println("Exiting from the thread")
				}
				return
			}
			conn.Write(b[:n])
		}
	}

}



// parsing the request header
func (ano *header) parseHeader(conn net.Conn) {
	// Close the connection when you're done with it.
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Split(bufio.ScanLines)

	// Get the first line of the header
	var tmp_line string
	var isConnect bool
	tmp_line = scanner.Text()
	isConnect = checkConnect(tmp_line, ano)
	// set is Connect field in the header
	ano.isConnect = isConnect
	// Send a response back to person contacting us.
	for scanner.Scan() {
		tmp := scanner.Text()
		if len(tmp) == 0 {
			// the end of the header
			break
		} else {
			llist := strings.Split(tmp, ": ")
			method := llist[0]
			// check the method
			// if it is NON-Connect, change it to close
			if (strings.Compare(method, CONTION) == 0 ||
				strings.Compare(method, PCONTION) == 0) && !isConnect {
				// change it close
				llist[1] = "close"
			}

			if strings.Compare(method, CL) == 0 {
				// parsing the content length
				if s, err := strconv.ParseInt(llist[1], 10, 64); err == nil {
					ano.contentLen = s
				}
			}

			if strings.EqualFold(method, HT){
				ano.host = strings.TrimSpace(llist[1])
				tmp_list := strings.Split(llist[1], ":")
				if len(tmp_list) != 0 {
					if s, err := strconv.ParseInt(llist[1], 10, 64); err == nil {
						ano.destPort = s
					}
				}
			}
			s := []string{llist[0], ": ", llist[1], "\t\n"}
			ano.methods = append(ano.methods, strings.Join(s, ""))
		}
	}
}

// Check whether the connection is a NON-Connect or that
// is a CONNECT
func checkConnect(firstLine string, reqHeader *header) bool {
	words := strings.Fields(firstLine)
	res := false
	if strings.Compare(words[0], CONNECT_METHOD) {
		res = true
	} else {
		// it is a NON-Connect
		// lower to http version 1.0
		words[2] = "HTTP/1.0"
	}
	// assign a port
	if (strings.Contains(words[1], HTTPS)) {
		reqHeader.destPort = 443
	} else {
		reqHeader.destPort = 80
	}
	// joint the first line of request
	reqHeader.firstLine = strings.Join(words, " ")
	// give the version to header struct
	reqHeader.version = strings.TrimSpace(words[2])
	return res
}


// Pipe creates a full-duplex pipe between the two sockets and transfers data from one to the other.
func Pipe(conn1 net.Conn, conn2 net.Conn) {
	chan1 := chanFromConn(conn1)
	chan2 := chanFromConn(conn2)

	for {
		select {
		case b1 := <-chan1:
			if b1 == nil {
				return
			} else {
				conn2.Write(b1)
			}
		case b2 := <-chan2:
			if b2 == nil {
				return
			} else {
				conn1.Write(b2)
			}
		}
	}
}

// chanFromConn creates a channel from a Conn object, and sends everything it
//  Read()s from the socket to the channel.
func chanFromConn(conn net.Conn) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := conn.Read(b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()

	return c
}