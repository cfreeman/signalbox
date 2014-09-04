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
	"fmt"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
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
		It("should be able to handle zero-length messages", func() {
			action, message, err := ParseMessage("")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.ignore"))
			Ω(len(message)).Should(Equal(1))
		})

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

		It("should be able to parse a close message", func() {
			action, message, err := ParseMessage("/close")
			Ω(err).Should(BeNil())
			Ω(runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()).Should(Equal("github.com/cfreeman/signalbox.closePeer"))
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

	Context("Test configuration parsing", func() {
		It("Should throw an error for an invalid config file", func() {
			config, err := parseConfiguration("foo")
			Ω(err).ShouldNot(BeNil())
			Ω(config.ListenAddress).Should(Equal(":3000"))
			Ω(config.SocketTimeout).Should(Equal(time.Duration(300) * time.Nanosecond))
		})

		It("Should be able to parse a valid config file", func() {
			config, err := parseConfiguration("testdata/test-config.json")
			Ω(err).Should(BeNil())
			Ω(config.ListenAddress).Should(Equal("10.1.2.3:4000"))
			Ω(config.SocketTimeout).Should(Equal(time.Duration(200) * time.Nanosecond))
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
		// Spin up the signalbox.
		go main()

		It("Should be to send announce and leave messages to peers", func() {
			a, err := connectPeer("a", "test-room")
			Ω(err).Should(BeNil())
			b, err := connectPeer("b", "test-room")
			Ω(err).Should(BeNil())

			socketShouldContain(a, "/announce|{\"id\":\"b\"}|{\"room\":\"test-room\"}")

			socketSend(a, "/leave|{\"id\":\"a\"}|{\"room\":\"test-room\"}")
			err = a.Close()
			Ω(err).Should(BeNil())

			socketShouldContain(b, "/leave|{\"id\":\"a\"}|{\"room\":\"test-room\"}")
			err = b.Close()
			Ω(err).Should(BeNil())
		})

		It("Should be able to send messages just to specified recipients", func() {
			a2, err := connectPeer("a2", "to-test")
			Ω(err).Should(BeNil())

			b2, err := connectPeer("b2", "to-test")
			Ω(err).Should(BeNil())

			c2, err := connectPeer("c2", "to-test")
			Ω(err).Should(BeNil())

			socketShouldContain(a2, "/announce|{\"id\":\"b2\"}|{\"room\":\"to-test\"}")
			socketShouldContain(a2, "/announce|{\"id\":\"c2\"}|{\"room\":\"to-test\"}")

			socketShouldContain(b2, "/announce|{\"id\":\"c2\"}|{\"room\":\"to-test\"}")
			socketSend(a2, "/to|c2|/hello|{\"id\":\"a1\"}")

			socketShouldContain(c2, "/to|c2|/hello|{\"id\":\"a1\"}")

			_, _, err = b2.ReadMessage()
			Ω(err).ShouldNot(BeNil())
		})

		It("Should be able to send custom messages to peers", func() {
			a3, err := connectPeer("a3", "custom-test")
			Ω(err).Should(BeNil())
			b3, err := connectPeer("b3", "custom-test")
			Ω(err).Should(BeNil())

			socketShouldContain(a3, "/announce|{\"id\":\"b3\"}|{\"room\":\"custom-test\"}")
			socketSend(a3, "/hello|{\"id\":\"a3\"}")
			socketShouldContain(b3, "/hello|{\"id\":\"a3\"}")
		})

		It("Should get a leave message when a peer disconnects", func() {
			a4, err := connectPeer("a4", "close-test")
			Ω(err).Should(BeNil())
			b4, err := connectPeer("b4", "close-test")

			socketShouldContain(a4, "/announce|{\"id\":\"b4\"}|{\"room\":\"close-test\"}")

			err = a4.Close()
			Ω(err).Should(BeNil())
			socketShouldContain(b4, "/leave|{\"id\":\"a4\"}|{\"room\":\"close-test\"}")
		})

		It("Should be able to handle very long messages", func() {
			a5, err := connectPeer("a5", "long-test")
			Ω(err).Should(BeNil())
			b5, err := connectPeer("b5", "long-test")

			socketShouldContain(a5, "/announce|{\"id\":\"b5\"}|{\"room\":\"long-test\"}")

			socketSend(b5, "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa|{\"id\":\"b5\"}")

			socketShouldContain(a5, "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa|{\"id\":\"b5\"}")
		})
	})
})

func socketSend(ws *websocket.Conn, content string) {
	msg, err := json.Marshal(content)
	Ω(err).Should(BeNil())
	ws.WriteMessage(websocket.TextMessage, msg)
	ws.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
}

func socketShouldContain(ws *websocket.Conn, content string) {
	_, message, err := ws.ReadMessage()
	Ω(err).Should(BeNil())
	expected, err := json.Marshal(content)
	Ω(err).Should(BeNil())
	Ω(message).Should(Equal(expected))
	ws.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
}

func connectPeer(id string, room string) (*websocket.Conn, error) {
	url := "ws://localhost:3000"
	res, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil || res == nil {
		return nil, err
	}

	connect, err := json.Marshal(fmt.Sprintf("/announce|{\"id\":\"%s\"}|{\"room\":\"%s\"}", id, room))
	if err != nil {
		return nil, err
	}

	err = res.WriteMessage(websocket.TextMessage, connect)
	if err != nil {
		return nil, err
	}

	res.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	res.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))

	return res, nil
}
