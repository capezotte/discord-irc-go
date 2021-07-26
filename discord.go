package main

import (
	"fmt"; "log"
	"github.com/bwmarrin/discordgo"
	"strings"
	"regexp"
)

var DBot   *discordgo.Session
var DEmoji []*discordgo.Emoji

var DEmojiRegex *regexp.Regexp

// Formats message for IRC
func DMessageForIRC(m *discordgo.Message) string {
	msg := m.ContentWithMentionsReplaced()
	// add embeds
	for _, e := range m.Embeds {
		msg = fmt.Sprintf("%s %s", msg, e.URL)
	}
	// add attachments
	for _, e := range m.Attachments {
		msg = fmt.Sprintf("%s %s", msg, e.URL)
	}
	// Fix emoji
	msg = DEmojiRegex.ReplaceAllString(msg, "$1")
	// IRC doesn't like multiline
	msg = strings.Replace(msg,"\n", " ",-1)
	return msg
}

func DGetNick(u *discordgo.User) string {
	member, err := DBot.GuildMember(Config.GuildID, u.ID)
	if err == nil && member.Nick != "" {
		return member.Nick
	} else {
		return u.Username
	}

}

func DGetNickByID(ID string) string {
	u, err := DBot.User(ID)
	if err == nil {
		return DGetNick(u)
	}
	return "usuário desconhecido"
}

func DChannelName(ID string) string {
	channel, err := DBot.Channel(ID)
	if err == nil {
		return channel.Name
	}
	return "canal desconhecido"
}

func DOnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from ourselves
	for _, wh := range Config.Hooks { if wh.ID == m.WebhookID { return } }

	// Holds nick/user from poster, as well as who he replies to
	OriginString := DGetNick(m.Author)

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
			DestString = fmt.Sprintf("%s \"%-10s\"...", DGetNick(ReplyTo.Author), DMessageForIRC(ReplyTo))
		}
		OriginString = fmt.Sprintf("%s em resposta a %s", OriginString, DestString)
	}

	// Try to get channel name
	msg := DMessageForIRC(m.Message)
	IBot.Cmd.Messagef(Config.IRCChannel, "%s no %s: %s", OriginString, DChannelName(m.ChannelID), msg)
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
	_, err := DBot.WebhookExecute(Config.Hooks[id].ID, Config.Hooks[id].Token, false, &params)
	if err == nil {
		log.Printf("[DBot] Relayed message \"%s\" (from %s) to Discord", Im, Iu)
	} else {
		log.Printf("[DBot] Failed to relay \"%s\" to Discord (%s)", Im, err)
	}
}

func DOnGuildEmojisUpdate(s *discordgo.Session, e *discordgo.GuildEmojisUpdate) { DEmoji = e.Emojis }

func DRelayReaction(e discordgo.MessageReaction, verb string) {
	var err error
	m, err := DBot.ChannelMessage(e.ChannelID, e.MessageID)
	if err != nil {
		log.Printf("Can't log reaction: %s\n", err)
		return
	}
	uname := DGetNick(m.Author)
	IBot.Cmd.Messagef(Config.IRCChannel,"%s %s :%s: a \"%-10s\"... de %s no %s",
		DGetNickByID(e.UserID),
		verb,
		e.Emoji.Name,
		DMessageForIRC(m),
		uname,
		DChannelName(e.ChannelID),
	)
	log.Printf("[DBot] Relayed reaction %s (verb: %s) from %s to IRC", e.Emoji.Name, verb, uname)
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
	DBot.AddHandler(DOnGuildEmojisUpdate)
	DBot.AddHandler(func (s *discordgo.Session, e *discordgo.MessageReactionAdd) { DRelayReaction(*e.MessageReaction, "reagiu com") })
	DBot.AddHandler(func (s *discordgo.Session, e *discordgo.MessageReactionRemove) { DRelayReaction(*e.MessageReaction, "retirou a reação com") })

	DBot.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildEmojis | discordgo.IntentsGuildMessageReactions

	if DBot.Open() != nil {
		return fmt.Errorf("%s", "Can't open a socket with Discord")
	}

	DEmoji, err = DBot.GuildEmojis(Config.GuildID)
	if err != nil {
		fmt.Println("Failed to get emoji.")
	}
	return nil
}
