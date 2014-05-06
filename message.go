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
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type messageFn func(messageBody []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error)

func announce(messageBody []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source, destination, err := ParsePeerAndRoom(messageBody)
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

	roomContainsPeer := false
	for _, p := range state.RoomContains[room.Room] {
		if p.Id != peer.Id {
			if p.socket != nil {
				fmt.Printf("writing %s to %s\n", strings.Join(messageBody, "|"), p.Id)
				_, err := p.socket.Write([]byte(strings.Join(messageBody, "|")))
				if err != nil {
					fmt.Printf("Unable to write - %s\n", err)
				}
			}
		} else {
			roomContainsPeer = true
		}
	}
	if !roomContainsPeer {
		state.RoomContains[room.Room] = append(state.RoomContains[room.Room], peer)
	}

	peerIsInRoom := false
	for _, r := range state.PeerIsIn[peer.Id] {
		if r.Room == room.Room {
			peerIsInRoom = true
		}
	}
	if !peerIsInRoom {
		state.PeerIsIn[peer.Id] = append(state.PeerIsIn[peer.Id], room)
	}

	return state, nil
}

func leave(messageBody []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	source, destination, err := ParsePeerAndRoom(messageBody)
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

	fmt.Printf("%s is leaving %s\n", peer.Id, room.Room)

	// TODO tell the other peers in the room that source is leaving.

	// TODO clean up our data structure.

	return state, nil
}

func to(messageBody []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	fmt.Printf("to message\n")
	return state, nil
}

func custom(messageBody []string,
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
