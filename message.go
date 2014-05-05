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

	// Inject reference to websocket into new peer.
	source.socket = sourceSocket

	peer, exists := state.Peers[source.Id]
	if !exists {
		fmt.Printf("Adding Peer: %s\n", source.Id)
		state.Peers[source.Id] = source
	}

	room, exists := state.Rooms[destination.Room]
	if !exists {
		fmt.Printf("Adding Room: %s\n", destination.Room)
		state.Rooms[destination.Room] = destination
	}

	fmt.Printf("announcing - %s to %s\n", source.Id, destination.Room)

	// TODO Announce to the other peers in the room of the arrival of source.

	state.RoomContains[room.Room] = append(state.RoomContains[room.Room], &peer)
	state.PeerIsIn[peer.Id] = append(state.PeerIsIn[peer.Id], &room)

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

	fmt.Printf("to!\n")
	return state, nil
}

func custom(messageBody []string,
	sourceSocket *websocket.Conn,
	state SignalBox) (newState SignalBox, err error) {

	fmt.Printf("custom!\n")
	return state, nil
}

func ParsePeerAndRoom(messageBody []string) (source Peer, destination Room, err error) {
	err = json.Unmarshal([]byte(messageBody[0]), &source)
	if err != nil {
		return Peer{}, Room{}, err
	}

	err = json.Unmarshal([]byte(messageBody[1]), &destination)
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
		return announce, parts[1:], nil

	case "/leave":
		return leave, parts[1:], nil

	case "/to":
		return to, parts[1:], nil

	default:
		return custom, parts[1:], nil
	}

	return nil, nil, nil
}
