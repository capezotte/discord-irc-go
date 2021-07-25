package main

import (
	"fmt"; "log"
	"github.com/bwmarrin/discordgo"
	"strings"
	"regexp"
)

var DBot     *discordgo.Session
var DEmoji []*discordgo.Emoji

var DEmojiRegex *regexp.Regexp


func DOnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from ourselves
	for _, wh := range Config.Hooks { if wh.ID == m.WebhookID { return } }

	// Holds nick/user from poster, as well as who he replies to
	var OriginString string

	// Try to get nickname
	member, err := DBot.GuildMember(m.GuildID, m.Author.ID)
	if err == nil && member.Nick != "" {
		OriginString = member.Nick
	} else {
		OriginString = m.Author.Username
	}

	// Cache UID and avatar for mentions and webhook
	aurl := m.Author.AvatarURL("")
	log.Printf("Caching data for user %s: ID %s, URL: %s\n", OriginString, m.Author.ID, aurl)
	DNameToID[OriginString] = DCachedUserData{
		ID: m.Author.ID,
		AvatarURL: aurl,
	}

	// Handle replying
	if m.MessageReference != nil {
		var DestString string
		ReplyTo, err := DBot.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
		if err == nil {
			// Try to get nick of person being replied to
			replymember, err  := DBot.GuildMember(m.MessageReference.GuildID, ReplyTo.Author.ID)
			if err == nil && replymember.Nick != "" {
				DestString = replymember.Nick
			} else {
				DestString = ReplyTo.Author.Username
			}
		} else {
			DestString = "uma mensagem apagada"
		}
		OriginString = fmt.Sprintf("%s em resposta a %s", OriginString, DestString)
	}

	// message
	msg := m.ContentWithMentionsReplaced()
	// add embeds
	for _, e := range m.Embeds {
		msg = fmt.Sprintf("%s %s", msg, e.URL)
	}
	// add attachments
	for _, e := range m.Attachments {
		msg = fmt.Sprintf("%s %s", msg, e.URL)
	}

	// Try to get channel name
	channel, err := DBot.Channel(m.ChannelID)
	if err != nil { return }
	msg = DEmojiRegex.ReplaceAllString(msg, "$1")
	msg = strings.Replace(msg,"\n", " ",-1)
	IBot.Cmd.Messagef(Config.IRCChannel, "%s no %s: %s", OriginString, channel.Name, msg)
	log.Printf("[DBot] Relayed message \"%s\" (from %s) to IRC", msg, m.Author.Username)
}

func DIRCMessage(Iu string, Im string) {
	// Select channel
	id := Config.DefaultHook
	if IUserDChannels[Iu] != "" {
		id = IUserDChannels[Iu]
	}
	Im = IMentionRegex.ReplaceAllStringFunc(Im, IReplMention)
	Im = IEmojiRegex.ReplaceAllStringFunc(Im, IReplEmoji)
	params := discordgo.WebhookParams{
		Username: Iu,
		Content: Im,
		AvatarURL: DNameToID[Iu].AvatarURL,
	}
	DBot.WebhookExecute(Config.Hooks[id].ID, Config.Hooks[id].Token, false, &params)
	log.Printf("[DBot] Relayed message \"%s\" (from %s) to Discord", Im, Iu)
}

func DInit() error {
	var err error

	// Regexps
	DEmojiRegex = regexp.MustCompile("<a?(:[[:alnum:]_]+:)[[:alnum:]]+>")

	DBot, err = discordgo.New("Bot " + Config.Token)

	if err != nil {
		return fmt.Errorf("%s", "Failed to create a discord session.")
	}

	DBot.AddHandler(DOnMessageCreate)

	DBot.Identify.Intents = discordgo.IntentsGuildMessages // fodase copiei e colei

	if DBot.Open() != nil {
		return fmt.Errorf("%s", "Can't open a socket with Discord")
	}

	DEmoji, err = DBot.GuildEmojis(Config.GuildID)
	if err != nil {
		fmt.Println("Failed to get emoji.")
	}
	return nil
}
