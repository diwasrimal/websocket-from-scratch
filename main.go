package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"github.com/diwasrimal/websocket-server/myhttp"
	"github.com/diwasrimal/websocket-server/myws"
)

func main() {
	httpPort := 3031
	httpLn := createTcpListener(httpPort)
	defer httpLn.Close()
	fmt.Printf("Http server listening on port %v...\n", httpPort)

	for true {
		conn, err := httpLn.Accept()
		if err != nil {
			fmt.Println("Error accepting conn:", conn)
			continue
		}
		id := randId()
		go handleHttpConn(conn, id)
	}
}

func createTcpListener(port int) net.Listener {
	addr := fmt.Sprintf(":%v", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error creating server")
		panic(err)
	}
	return ln
}

func handleHttpConn(conn net.Conn, id string) {
	defer func() {
		fmt.Printf("Closing connection for %v...\n", id)
		conn.Close()
	}()
	fmt.Printf("Handling http connection for %v...\n", id)
	buf := make([]byte, 2048)

	for true {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Received EOF from %v\n", id)
				break
			}
			fmt.Printf("Error: %v\n", err)
			continue
		}

		data := string(buf[:n])
		fmt.Printf("<-- http: %v --\n%v<--\n", id, data)
		req, err := myhttp.ParseRequest(data)
		if err != nil {
			fmt.Println("Error parsing http request:", err)
			continue
		}

		// Complete handshake and send the conn over to ws handler
		if req.Headers["Upgrade"] == "websocket" {
			sb := strings.Builder{}
			acceptKey := createWsAcceptKey(req.Headers["Sec-WebSocket-Key"])
			sb.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
			sb.WriteString("Upgrade: websocket\r\n")
			sb.WriteString("Connection: Upgrade\r\n")
			sb.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %v\r\n", acceptKey))
			sb.WriteString("\r\n")
			resp := sb.String()
			conn.Write([]byte(resp))
			fmt.Printf("-- http: %v -->\n%v-->\n", id, resp)
			handleWsConn(conn, id)
			return
		} else {
			// Can check routes and request methods, but just return simple response for now
			sb := strings.Builder{}
			sb.WriteString("HTTP/1.1 200 OK\r\n")
			sb.WriteString("Content-Type: text/html\r\n")
			sb.WriteString("Content-Length: 11\r\n")
			sb.WriteString("Connection: Close\r\n")
			sb.WriteString("\r\n")
			sb.WriteString("Hi client\r\n")
			resp := sb.String()
			conn.Write([]byte(resp))
			fmt.Printf("-- http: %v -->\n%v-->\n", id, resp)
		}
	}
}

func handleWsConn(conn net.Conn, id string) {
	fmt.Printf("Handling ws connection for %v...\n", id)
	var buf bytes.Buffer
	for true {
		// Parse and handle frame
		frame, err := myws.ParseWsBytes(conn)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Received EOF from %v, closing ws connection\n", id)
				break
			}
			fmt.Println("Error parsing bytes:", err)
		}

		// Decode the payload
		decodedPayload := frame.Payload
		for i := 0; i < len(decodedPayload); i++ {
			decodedPayload[i] = decodedPayload[i] ^ frame.MaskingKey[i%4]
		}

		// Send the same frame back to complete close handshake
		// Masking should be omitted and decoded payload should be sent
		if frame.Opcode == myws.OpcodeClose {
			fmt.Println("Received close opcode, sending frame back..")
			closeFrame := frame
			closeFrame.Masked = 0
			closeFrame.MaskingKey = []byte{}
			closeFrame.Payload = decodedPayload
			myws.SendWsByteFrame(conn, closeFrame)
			break
		} else if frame.Opcode == myws.OpcodePing {
			pingFrame := frame
			pingFrame.Opcode = myws.OpcodePong
			myws.SendWsByteFrame(conn, pingFrame)
		}

		buf.Write(decodedPayload)
		if frame.IsFinal() {
			data := buf.Bytes()
			// handle data ...
			fmt.Printf("<-- ws msg --\n%v--\n", string(data))
			buf = bytes.Buffer{}
		}
	}
}

func handleWsMessage(data []byte) {
	fmt.Printf("<-- ws msg --\n%v<--\n", string(data))
}

const randChars = "qwertyuiopasdfghjklzxcvbnm1234567890"

func randId() string {
	chars := make([]byte, 8)
	for i := range len(chars) {
		chars[i] = randChars[rand.Intn(len(randChars))]
	}
	return string(chars)
}

const wsMagicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func createWsAcceptKey(requestKey string) string {
	sha1bytes := sha1.Sum([]byte(requestKey + wsMagicString))
	return base64.StdEncoding.EncodeToString(sha1bytes[:])
}
