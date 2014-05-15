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
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

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

func messagePump(msg chan Message, ws *websocket.Conn) {
	for {
		_, reader, err := ws.NextReader()
		if err != nil {
			log.Printf("messagePump error: Unable to get next reader.")
			log.Print(err)

			// TODO: Teardown peer from signalbox.
			// TODO: Need to handle websocket pings to see what is alive.
			// TODO: Configuration file.
			// TODO: Test when message is longer than will fit in buffer.
			return
		}

		buffer := make([]byte, 2048)
		n, err := reader.Read(buffer)
		if err != nil {
			log.Printf("messagePump error: Unable to read from websocket.")
			log.Print(err)
			continue
		}

		// Pump the new message into the signalbox.
		var message string
		json.Unmarshal(buffer[0:n], &message)
		msg <- Message{ws, message}
	}
}

func signalbox(msg chan Message) {
	s := SignalBox{make(map[string]*Peer),
		make(map[string]*Room),
		make(map[string]map[string]*Peer),
		make(map[string]map[string]*Room)}

	for {
		m := <-msg

		log.Printf("Message: %s\n", m.msgBody)
		action, messageBody, err := ParseMessage(m.msgBody)
		if err != nil {
			log.Printf("signalbox error: Unable to parse message.")
			log.Print(err)
			continue
		}

		s, err = action(messageBody, m.msgSocket, s)
		if err != nil {
			log.Printf("signalbox error: Unable to update state.")
			log.Print(err)
		}
	}

	// TODO: Leave messages on socket closes.
}

func main() {
	log.Printf("Started SignalBox\n")

	msg := make(chan Message)
	go signalbox(msg)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Upgrade the HTTP server connection to the WebSocket protocol.
		ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			log.Println(err)
			return
		}

		// Start pumping messages from this websocket into the signal box.
		go messagePump(msg, ws)
	})

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
