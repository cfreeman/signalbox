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
	"github.com/gorilla/websocket"
	"reflect"
	"runtime"
	"testing"
)

func TestUtf8Encoding(t *testing.T) {
	message := string([]byte{0xff, 0xfe, 0xfd})
	_, _, err := ParseMessage(message)
	if err == nil {
		t.Errorf("Expected utf8 error")
	}

	_, _, err = ParseMessage("/announce|{\"id\":\"dc6ac0ae-6e15-409b-b211-228a8f4a43b9\"}|{\"browser\":\"node\",\"browserVersion\":\"?\",\"id\":\"dc6ac0ae-6e15-409b-b211-228a8f4a43b9\",\"agent\":\"signaller@0.18.3\",\"room\":\"test-room\"}")
	if err != nil {
		t.Errorf("Unexpected utf8 error")
	}
}

func TestParseMessage(t *testing.T) {
	action, message, err := ParseMessage("/announce")
	if err != nil || runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name() != "github.com/cfreeman/signalbox.announce" {
		t.Errorf("Announce message incorrectly parsed")
	}

	if len(message) != 1 {
		t.Errorf("Non empty body for announce message.")
	}

	action, _, err = ParseMessage("/leave")
	if err != nil || runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name() != "github.com/cfreeman/signalbox.leave" {
		t.Errorf("Leave message incorrectly parsed")
	}

	action, _, err = ParseMessage("/to")
	if err != nil || runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name() != "github.com/cfreeman/signalbox.to" {
		t.Errorf("To message incorrectly parsed")
	}

	action, message, err = ParseMessage("/custom|part1|part2")
	if err != nil || runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name() != "github.com/cfreeman/signalbox.custom" {
		t.Errorf("Custom message incorrectly parsed")
	}

	if len(message) != 3 || message[1] != "part1" || message[2] != "part2" {
		t.Errorf("incorrectly parsed the parts to the body of the message")
	}

	// TODO test malformed messages.
}

func TestParsePeerAndRoom(t *testing.T) {
	_, message, _ := ParseMessage("/foo")
	source, destination, err := ParsePeerAndRoom(message)
	if err == nil {
		t.Errorf("Expected pre-condition error parsing peer and room.")
	}

	_, message, _ = ParseMessage("/announce|{\"id\":\"abc\"}|{\"room\":\"test\"}")
	source, destination, err = ParsePeerAndRoom(message)
	if err != nil {
		t.Errorf("Unexpected error parsing peer and room from message")
	}

	if source.Id != "abc" || destination.Room != "test" {
		t.Errorf("Source or destination incorrectly parsed")
	}
}

func TestAnnounce(t *testing.T) {
	action, message, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test\"}")
	if err != nil {
		t.Errorf("Unexpected error parsing announce message")
	}

	action, message2, err := ParseMessage("/announce|{\"id\":\"b\"}|{\"room\":\"test\"}")
	if err != nil {
		t.Errorf("Unexpected error parsing announce message")
	}

	state := SignalBox{make(map[string]*Peer),
		make(map[string]*Room),
		make(map[string]map[string]*Peer),
		make(map[string]map[string]*Room)}

	state, err = action(message, nil, state)
	state, err = action(message, nil, state)
	if len(state.Peers) != 1 {
		t.Errorf("Expected the total number of peers in the signal box to be 1.")
	}

	if len(state.Rooms) != 1 {
		t.Errorf("Expected the total number of rooms in the signal box to be 1.")
	}

	if len(state.RoomContains["test"]) == 1 && state.RoomContains["test"]["a"].Id != "a" {
		t.Errorf("Room doesn't contain a.")
	}

	if len(state.PeerIsIn["a"]) == 1 && state.PeerIsIn["a"]["test"].Room != "test" {
		t.Errorf("abc is not in room test")
	}

	state, err = action(message2, nil, state)
	if len(state.Peers) != 2 {
		t.Errorf("Expected the total number of peers in the signal box to be 2.")
	}

	if len(state.Rooms) != 1 {
		t.Errorf("Exected the total number of rooms in the signal box to be 1.")
	}

	if len(state.RoomContains["test"]) != 2 {
		t.Errorf("Expected the test room to contain two peers.")
	}
}

func TestAnnounceBroadcast(t *testing.T) {
	go main()

	url := "ws://localhost:3000"
	a, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil || a == nil {
		t.Errorf("Bad socket")
		t.Error(err)
		return
	}

	err = a.WriteMessage(websocket.TextMessage, []byte("/announce|{\"id\":\"a\"}|{\"room\":\"test-room\"}"))
	if err != nil {
		t.Error(err)
	}

	b, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil || b == nil {
		t.Errorf("Bad Socket")
		t.Error(err)
		return
	}
	err = b.WriteMessage(websocket.TextMessage, []byte("/announce|{\"id\":\"b\"}|{\"room\":\"test-room\"}"))
	if err != nil {
		t.Error(err)
	}

	_, message, err := a.ReadMessage()
	if err != nil || string(message) != "/announce|{\"id\":\"b\"}|{\"room\":\"test-room\"}" {
		t.Errorf("Peer A did not recieve the announce message for b.")
	}
}

func TestLeave(t *testing.T) {
	announceA, message, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test\"}")
	if err != nil {
		t.Errorf("Unexpected error parsing announce message for a")
	}

	announceB, message2, err := ParseMessage("/announce|{\"id\":\"b\"}|{\"room\":\"test\"}")
	if err != nil {
		t.Errorf("Unexpected error parsing announce message for b")
	}

	announceA2, message4, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test2\"}")
	if err != nil {
		t.Errorf("Unexpected error pasring announce message for a entering test2")
	}

	leaveA, message3, err := ParseMessage("/leave|{\"id\":\"a\"}|{\"room\":\"test2\"}")
	if err != nil {
		t.Errorf("Unexpected error parsing leave message for a")
	}

	state := SignalBox{make(map[string]*Peer),
		make(map[string]*Room),
		make(map[string]map[string]*Peer),
		make(map[string]map[string]*Room)}

	state, err = announceA(message, nil, state)
	if err != nil {
		t.Errorf("Unexpected error announcing A to the signalbox")
	}

	state, err = announceB(message2, nil, state)
	if err != nil {
		t.Errorf("Unexpected error announcing B to the signalbox")
	}

	state, err = announceA2(message4, nil, state)
	if err != nil {
		t.Errorf("Unexpected error anouncing A to the test2 room.")
	}

	if len(state.Peers) != 2 {
		t.Errorf("Expected two peers within the signal box.")
	}

	if len(state.Rooms) != 2 {
		t.Errorf("Expected two rooms within the signal box.")
	}

	if state.PeerIsIn["a"]["test"] != state.Rooms["test"] {
		t.Errorf("Expected a to be within room test")
	}

	if state.PeerIsIn["a"]["test2"] != state.Rooms["test2"] {
		t.Errorf("Expected a to be within room test2")
	}

	if state.RoomContains["test"]["a"] != state.Peers["a"] {
		t.Errorf("Expected room test to contain 'a'")
	}

	if state.RoomContains["test"]["b"] != state.Peers["b"] {
		t.Errorf("Expected room test to contain 'b'")
	}

	if state.RoomContains["test2"]["a"] != state.Peers["a"] {
		t.Errorf("Expected room test2 to contain 'a'")
	}

	state, err = leaveA(message3, nil, state)

	if len(state.Rooms) != 1 {
		t.Errorf("Expected only one room to be left within the signal box.")
	}

	_, exists := state.Rooms["test"]
	if !exists {
		t.Errorf("Expected signalbox to contain the test room.")
	}

	// TODO: Finish testing the rest of the leave state.
}
