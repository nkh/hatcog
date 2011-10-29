package main

import (
	"strings"
    "os"
    "io"
    "time"
    "log"
)

const (
	SYS_COMMANDS = "004 005 254 353 366 376 MODE JOIN PING"
)

type Line struct {
	raw     string
	User    string
	host    string
	Command string
	Args    []string
	Content string
    isAction bool
}

// Maps nicks to color
var colorMap = make(map[string]string)

// Logs raw IRC messages
var rawLog *log.Logger;

func init() {
    var logfile *os.File;
    logfile, _ = os.Create("/tmp/go-join.log");
    rawLog = log.New(logfile, "", log.LstdFlags);
}

func (self *Line) HasDisplay() bool {
	return !strings.Contains(SYS_COMMANDS, self.Command)
}

func (self *Line) String() string {

    var now *time.Time
    var output string
    var username string

    now = time.LocalTime()

    // see http://golang.org/src/pkg/time/format.go?s=7285:7328#L17
    output = now.Format("15:04")

	if self.User != "" {

        // TODO: if self.User == conn.nick: username=bold(username)
        username = colorfullUser(self.User)

        if self.isAction {
            username = Lpad(23, "* " + username)
        } else {
            username = Lpad(23, username)
        }

        output += " " + username + " "
	}

    output += self.Content

    output += "\n\r"
    return output
}

func (self *Line) Display(out io.Writer) {
    out.Write( []uint8(self.String()) )
}

func ParseLine(data string) Line {

	var line Line
	var prefix, command, trailing, user, host, raw string
	var args, parts []string
    var isAction bool

	data = sane(data)

    rawLog.Println(data);

	raw = data
	if data[0] == ':' { // Do we have a prefix?
		parts = strings.SplitN(data[1:], " ", 2)
		prefix = parts[0]
		data = parts[1]

		if strings.Contains(prefix, "!") {
			parts = strings.Split(prefix, "!")
			user = parts[0]
			host = parts[1]
		} else {
			host = prefix
		}
	}

	if strings.Index(data, " :") != -1 {
		parts = strings.SplitN(data, " :", 2)
		data = parts[0]
		args = strings.Split(data, " ")

		trailing = sane(parts[1])
	} else {
		args = strings.Split(data, " ")
	}
	command = args[0]
	args = args[1:len(args)]

    isAction = false
    if strings.HasPrefix(trailing, "ACTION") {
        // Received a /me line
        trailing = strings.SplitN(trailing, " ", 2)[1]
        isAction = true
    }

	line = Line{
		raw:     raw,
		User:    user,
		host:    host,
		Command: command,
		Args:    args,
		Content: trailing,
        isAction: isAction,
	}

	return line
}

func colorfullUser(nick string) string {

    if colorMap[nick] == "" {
        nextColorIndex := len(colorMap) % len(UserColors)
        colorMap[nick] = UserColors[nextColorIndex]
    }

    return Color(colorMap[nick], nick)
}
