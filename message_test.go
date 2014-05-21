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

		It("should ignore malformed messages", func() {
			action, message, err := ParseMessage(":lkajsd??asdj/foo")
			Ω(err).Should(BeNil())
			Ω(len(message)).Should(Equal(1))
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.ignore"))
		})
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

	Context("Test SignalBox State", func() {
		var state SignalBox
		var announceAAct messageFn
		var announceAMsg []string

		var announceA2Act messageFn
		var announceA2Msg []string

		var leaveAAct messageFn
		var leaveAMsg []string

		var leaveA2Act messageFn
		var leaveA2Msg []string

		var announceBAct messageFn
		var announceBMsg []string

		var leaveBAct messageFn
		var leaveBMsg []string

		BeforeEach(func() {
			var err error
			state = SignalBox{make(map[string]*Peer),
				make(map[string]*Room),
				make(map[string]map[string]*Peer),
				make(map[string]map[string]*Room)}

			announceAAct, announceAMsg, err = ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test\"}")
			Ω(err).Should(BeNil())

			announceA2Act, announceA2Msg, err = ParseMessage("/announce|{\"id\":\"a\"}|{\"room\":\"test2\"}")
			Ω(err).Should(BeNil())

			leaveAAct, leaveAMsg, err = ParseMessage("/leave|{\"id\":\"a\"}|{\"room\":\"test\"}")
			Ω(err).Should(BeNil())

			leaveA2Act, leaveA2Msg, err = ParseMessage("/leave|{\"id\":\"a\"}|{\"room\":\"test2\"}")
			Ω(err).Should(BeNil())

			announceBAct, announceBMsg, err = ParseMessage("/announce|{\"id\":\"b\"}|{\"room\":\"test\"}")
			Ω(err).Should(BeNil())

			leaveBAct, leaveBMsg, err = ParseMessage("/leave|{\"id\":\"b\"}|{\"room\":\"test\"}")
			Ω(err).Should(BeNil())
		})

		It("only add someone to the roome once, even if they announce more than once", func() {
			state, err := announceAAct(announceAMsg, nil, state)
			Ω(err).Should(BeNil())
			state, err = announceAAct(announceAMsg, nil, state)
			Ω(err).Should(BeNil())

			Ω(len(state.Peers)).Should(Equal(1))
			Ω(len(state.Rooms)).Should(Equal(1))
			Ω(len(state.RoomContains)).Should(Equal(1))
			Ω(len(state.PeerIsIn)).Should(Equal(1))
			Ω(len(state.RoomContains["test"])).Should(Equal(1))
			Ω(len(state.PeerIsIn["a"])).Should(Equal(1))
			Ω(state.RoomContains["test"]["a"].Id).Should(Equal("a"))
			Ω(state.PeerIsIn["a"]["test"].Room).Should(Equal("test"))
		})

		It("should be able to add multiple people to a signalbox room", func() {
			state, err := announceAAct(announceAMsg, nil, state)
			Ω(err).Should(BeNil())
			state, err = announceBAct(announceBMsg, nil, state)
			Ω(err).Should(BeNil())

			Ω(len(state.Peers)).Should(Equal(2))
			Ω(len(state.Rooms)).Should(Equal(1))
			Ω(len(state.PeerIsIn)).Should(Equal(2))
			Ω(len(state.RoomContains)).Should(Equal(1))
			Ω(len(state.PeerIsIn["a"])).Should(Equal(1))
			Ω(len(state.PeerIsIn["b"])).Should(Equal(1))
			Ω(len(state.RoomContains["test"])).Should(Equal(2))
		})

		It("Should be able to have a person leave a signalbox room", func() {
			state, err := announceAAct(announceAMsg, nil, state)
			Ω(err).Should(BeNil())
			state, err = leaveAAct(leaveAMsg, nil, state)
			Ω(err).Should(BeNil())

			Ω(len(state.Peers)).Should(Equal(0))
			Ω(len(state.Rooms)).Should(Equal(0))
			Ω(len(state.PeerIsIn)).Should(Equal(0))
			Ω(len(state.RoomContains)).Should(Equal(0))
		})

		It("Should keep a room, if a person leaves but it still contains peers", func() {
			state, err := announceBAct(announceBMsg, nil, state)
			Ω(err).Should(BeNil())
			state, err = announceAAct(announceAMsg, nil, state)
			Ω(err).Should(BeNil())
			state, err = announceA2Act(announceA2Msg, nil, state)
			Ω(err).Should(BeNil())
			state, err = leaveBAct(leaveBMsg, nil, state)
			Ω(err).Should(BeNil())

			Ω(len(state.Peers)).Should(Equal(1))
			Ω(len(state.Rooms)).Should(Equal(2))
			Ω(len(state.PeerIsIn)).Should(Equal(1))
			Ω(len(state.RoomContains)).Should(Equal(2))
			Ω(len(state.PeerIsIn["a"])).Should(Equal(2))
			Ω(len(state.RoomContains["test"])).Should(Equal(1))
			Ω(len(state.RoomContains["test2"])).Should(Equal(1))
			Ω(state.RoomContains["test"]["a"].Id).Should(Equal("a"))
			Ω(state.RoomContains["test2"]["a"].Id).Should(Equal("a"))
		})
	})

	Context("Broadcast messages", func() {
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

		go main()

	})
})

// TODO: Port the rest of the tests over to Ginkgo.

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
