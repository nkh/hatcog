package main

import (
	"net"
	"crypto/tls"
	"os"
	"log"
	"time"
	"strings"
)

const (
    // One second in nanoseconds
	ONE_SECOND_NS = 1000 * 1000 * 1000

	// Standard IRC SSL port
	// http://blog.freenode.net/2011/02/port-6697-irc-via-tlsssl/
	SSL_PORT = "6697"
)

type External struct {
	socket     net.Conn
	name       string
	isClosing  bool
	fromServer chan *Line
    rawLog     *log.Logger
}

func NewExternal(server string,
nick string,
name string,
password string,
fromServer chan *Line) *External {

    logFilename := HOME + LOG_DIR + "raw.log"
    logFile, err := os.Create(logFilename)
    if err != nil {
        LOG.Println("Error creating raw log file: ", logFilename, err)
        os.Exit(1)
    }

    rawLog := log.New(logFile, "", log.LstdFlags)
	LOG.Println("Logging raw IRC messages to: ", logFilename)

	var socket net.Conn

	if strings.HasSuffix(server, SSL_PORT) {
		socket, err = tls.Dial("tcp", server, nil)
	} else {
		socket, err = net.Dial("tcp", server)
	}

	if err != nil {
		log.Fatal("Error on IRC connect:", err)
	}
	time.Sleep(ONE_SECOND_NS)

	socket.SetReadTimeout(ONE_SECOND_NS)

	conn := External{
		socket:     socket,
		name:       name,
		fromServer: fromServer,
        rawLog:     rawLog,
	}
	conn.SendRaw("USER " + nick + " localhost localhost :" + name)
	conn.SendRaw("NICK " + nick)
	time.Sleep(ONE_SECOND_NS)

	if password != "" {
		conn.SendMessage("NickServ", "identify "+password)
	}

	return &conn
}

// Send a regular (non-system command) IRC message
func (self *External) SendMessage(channel, msg string) {
	fullmsg := "PRIVMSG " + channel + " :" + msg
	self.SendRaw(fullmsg)
}

// Send a /me action message
func (self *External) SendAction(channel, msg string) {
	fullmsg := "PRIVMSG " + channel + " :\u0001ACTION " + msg + "\u0001"
	self.SendRaw(fullmsg)
}

// Send message down socket. Add \n at end first.
func (self *External) SendRaw(msg string) {

	var err os.Error
	msg = msg + "\n"

	self.rawLog.Print(" -->", msg)

	_, err = self.socket.Write([]byte(msg))
	if err != nil {
		log.Fatal("Error writing to socket", err)
	}
}

// Process a slash command
func (self *External) doCommand(content string) {

	content = content[1:]
	self.SendRaw(content)
}

// Read IRC messages from the connection and send to stdout
func (self *External) Consume() {
	var data []byte = make([]byte, 1)
	var linedata []byte = make([]byte, 4096)
	var index int
	var line *Line
	var rawLine string
	var err os.Error = nil
	var netErr net.Error

	for {

		if self.isClosing {
			return
		}

		_, err = self.socket.Read(data)

		if err != nil {
			netErr, _ = err.(net.Error)

			// Need to timeout occasionally or we never check isClosing
			if netErr.Timeout() == true {
				continue
			} else {
				log.Fatal("Consume Error:", err)
			}
		}

		if data[0] == '\n' {
			if index == 0 {
				continue
			}
			rawLine = string(linedata[:index])
			self.rawLog.Println(rawLine)

			line, err = ParseLine(rawLine)
			if err == nil {
				self.act(line)
			} else {
				self.rawLog.Println(err)
				LOG.Println("Invalid line: " + rawLine)
			}

			index = 0
		} else if data[0] != '\r' { // Ignore CR, because LF is next
			linedata[index] = data[0]
			index++
		}
	}
}

// Do something with a line
func (self *External) act(line *Line) {

	if line.Command == "PING" {
		self.SendRaw("PONG goirc")
		return
	} else if line.Command == "VERSION" {
		versionMsg := "NOTICE " + line.User + " :\u0001VERSION " + VERSION + "\u0001\n"
		self.SendRaw(versionMsg)
	}

	self.fromServer <- line
}

func (self *External) Close() os.Error {
	return self.socket.Close()
}

/* Close connection, return from event loop.
 */
func (self *External) Quit() {
	self.isClosing = true
}