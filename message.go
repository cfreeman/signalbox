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
		fmt.Printf("Adding Peer: %s\n", source.Id)
		state.Peers[source.Id] = new(Peer)
		state.Peers[source.Id].Id = source.Id
		state.Peers[source.Id].socket = sourceSocket // Inject a reference to the websocket within the new peer.
		peer = state.Peers[source.Id]
	}

	room, exists := state.Rooms[destination.Room]
	if !exists {
		fmt.Printf("Adding Room: %s\n", destination.Room)
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
		if p.Id != peer.Id {
			if p.socket != nil {
				p.socket.WriteMessage(websocket.TextMessage, []byte(strings.Join(message, "|")))
			}
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
				p.socket.WriteMessage(websocket.TextMessage, []byte(strings.Join(message, "|")))
			}
		}
	}

	return state, nil
}

func to(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	fmt.Printf("to message\n")
	return state, nil
}

func custom(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	fmt.Printf("custom message\n")
	return state, nil
}

func ParsePeerAndRoom(messageBody []string) (source Peer, destination Room, err error) {
	if len(messageBody) < 3 {
		return Peer{}, Room{}, errors.New("Not enough parts in the message body to parse peer and room.")
	}

	err = json.Unmarshal([]byte(messageBody[1]), &source)
	if err != nil {
		return Peer{}, Room{}, err
	}

	err = json.Unmarshal([]byte(messageBody[2]), &destination)
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

	// Message parts are delimited by a pipe (|) character
	parts := strings.Split(message, "|")

	switch parts[0] {
	case "/announce":
		return announce, parts, nil

	case "/leave":
		return leave, parts, nil

	case "/to":
		return to, parts, nil

	default:
		return custom, parts, nil
	}
}
