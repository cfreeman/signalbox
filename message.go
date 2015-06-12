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
		log.Printf("INFO - Adding Peer: %s\n", source.Id)
		state.Peers[source.Id] = new(Peer)
		state.Peers[source.Id].Id = source.Id
		state.Peers[source.Id].socket = sourceSocket // Inject a reference to the websocket within the new peer.
		peer = state.Peers[source.Id]
	}

	room, exists := state.Rooms[destination.Room]
	if !exists {
		log.Printf("INFO - Adding Room: %s\n", destination.Room)
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

	// Report back to the announcer the number of peers in the room.
	members := fmt.Sprintf("{\"memberCount\":%d}", len(state.RoomContains[room.Room]))
	err = writeMessage(sourceSocket, []string{"/roominfo", members})

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

	return removePeer(peer, room, message, state)
}

func closePeer(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source := findPeerBySocket(sourceSocket, state)
	if source == nil {
		return state, errors.New("Unable to close - no Peer matching socket.")
	}

	// Announce to everyone that the peer belonging to sourceSocket
	// has closed and bailed out of their rooms.
	for _, r := range state.PeerIsIn[source.Id] {
		for _, p := range state.RoomContains[r.Room] {
			if p.Id != source.Id && p.socket != nil {
				rm := fmt.Sprintf("{\"room\":\"%s\"}", r.Room)

				state, err = removePeer(source, r, []string{"/leave", source.Id, rm}, state)
				if err != nil {
					return state, err
				}
			}
		}
	}

	// Make sure the socket is closed from this end.
	err = sourceSocket.Close()

	return state, err
}

func removePeer(source *Peer, destination *Room, message []string, state SignalBox) (newState SignalBox, err error) {
	delete(state.PeerIsIn[source.Id], destination.Room)
	if len(state.PeerIsIn[source.Id]) == 0 {
		log.Printf("INFO - Removing Peer: %s\n", source.Id)
		delete(state.Peers, source.Id)
		delete(state.PeerIsIn, source.Id)
	}

	delete(state.RoomContains[destination.Room], source.Id)
	if len(state.RoomContains[destination.Room]) == 0 {
		log.Printf("INFO - Removing Room: %s\n", destination.Room)
		delete(state.Rooms, destination.Room)
		delete(state.RoomContains, destination.Room)
	} else {
		// Broadcast the departure to everyone else still in the room
		for _, p := range state.RoomContains[destination.Room] {
			if p.socket != nil && err == nil {
				err = writeMessage(p.socket, message)
			}
		}
	}

	return state, err
}

func to(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	if len(message) < 3 {
		return state, errors.New("Not enouth parts for personalised 'to' message")
	}

	d, exists := state.Peers[message[1]]
	if !exists {
		return state, nil
	}

	if d.socket != nil {
		err = writeMessage(d.socket, message)
	}

	return state, err
}

func writeMessage(ws *websocket.Conn, message []string) error {
	b := strings.Join(message, "|")
	if ws != nil {
		log.Printf("INFO - Writing %s to %p", b, ws)
		return ws.WriteMessage(websocket.TextMessage, []byte(b))
	}

	return nil
}

func custom(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	if len(message) < 2 {
		return state, errors.New("Not enough parts to custom message")
	}

	source := Peer{message[1], nil}

	peer, exists := state.Peers[source.Id]
	if !exists {
		return state, nil
	}

	for _, r := range state.PeerIsIn[peer.Id] {
		for _, p := range state.RoomContains[r.Room] {
			if p.Id != peer.Id && p.socket != nil && err == nil {
				err = writeMessage(p.socket, message)
			}
		}
	}

	return state, err
}

func ignore(message []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {
	return state, nil
}

func findPeerBySocket(sourceSocket *websocket.Conn, state SignalBox) *Peer {
	for _, p := range state.Peers {
		if p.socket == sourceSocket {
			return p
		}
	}

	return nil
}

func ParsePeerAndRoom(message []string) (source Peer, destination Room, err error) {
	if len(message) < 3 {
		return Peer{}, Room{}, errors.New("Not enough parts in the message body to parse peer and room.")
	}

	err = json.Unmarshal([]byte(message[2]), &destination)
	if err != nil {
		return Peer{}, Room{}, err
	}

	return Peer{message[1], nil}, destination, nil
}

func ParseMessage(message string) (action messageFn, messageBody []string, err error) {
	// All messages are text (utf-8 encoded at present)
	if !utf8.Valid([]byte(message)) {
		return nil, nil, errors.New("Message is not utf-8 encoded")
	}

	parts := strings.Split(message, "|")

	// rtc.io commands start with "/" - ignore everything else.
	if len(message) > 0 && message[0:1] == "/" {
		switch parts[0] {
		case "/announce":
			return announce, parts, nil

		case "/leave":
			return leave, parts, nil

		case "/to":
			return to, parts, nil

		case "/close":
			return closePeer, parts, nil

		default:
			return custom, parts, nil
		}
	}

	return ignore, parts, nil
}
