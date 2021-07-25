package main

import (
	"log"
	"os"; //"os/signal"; "syscall"
	"github.com/lrstanley/girc"
	"encoding/json"
	"io/ioutil"
)

type DCachedUserData struct {
	ID string
	AvatarURL string
}
var DNameToID map[string]DCachedUserData

type Settings struct {
	Token string
	IRCSettings girc.Config
	DefaultHook string
	Hooks map[string]struct{ ID string; Token string }
	GuildID string
	IRCChannel string
}
var Config Settings

func main() {
	wd, _ := os.Getwd()
	log.Printf("Starting bridge. Work directory is %s.\n", wd)

	// Read config
	log.Println("Importing settings from cfg.json.")
	content, _ := ioutil.ReadFile("cfg.json")
	var err error = json.Unmarshal([]byte(content), &Config)
	if err != nil { log.Fatalln("Absent or invalid cfg.json!", err) }

	// Read previous avatar data
	log.Println("Importing previous avatar data from av.json.")
	avatars, _ := ioutil.ReadFile("av.json")
	err = json.Unmarshal([]byte(avatars), &DNameToID)
	if err != nil {
		log.Println("Can't read data, starting from scratch.")
		DNameToID = make(map[string]DCachedUserData)
	}

	sc := make(chan os.Signal, 1)
	go func() {
		<-sc
		log.Println("Shutting down...")
		DBot.Close()
		IBot.Close()
		jout, err := json.Marshal(DNameToID)
		if err != nil {
			log.Fatal("Can't translate avatar data to JSON!")
		}
		ioutil.WriteFile("av.json", jout, 0644)
	}()

	// Instance bots
	if DInit() != nil || IInit() != nil {
		panic("One of the bots failed to initialize!")
	}
}
