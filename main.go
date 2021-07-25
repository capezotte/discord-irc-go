package main

import (
	"fmt"
	"strings"
	"os"; "os/signal"; "syscall"
	"github.com/bwmarrin/discordgo"
	"github.com/lrstanley/girc"
	"encoding/json"
	"io/ioutil"
	"regexp"
)

type Settings struct {
	Token string
	IRCSettings girc.Config
	DefaultHook string
	Hooks map[string]struct{ Id string; Token string }
	GuildID string
	IRCChannel string
}

type CachedUserData struct {
	Id string
	AvatarURL string
}



func main() {
	wd, _ := os.Getwd()

	fmt.Printf("Starting bridge. Work directory is %s.\n", wd)
	var av map[string]CachedUserData

	fmt.Println("Importing previous avatar data from av.json.")
	avatars, _ := ioutil.ReadFile("av.json")
	err := json.Unmarshal([]byte(avatars), &av)
	if err != nil {
		fmt.Println("Can't read data, starting from scratch.")
		av = make(map[string]CachedUserData)
	}
	// Where users will post (user → hook name)
	WhereUser := make(map[string]string)

	fmt.Println("Importing settings from cfg.json.")
	var Config Settings;
	content, _ := ioutil.ReadFile("cfg.json")
	err = json.Unmarshal([]byte(content), &Config)
	if err != nil { fmt.Println("Absent or invalid cfg.json!", err); return }

	IBot := girc.New(Config.IRCSettings)

	IBot.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		fmt.Printf("Joining %s\n", Config.IRCChannel)
		c.Cmd.Join(Config.IRCChannel)
	})

	DBot, err := discordgo.New("Bot " + Config.Token)
	if err != nil {
		fmt.Println("Failed to create a discord session.")
		return
	}

	// Match emojis on IRC messages
	IEmojiRegex := regexp.MustCompile(":[[:alnum:]]+:")

	// Send Discord to IRC
	DBot.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages from our own webhooks
		for _, wh := range Config.Hooks {
			if wh.Id == m.WebhookID { return }
		}

		// Try to get nickname
		member, err := DBot.GuildMember(m.GuildID, m.Author.ID)

		var OriginString string
		if member.Nick == "" {
			OriginString = m.Author.Username
		} else {
			OriginString = member.Nick
		}

		// Get user's avatar
		if av[OriginString].AvatarURL == "" {
			av[OriginString] = CachedUserData{
				Id: m.Author.ID,
				AvatarURL: m.Author.AvatarURL(""),
			}
		}

		// Handle replying
		if m.MessageReference != nil {
			// Try to get info on message being replied to
			ReplyTo, _ := DBot.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
			replymember, err  := DBot.GuildMember(m.MessageReference.GuildID, ReplyTo.Author.ID)
			// fuck people deleting messages
			if err != nil {
				OriginString += " em resposta a uma conta/mensagem apagada"
			} else {
				var DestString string
				if replymember.Nick == "" {
					DestString = ReplyTo.Author.Username
				} else {
					DestString = replymember.Nick
				}
				OriginString = fmt.Sprintf("%s em reposta a %s", OriginString, DestString)
			}
		}
		// Try to get channel name
		channel, err := DBot.Channel(m.ChannelID)
		if err != nil { return }
		// message
		msg := m.ContentWithMentionsReplaced()
		for _, e := range m.Embeds {
			msg = fmt.Sprintf("%s %s", msg, e.URL)
		}
		// attachments
		for _, e := range m.Attachments {
			msg = fmt.Sprintf("%s %s", msg, e.URL)
		}
		IBot.Cmd.Messagef(Config.IRCChannel, "%s no %s: %s", OriginString, channel.Name, strings.Replace(msg,"\n", " ",-1))
	})

	DBot.Identify.Intents = discordgo.IntentsGuildMessages // fodase copiei e colei

	// Try to join discord
	err = DBot.Open()
	if err != nil {
		fmt.Println("Can't talk to Discord!", err)
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	// Create a thread to handle shutting down
	go func() {
		<-sc
		fmt.Println("Shutting down...")
		DBot.Close()
		IBot.Close()
		jout, err := json.Marshal(av)
		if err != nil {
			fmt.Println("Can't translate avatar data to JSON!")
		}
		ioutil.WriteFile("av.json", jout, 0644)
	}()

	// Send IRC to Discord
	IBot.Handlers.Add(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {
		// repost channel messages from discord
		if e.IsFromChannel() {
			id := Config.DefaultHook
			if WhereUser[e.Source.Name] != "" {
				id = WhereUser[e.Source.Name]
			}
			// emoji replacement
			msg := e.Last()
			emojis, _ := DBot.GuildEmojis(Config.GuildID)
			msg = IEmojiRegex.ReplaceAllStringFunc(msg, func (input string) string {
				for _, e := range emojis {
					if fmt.Sprintf(":%s:", e.Name) == input {
						prefix := ""
						if e.Animated { prefix = "a" }
						return fmt.Sprintf("<%s:%s:%s>", prefix, e.Name, e.ID)
					}
				}
				return input
			})
			params := discordgo.WebhookParams{
				Username: e.Source.Name,
				Content: msg,
				AvatarURL: av[e.Source.Name].AvatarURL,
			}
			DBot.WebhookExecute(Config.Hooks[id].Id, Config.Hooks[id].Token, false, &params)
		// handle user commands from PMs
		} else {
			// get ready for wrong
			reply := "Canal indisponível. Opções são:"
			msg := e.Last()
			for name, _ := range Config.Hooks {
				if name == msg {
					WhereUser[e.Source.Name] = name
					c.Cmd.Messagef(e.Source.Name, "Você está postando em %s", name)
					return
				}
				reply = fmt.Sprintf("%s %s ;", reply, name)
			}
			c.Cmd.Message(e.Source.Name, reply)
		}
	})

	// Connect to IRC
	if err := IBot.Connect(); err != nil {
		fmt.Println("Can't connect to IRC!", err)
		return
	}
}
