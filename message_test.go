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
	// "fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// "github.com/gorilla/websocket"
	"reflect"
	"runtime"
	"testing"
	// "time"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message Suite")
}

var _ = Describe("Message", func() {
	Context("Utf8 encoding", func() {
		It("should return an error for non-utf8 encoded messages", func() {
			_, _, err := ParseMessage(string([]byte{0xff, 0xfe, 0xfd}))
			Ω(err).ShouldNot(BeNil())
		})

		It("should should not return an error for utf8 encoded messages", func() {
			_, _, err := ParseMessage("/announce|{\"id\":\"dc6ac0ae-6e15-409b-b211-228a8f4a43b9\"}|{\"browser\":\"node\",\"browserVersion\":\"?\",\"id\":\"dc6ac0ae-6e15-409b-b211-228a8f4a43b9\",\"agent\":\"signaller@0.18.3\",\"room\":\"test-room\"}")
			Ω(err).Should(BeNil())
		})
	})

	Context("Action parsing", func() {
		It("should be able to parse an announce message", func() {
			action, message, err := ParseMessage("/announce")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.announce"))
			Ω(len(message)).Should(Equal(1))
		})

		It("should be able to parse a leave message", func() {
			action, message, err := ParseMessage("/leave")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.leave"))
			Ω(len(message)).Should(Equal(1))
		})

		It("should be able to parse a to message", func() {
			action, message, err := ParseMessage("/to")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.to"))
			Ω(len(message)).Should(Equal(1))
		})

		It("should be able to parse a custom message", func() {
			action, message, err := ParseMessage("/custom|part1|part2")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.custom"))
			Ω(len(message)).Should(Equal(3))
			Ω(message[0]).Should(Equal("/custom"))
			Ω(message[1]).Should(Equal("part1"))
			Ω(message[2]).Should(Equal("part2"))
		})

		// TODO: Test Malformed messages.
	})

	Context("ParsePeerAndRoom", func() {
		It("should return an error when their is not enough parts to a message", func() {
			_, message, _ := ParseMessage("/foo")
			_, _, err := ParsePeerAndRoom(message)
			Ω(err).ShouldNot(BeNil())
		})

		It("should parse source id and room", func() {
			_, message, _ := ParseMessage("/announce|{\"id\":\"abc\"}|{\"room\":\"test\"}")
			source, destination, err := ParsePeerAndRoom(message)
			Ω(err).Should(BeNil())
			Ω(source.Id).Should(Equal("abc"))
			Ω(destination.Room).Should(Equal("test"))
		})
	})
})

// func TestAnnounce(t *testing.T) {
// 	action, message, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing announce message")
// 	}

// 	action, message2, err := ParseMessage("/announce|{\"id\":\"b\"}|{\"room\":\"test\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing announce message")
// 	}

// 	state := SignalBox{make(map[string]*Peer),
// 		make(map[string]*Room),
// 		make(map[string]map[string]*Peer),
// 		make(map[string]map[string]*Room)}

// 	state, err = action(message, nil, state)
// 	state, err = action(message, nil, state)
// 	if len(state.Peers) != 1 {
// 		t.Errorf("Expected the total number of peers in the signal box to be 1.")
// 	}

// 	if len(state.Rooms) != 1 {
// 		t.Errorf("Expected the total number of rooms in the signal box to be 1.")
// 	}

// 	if len(state.RoomContains["test"]) == 1 && state.RoomContains["test"]["a"].Id != "a" {
// 		t.Errorf("Room doesn't contain a.")
// 	}

// 	if len(state.PeerIsIn["a"]) == 1 && state.PeerIsIn["a"]["test"].Room != "test" {
// 		t.Errorf("abc is not in room test")
// 	}

// 	state, err = action(message2, nil, state)
// 	if len(state.Peers) != 2 {
// 		t.Errorf("Expected the total number of peers in the signal box to be 2.")
// 	}

// 	if len(state.Rooms) != 1 {
// 		t.Errorf("Exected the total number of rooms in the signal box to be 1.")
// 	}

// 	if len(state.RoomContains["test"]) != 2 {
// 		t.Errorf("Expected the test room to contain two peers.")
// 	}
// }

// func connectPeer(id string, room string) (*websocket.Conn, error) {
// 	url := "ws://localhost:3000"
// 	res, _, err := websocket.DefaultDialer.Dial(url, nil)
// 	if err != nil || res == nil {
// 		return nil, err
// 	}

// 	connect := fmt.Sprintf("/announce|{\"id\":\"%s\"}|{\"room\":\"%s\"}", id, room)
// 	err = res.WriteMessage(websocket.TextMessage, []byte(connect))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

// func TestAnnounceBroadcast(t *testing.T) {
// 	go main()

// 	a, err := connectPeer("a", "test-room")
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	b, err := connectPeer("b", "test-room")
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	_, message, err := a.ReadMessage()
// 	if err != nil || string(message) != "/announce|{\"id\":\"b\"}|{\"room\":\"test-room\"}" {
// 		t.Errorf("Peer A did not recieve the announce message for b.")
// 	}

// 	err = a.WriteMessage(websocket.TextMessage, []byte("/leave|{\"id\":\"a\"}|{\"room\":\"test-room\"}"))
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, message, err = b.ReadMessage()
// 	if err != nil || string(message) != "/leave|{\"id\":\"a\"}|{\"room\":\"test-room\"}" {
// 		t.Errorf("Peer B did not recieve the leave message for a.")
// 	}
// }

// func TestMessage(t *testing.T) {
// 	a, err := connectPeer("a1", "test-to-message")
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	time.Sleep(2 * time.Millisecond)

// 	b, err := connectPeer("b2", "test-to-message")
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	time.Sleep(2 * time.Millisecond)

// 	c, err := connectPeer("c2", "test-to-message")
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	time.Sleep(2 * time.Millisecond)

// 	// Test custom Message.
// 	err = a.WriteMessage(websocket.TextMessage, []byte("/hello|{\"id\":\"a1\"}"))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	time.Sleep(2 * time.Millisecond)

// 	b.ReadMessage() // discard the c2 announce message.
// 	_, b_message, err := b.ReadMessage()
// 	if err != nil || string(b_message) != "/hello|{\"id\":\"a1\"}" {
// 		t.Errorf(string(b_message))
// 		t.Errorf("Peer B did not recieve the message from A.")
// 	}

// 	_, c_message, err := c.ReadMessage()
// 	if err != nil || string(c_message) != "/hello|{\"id\":\"a1\"}" {
// 		t.Errorf(string(c_message))
// 		t.Errorf("Peer C did not recieve the message from A.")
// 	}

// 	// Test TO Message.
// 	err = a.WriteMessage(websocket.TextMessage, []byte("/to|b2|/hello|{\"id\":\"a1\"}"))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	time.Sleep(2 * time.Millisecond)

// 	_, b_message, err = b.ReadMessage()
// 	if err != nil || string(b_message) != "/to|b2|/hello|{\"id\":\"a1\"}" {
// 		t.Errorf(string(b_message))
// 		t.Errorf("Peer B did not recieve the personal message from A.")
// 	}

// 	c.SetReadDeadline(time.Now().Add(4 * time.Millisecond))
// 	_, c_message, err = c.ReadMessage()
// 	if string(c_message) != "" {
// 		t.Errorf("Peer C was not expecting any messages.")
// 	}
// }

// func TestLeave(t *testing.T) {
// 	announceA, message, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing announce message for a")
// 	}

// 	announceB, message2, err := ParseMessage("/announce|{\"id\":\"b\"}|{\"room\":\"test\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing announce message for b")
// 	}

// 	announceA2, message4, err := ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test2\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error pasring announce message for a entering test2")
// 	}

// 	leaveA, message3, err := ParseMessage("/leave|{\"id\":\"a\"}|{\"room\":\"test2\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing leave message for a")
// 	}

// 	leaveA2, message5, err := ParseMessage("/leave|{\"id\":\"a\"}|{\"room\":\"test\"}")
// 	if err != nil {
// 		t.Errorf("Unexpected error parsing leave message for a")
// 	}

// 	state := SignalBox{make(map[string]*Peer),
// 		make(map[string]*Room),
// 		make(map[string]map[string]*Peer),
// 		make(map[string]map[string]*Room)}

// 	state, err = announceA(message, nil, state)
// 	if err != nil {
// 		t.Errorf("Unexpected error announcing A to the signalbox")
// 	}

// 	state, err = announceB(message2, nil, state)
// 	if err != nil {
// 		t.Errorf("Unexpected error announcing B to the signalbox")
// 	}

// 	state, err = announceA2(message4, nil, state)
// 	if err != nil {
// 		t.Errorf("Unexpected error anouncing A to the test2 room.")
// 	}

// 	if len(state.Peers) != 2 {
// 		t.Errorf("Expected two peers within the signal box.")
// 	}

// 	if len(state.Rooms) != 2 {
// 		t.Errorf("Expected two rooms within the signal box.")
// 	}

// 	if state.PeerIsIn["a"]["test"] != state.Rooms["test"] {
// 		t.Errorf("Expected a to be within room test")
// 	}

// 	if state.PeerIsIn["a"]["test2"] != state.Rooms["test2"] {
// 		t.Errorf("Expected a to be within room test2")
// 	}

// 	if state.RoomContains["test"]["a"] != state.Peers["a"] {
// 		t.Errorf("Expected room test to contain 'a'")
// 	}

// 	if state.RoomContains["test"]["b"] != state.Peers["b"] {
// 		t.Errorf("Expected room test to contain 'b'")
// 	}

// 	if state.RoomContains["test2"]["a"] != state.Peers["a"] {
// 		t.Errorf("Expected room test2 to contain 'a'")
// 	}

// 	state, err = leaveA(message3, nil, state)

// 	if len(state.Rooms) != 1 {
// 		t.Errorf("Expected only one room to be left within the signal box.")
// 	}

// 	_, exists := state.Rooms["test"]
// 	if !exists {
// 		t.Errorf("Expected signalbox to contain the test room.")
// 	}

// 	if len(state.Peers) != 2 {
// 		t.Errorf("Expected to have two peers left within the signal box.")
// 	}

// 	if state.RoomContains["test"]["a"] != state.Peers["a"] {
// 		t.Errorf("Expected room test to contain 'a'")
// 	}

// 	if state.RoomContains["test"]["b"] != state.Peers["b"] {
// 		t.Errorf("Expected room test to contain 'b'")
// 	}

// 	if len(state.PeerIsIn["a"]) != 1 {
// 		t.Errorf("Expected peer 'a' to be in only one room.")
// 	}

// 	if state.PeerIsIn["a"]["test"] != state.Rooms["test"] {
// 		t.Errorf("Expected peer 'a' to be within room test.")
// 	}

// 	state, err = leaveA2(message5, nil, state)

// 	if len(state.Rooms) != 1 {
// 		t.Errorf("Expected only one room to be left within the signal box.")
// 	}

// 	_, exists = state.Rooms["test"]
// 	if !exists {
// 		t.Errorf("Expected signalbox to contain the test room.")
// 	}

// 	if len(state.Peers) != 1 {
// 		t.Errorf("Expected to have one peer left within the signal box.")
// 	}

// 	if state.Peers["b"].Id != "b" {
// 		t.Errorf("Expected peer b to be within the signalbox still.")
// 	}

// 	_, exists = state.PeerIsIn["a"]
// 	if exists {
// 		t.Errorf("Expected peer 'a' not to be around anymore.")
// 	}

// 	if len(state.RoomContains) != 1 {
// 		t.Errorf("Expected room contains to only have 'test' left.")
// 	}

// 	if len(state.RoomContains["test"]) != 1 {
// 		t.Errorf("Expected test room to only contain one peer.")
// 	}

// 	_, exists = state.RoomContains["test"]["a"]
// 	if exists {
// 		t.Errorf("Expected peer 'a' not to be in room 'test' anymore")
// 	}
// }
