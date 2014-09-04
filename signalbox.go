/*
 * Copyright (c) Clinton Freeman 2014
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
 * associated documentation files (the "Software"), to deal in the Software without restriction,
 * including without limitation the rights to use, copy, modify, merge, publish, distribute,
 * sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or
 * substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
 * NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
 * DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const bufferSize int = 2048
const maxMessageSize int = 20480 // Ensure that inbound messages don't cause the signalbox to run out of memory.

type Peer struct {
	Id     string          // The unique identifier of the peer.
	socket *websocket.Conn // The socket for writing to the peer.
}

type Room struct {
	Room string // The unique name of the room (id).
}

type SignalBox struct {
	Peers        map[string]*Peer            // All the peers currently inside this signalbox.
	Rooms        map[string]*Room            // All the rooms currently inside this signalbox.
	RoomContains map[string]map[string]*Peer // All the peers currently inside a room.
	PeerIsIn     map[string]map[string]*Room // All the rooms a peer is currently inside.
}

type Message struct {
	msgSocket *websocket.Conn // The socket that the message was broadcast across.
	msgBody   string          // The body of the broadcasted message.
}

func messagePump(config Configuration, msg chan Message, ws *websocket.Conn) {
	//ws.SetReadDeadline(time.Now().Add(config.SocketTimeout * time.Second))
	//ws.SetWriteDeadline(time.Now().Add(config.SocketTimeout * time.Second))

	for {
		_, reader, err := ws.NextReader()

		if err != nil {
			// Unable to get reader from socket - probably closed, tell the signalbox.
			log.Printf("ERROR - messagePump: Can't read from %p, closing", ws)
			log.Print(err)
			msg <- Message{ws, "/close"}

			return
		}

		buffer := make([]byte, bufferSize)
		n, err := reader.Read(buffer)
		// Recieved content from socket - extend read deadline.
		//ws.SetReadDeadline(time.Now().Add(config.SocketTimeout * time.Second))

		socketContents := string(buffer[0:n])

		for err == nil && n == bufferSize && (len(socketContents)-bufferSize) < maxMessageSize {
			// filled the buffer - we might have more stuff in the message.
			n, err = reader.Read(buffer)
			// Recieved content from socket - extend read deadline.
			//ws.SetReadDeadline(time.Now().Add(config.SocketTimeout * time.Second))
			socketContents = socketContents + string(buffer[0:n])
		}

		if err != nil {
			log.Printf("ERROR - messagePump: Unable to read from websocket.")
			log.Print(err)
			continue
		}

		// Pump the new message into the signalbox.
		var message string
		json.Unmarshal([]byte(socketContents), &message)

		log.Printf("Recieved %s from %p", message, ws)

		msg <- Message{ws, message}
	}
}

func signalbox(config Configuration, msg chan Message) {
	s := SignalBox{make(map[string]*Peer),
		make(map[string]*Room),
		make(map[string]map[string]*Peer),
		make(map[string]map[string]*Room)}

	for {
		m := <-msg

		// Message matches a primus heartbeat message. Lightly massage the connection
		// with pong brand baby oil to keep everything running smoothly.
		if strings.HasPrefix(m.msgBody, "primus::ping::") {
			pong := fmt.Sprintf("primus::pong::%s", strings.Split(m.msgBody, "primus::ping::")[1])
			b, _ := json.Marshal(pong)

			m.msgSocket.WriteMessage(websocket.TextMessage, b)
			//m.msgSocket.SetWriteDeadline(time.Now().Add(config.SocketTimeout * time.Second))
			continue
		}

		action, messageBody, err := ParseMessage(m.msgBody)
		if err != nil {
			log.Printf("ERROR - signalbox: Unable to parse message.")
			log.Print(err)
			continue
		}

		s, err = action(messageBody, m.msgSocket, s)
		if err != nil {
			log.Printf("ERROR - signalbox: Unable to update state.")
			log.Print(err)
		}
	}
}

func main() {
	log.Printf("INFO - Started SignalBox\n")

	configFile := "signalbox.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, err := parseConfiguration(configFile)
	if err != nil {
		log.Printf("ERROR - main: Unable to parse config %s - using defaults.", err)
	}

	msg := make(chan Message)
	go signalbox(config, msg)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Upgrade the HTTP server connection to the WebSocket protocol.
		ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			log.Printf("ERROR - http.HandleFunc: %s", err)
			return
		}

		// Start pumping messages from this websocket into the signal box.
		go messagePump(config, msg, ws)
	})

	err = http.ListenAndServe(config.ListenAddress, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
