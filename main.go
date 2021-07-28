package main

import (
	"log"
	"os"; "os/signal"; "syscall"
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

func DotJsonToStruct(path string, st interface{}) error {
	log.Printf("Reading %s\n", path)
	content, _ := ioutil.ReadFile(path)
	err := json.Unmarshal([]byte(content), st)
	return err
}

func main() {
	wd, _ := os.Getwd()
	log.Printf("Starting bridge. Work directory is %s.\n", wd)

	var err error
	// Read config
	if err = DotJsonToStruct("cfg.json", &Config); err != nil { log.Fatalln("Absent or invalid cfg.json!", err) }

	// Read previous avatar data
	if err = DotJsonToStruct("av.json", &DNameToID); err != nil {
		log.Printf("Can't read data: %s, starting from scratch.", err)
		DNameToID = make(map[string]DCachedUserData)
	}

	heavy := make(chan os.Signal, 1)
	signal.Notify(heavy, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go func() {
		<-heavy
		// oh no i'm dead
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
	DInit(); IInit()
}
