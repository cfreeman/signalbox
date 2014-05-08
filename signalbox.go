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
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

type Peer struct {
	Id     string          // The unique identifier of the peer.
	socket *websocket.Conn //The socket for writing to the peer.
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

func main() {
	fmt.Printf("SignalBox Started!\n")

	s := SignalBox{make(map[string]*Peer),
		make(map[string]*Room),
		make(map[string]map[string]*Peer),
		make(map[string]map[string]*Room)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Upgrade the HTTP server connection to the WebSocket protocol.
		ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			fmt.Println(err)
			return
		}

		// TODO: Read messages from socket continuously.
		mt, message, err := ws.ReadMessage()
		switch mt {
		case websocket.TextMessage:
			fmt.Printf("Message: %s\n", message)
			action, messageBody, err := ParseMessage(string(message))
			if err != nil {
				fmt.Printf("Unable to parse message: %s!\n", message)
			}

			s, err = action(messageBody, ws, s)
			if err != nil {
				fmt.Printf("Error unable to alter signal box")
			}
		}
	})

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
