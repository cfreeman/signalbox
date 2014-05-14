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
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"strings"
	"unicode/utf8"
)

type messageFn func(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error)

func announce(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source, destination, err := ParsePeerAndRoom(message)
	if err != nil {
		return state, err
	}

	peer, exists := state.Peers[source.Id]
	if !exists {
		log.Printf("Adding Peer: %s\n", source.Id)
		state.Peers[source.Id] = new(Peer)
		state.Peers[source.Id].Id = source.Id
		state.Peers[source.Id].socket = sourceSocket // Inject a reference to the websocket within the new peer.
		peer = state.Peers[source.Id]
	}

	room, exists := state.Rooms[destination.Room]
	if !exists {
		log.Printf("Adding Room: %s\n", destination.Room)
		state.Rooms[destination.Room] = new(Room)
		state.Rooms[destination.Room].Room = destination.Room
		room = state.Rooms[destination.Room]
	}

	if state.PeerIsIn[peer.Id] == nil {
		state.PeerIsIn[peer.Id] = make(map[string]*Room)
	}
	state.PeerIsIn[peer.Id][room.Room] = room

	if state.RoomContains[room.Room] == nil {
		state.RoomContains[room.Room] = make(map[string]*Peer)
	}
	state.RoomContains[room.Room][peer.Id] = peer

	// Annouce the arrival to all the peers currently in the room.
	for _, p := range state.RoomContains[room.Room] {
		if p.Id != peer.Id && p.socket != nil {
			writeMessage(p.socket, message)
		}
	}

	return state, nil
}

func leave(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source, destination, err := ParsePeerAndRoom(message)
	if err != nil {
		return state, err
	}

	peer, exists := state.Peers[source.Id]
	if !exists {
		return state, errors.New(fmt.Sprintf("Unable to leave, peer %s doesn't exist", source.Id))
	}

	room, exists := state.Rooms[destination.Room]
	if !exists {
		return state, errors.New(fmt.Sprintf("Unable to leave, room %s doesn't exist", destination.Room))
	}

	delete(state.PeerIsIn[peer.Id], destination.Room)
	if len(state.PeerIsIn[peer.Id]) == 0 {
		delete(state.Peers, peer.Id)
		delete(state.PeerIsIn, peer.Id)
	}

	delete(state.RoomContains[destination.Room], peer.Id)
	if len(state.RoomContains[destination.Room]) == 0 {
		delete(state.Rooms, destination.Room)
		delete(state.RoomContains, destination.Room)
	} else {
		// Broadcast the departure to everyone else still in the room
		for _, p := range state.RoomContains[room.Room] {
			if p.socket != nil {
				writeMessage(p.socket, message)
			}
		}
	}

	return state, nil
}

func to(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	if len(message) < 3 {
		return state, errors.New("Not enouth parts to personalised message")
	}

	d, exists := state.Peers[message[1]]
	if !exists {
		return state, nil
	}

	if d.socket != nil {
		writeMessage(d.socket, message)
	}

	return state, nil
}

func writeMessage(ws *websocket.Conn, message []string) {
	b, err := json.Marshal(strings.Join(message, "|"))
	if err == nil {
		log.Printf("Writing %s\n", string(b))
		ws.WriteMessage(websocket.TextMessage, b)
	}
}

func custom(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source := Peer{}
	if len(message) < 2 {
		return state, errors.New("Not enough parts to custom message")
	}

	err = json.Unmarshal([]byte(message[1]), &source)
	if err != nil {
		return state, err
	}

	peer, exists := state.Peers[source.Id]
	if !exists {
		return state, nil
	}

	for _, r := range state.PeerIsIn[peer.Id] {
		for _, p := range state.RoomContains[r.Room] {
			if p.Id != peer.Id && p.socket != nil {
				p.socket.WriteMessage(websocket.TextMessage, []byte(strings.Join(message, "|")))
			}
		}
	}

	return state, nil
}

func ParsePeerAndRoom(message []string) (source Peer, destination Room, err error) {
	if len(message) < 3 {
		return Peer{}, Room{}, errors.New("Not enough parts in the message body to parse peer and room.")
	}

	err = json.Unmarshal([]byte(message[1]), &source)
	if err != nil {
		log.Print(err)
		return Peer{}, Room{}, err
	}

	err = json.Unmarshal([]byte(message[2]), &destination)
	if err != nil {
		return Peer{}, Room{}, err
	}

	return source, destination, nil
}

func ParseMessage(message string) (action messageFn, messageBody []string, err error) {
	// All messages are text (utf-8 encoded at present)
	if !utf8.Valid([]byte(message)) {
		return nil, nil, errors.New("Message is not utf-8 encoded")
	}

	parts := strings.Split(message, "|")

	switch parts[0] {
	case "/announce":
		log.Printf("Announce.")
		return announce, parts, nil

	case "/leave":
		log.Printf("Leave.")
		return leave, parts, nil

	case "/to":
		log.Printf("To.")
		return to, parts, nil

	default:
		log.Printf("Custom.")
		return custom, parts, nil
	}

}
