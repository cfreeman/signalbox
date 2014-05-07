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
		make(map[string][]*Peer),
		make(map[string][]*Room)}

	state, err = action(message, nil, state)
	state, err = action(message, nil, state)
	if len(state.Peers) != 1 {
		t.Errorf("Expected the total number of peers in the signal box to be 1.")
	}

	if len(state.Rooms) != 1 {
		t.Errorf("Expected the total number of rooms in the signal box to be 1.")
	}

	if len(state.RoomContains["test"]) == 1 && state.RoomContains["test"][0].Id != "a" {
		t.Errorf("Room doesn't contain a.")
	}

	if len(state.PeerIsIn["a"]) == 1 && state.PeerIsIn["a"][0].Room != "test" {
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
