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
	"fmt"
	"io"
	"net/http"
)

func Message(ws *websocket.Conn) {
	// Pull the message out and parse the command structure.
	// All messages are text (utf-8 encoded at present)

	// Message parts are delimited by a pipe (|) character
	// Message commands must be contained in the initial message part and can be recognized simply as their first character is the forward slash (/) character.
	// All messages (apart from /to messages) are distributed to all active peers currently "announced" in a room.
	// All signaling clients identify themselves with a unique, non-reusable id.

	// Echo the data received on the WebSocket.
	io.Copy(ws, ws)
}

func main() {
	fmt.Printf("SignalBox!\n")

	// Routes.
	http.Handle("/", websocket.Handler(Message))

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
