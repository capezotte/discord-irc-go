package main

import (
	"fmt"; "log"
	"regexp"
	"github.com/lrstanley/girc"
)

var IEmojiRegex    *regexp.Regexp
var IMentionRegex  *regexp.Regexp
var IUserDChannels map[string]string // map IRC username → Discord Channel

var IBot *girc.Client

func IOnConnect(c *girc.Client, e girc.Event) {
	ch := Config.IRCChannel
	c.Cmd.Join(ch)
	log.Printf("Joining %s\n", ch)
}

func IOnPrivMsg(c *girc.Client, e girc.Event) {
	// Delegate channel messages to discord bot so he can do webhook stuff
	if e.IsFromChannel() { DIRCMessage(e.Source.Name, e.Last()); return }
	// private message, channel switch
	// get ready for wrong
	reply := "Canal indisponível. Opções são:"
	msg := e.Last()
	for hname, _ := range Config.Hooks {
		if hname == msg {
			IUserDChannels[e.Source.Name] = hname
			c.Cmd.Messagef(e.Source.Name, "Você está postando em %s", hname)
			log.Printf("[IRC] Moved %s's messages to %s", e.Source.Name, hname)
			return
		}
		reply = fmt.Sprintf("%s %s ;", reply, hname)
	}
	c.Cmd.Message(e.Source.Name, reply)
}

func IReplEmoji(input string) string {
	if DEmoji == nil { return input }
	for _, e := range DEmoji {
		if e.Name == input[1:len(input)-1] {
			prefix := ""
			if e.Animated { prefix = "a" }
			return fmt.Sprintf("<%s:%s:%s>", prefix, e.Name, e.ID)
		}
	}
	return input
}

func IReplMention(input string) string {
	if DNameToID == nil { return input }
	u := string(IMentionRegex.ExpandString([]byte{}, "$u", input, IMentionRegex.FindStringSubmatchIndex(input)))
	if DNameToID[u].ID != "" {
		return fmt.Sprintf("<@!%s>", DNameToID[u].ID)
	} else {
		return input
	}
}

func IInit() {
	IEmojiRegex = regexp.MustCompile(":[[:alnum:]_]+:")
	IMentionRegex = regexp.MustCompile("^(?P<u>[[:alnum:]_]+):|@(?P<u>[[:alnum:]_]+)")
	IUserDChannels = make(map[string]string)

	// Init bot
	IBot = girc.New(Config.IRCSettings)
	IBot.Handlers.Add(girc.CONNECTED, IOnConnect)
	IBot.Handlers.Add(girc.PRIVMSG, IOnPrivMsg)

	if err := IBot.Connect(); err != nil {
		panic(fmt.Sprintf("Can't connect to IRC, %s", err))
	}
}
