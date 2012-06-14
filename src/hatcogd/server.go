package main

import (
	"log"
	"os/exec"
	"strings"
)

const (
	RPL_NAMREPLY = "353"
)

var (
	INFO_CMDS = []string{"001", "002", "003", "004", "372", "NOTICE"}
)

type Server struct {
	nick           string
	external       *ExternalManager
	internal       *InternalManager
	cmdPrivateChat string
	fromServer     chan *Line
	fromUser       chan Message
}

func NewServer(host, port, cmdPrivateChat string) *Server {

	fromServer := make(chan *Line)
	fromUser := make(chan Message)

	// Socket connections from client programs
	internal := NewInternalManager(host, port, fromUser)

	// Socket connections to IRC servers
	external := NewExternalManager(fromServer)

    log.Println("Listening for internal connection on " + host + ":" + port)

	return &Server{
		"",
		external,
		internal,
		cmdPrivateChat,
		fromServer,
		fromUser}
}

// Main loop
func (self *Server) Run() {

	go self.internal.Run()

	for {

		select {
		case serverLine := <-self.fromServer:
			self.onServer(serverLine)

		case userMessage := <-self.fromUser:
			self.onUser(userMessage)
		}
	}
}

func (self *Server) Close() error {
	self.internal.Close()
	return self.external.Close()
}

// Act on server messages
func (self *Server) onServer(line *Line) {

	if isInfoCommand(line.Command) {
		log.Println(line.Content)
	}

	if len(line.Channel) == 0 && !isChannelRequired(line.Command) {
		self.internal.WriteAll(line.Network, line.AsJson())
	} else {
		self.internal.WriteChannel(line.Network, line.Channel, line.AsJson())
	}

	isMsg := (line.Command == "PRIVMSG")
	isPrivate := isMsg && (line.User == line.Channel)

	if isPrivate && !self.internal.HasChannel(line.Channel) {
		self.internal.lastPrivate = []byte(line.AsJson())
		go self.openPrivate(line.User)
	}

}

// Act on user input
func (self *Server) onUser(message Message) {

	var cmd, content string

	if isCommand(message.content) {

		parts := strings.SplitN(message.content[1:], " ", 2)
		cmd = parts[0]
		if len(parts) == 2 {
			content = parts[1]
		}

		if cmd == "pw" {
			self.external.Identify(message.network, content)

		} else if cmd == "me" {
			self.external.SendAction(message.network, message.channel, content)

		} else if cmd == "nick" {
			newNick := content
			self.nick = newNick
			self.internal.Nick = newNick

			self.external.doCommand(message.network, message.content)

		} else if cmd == "connect" {
			// Connect to a remote IRC server
			self.external.Connect(content)

		} else {
			self.external.doCommand(message.network, message.content)
		}

	} else {
		self.external.SendMessage(message.network, message.channel, message.content)
	}

}

// Ask window manager to open a new pane for private messages with given user
func (self *Server) openPrivate(nick string) {

	// TODO: Sanitise nick to prevent command execution

	parts := strings.Split(self.cmdPrivateChat, " ")
	parts = append(parts, "/usr/local/bin/hjoin -private="+nick)

	command := exec.Command(parts[0], parts[1:]...)
	command.Run()
}

// Is 'content' an IRC command?
func isCommand(content string) bool {
	return len(content) > 1 && content[0] == '/'
}

// Is 'command' an IRC information command?
func isInfoCommand(command string) bool {

	for _, cmd := range INFO_CMDS {
		if cmd == command {
			return true
		}
	}
	return false
}

// Does command require a channel
func isChannelRequired(command string) bool {
	return command == RPL_NAMREPLY
}
