/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package pubsubsql

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//
// lineReader implements standard input line reader.
type lineReader struct {
	reader *bufio.Reader
	quit   string
	line   string
}

// returns a new lineReader. 
func newLineReader(quit string) *lineReader {
	return &lineReader{
		reader: bufio.NewReader(os.Stdin),
		quit:   quit,
	}
}

// readLine reads line of text from standard input. 
// Returns true if quit string was read.
func (l *lineReader) readLine() bool {
	line, err := l.reader.ReadString('\n')
	l.line = strings.TrimSpace(line)
	if err != nil {
		return false
	}
	return l.line != l.quit
}

//
// cli implements text prompt command line interface.
type cli struct {
	prefix        string
	quit          *Quitter
	fromStdin     chan string
	fromServer    chan string
	toServer      chan string
	conn          net.Conn
	disconnecting bool
}

// Returns new cli.
func newCli() *cli {
	return &cli{
		quit: 	       NewQuitter(),
		fromStdin:     make(chan string),
		fromServer:    make(chan string),
		toServer:      make(chan string),
		disconnecting: false,
	}
}

// the run is an event loop function that recieves a command line input and forwards it to the server.
func (this *cli) run() {
	this.initConsolePrefix()
	if !this.connect() {
		return
	}
	// start processing goroutines
	go this.readInput()
	go this.readMessages()
	go this.writeMessages()
	// 
	cout := bufio.NewWriter(os.Stdout)
	for {
		// display console prefix
		cout.WriteString(this.prefix)
		cout.Flush()
		select {
		case userInput := <-this.fromStdin:
			// indicate that we are trying to disconnect from the server.
			// but not quiting yet.
			switch userInput {
			case "close":
				this.disconnecting = true
			case "stop":
				this.disconnecting = true
			}
			// forward command to the server.
			this.toServer <- userInput
		case serverMessage := <-this.fromServer:
			// display the message returned from the server.
			cout.WriteString(serverMessage)
			cout.WriteString("\n")
			cout.Flush()
		case <-this.quit.GetChan():
			break
		}
	}
	this.conn.Close()
	this.quit.Wait(time.Millisecond * config.WAIT_MILLISECOND_CLI_SHUTDOWN)
	debug("cli done")
}

// connect establishes tcp connection to the serer.
func (this *cli) connect() bool {
	conn, err := net.Dial("tcp", config.netAddress())
	if err != nil {
		this.outputError(err)
		return false
	}
	this.conn = conn
	return true
}

// initConsolePrefix initializes console prefix string displayed to a user when waiting for the user's input.
func (this *cli) initConsolePrefix() {
	def := defaultConfig()
	this.prefix = "pubsubsql"
	if def.IP != config.IP {
		this.prefix += " " + config.netAddress()
	} else if def.PORT != config.PORT {
		this.prefix += ":" + strconv.Itoa(int(config.PORT))
	}
	this.prefix += ">"
}

// readInput reads a command line input from the standard input and forwards it for further processing. 
func (this *cli) readInput() {
	// we do not join the quitter because there is no way to return from blocking readLine
	cin := newLineReader("q")
	for cin.readLine() {
		if len(cin.line) > 0 {
			this.fromStdin <- cin.line
		}
	}
	// notify the connected server that we want to close the connection
	this.fromStdin <- "close"
}

// read reads messages from the server and forwards it for further processing.
func (this *cli) readMessages() {
	this.quit.Join()
	defer this.quit.Leave()
	reader := newNetMessageReaderWriter(this.conn, nil)
	for {
		bytes, err := reader.readMessage()
		if err != nil {
			this.outputError(err)
			break
		}
		select {
		case this.fromServer <- string(bytes):
		case <-this.quit.GetChan():
			break
		}
	}
	this.quit.Quit(0)
	debug("done readMessages")
}

// writeMessages writes messages to the server.
func (this *cli) writeMessages() {
	this.quit.Join()
	defer this.quit.Leave()
	writer := newNetMessageReaderWriter(this.conn, nil)
	for {
		select {
		case message, ok := <-this.toServer:
			if !ok {
				break
			}
			bytes := []byte(message)
			err := writer.writeHeaderAndMessage(bytes)
			if err != nil {
				this.outputError(err)
				break
			}
		case <-this.quit.GetChan():
			break
		}
	}
	this.quit.Quit(0)
	debug("done writeMessages")
}

// outputs error string if quit protocol is not in progress and the client is not trying to disconnect from the server.
func (this *cli) outputError(err error) {
	if !this.quit.Done() && !this.disconnecting {
		errorx(err)
	}
}

