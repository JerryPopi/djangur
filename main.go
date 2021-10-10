package main

import (
	"fmt"
	"strings"
	"os"
	"os/signal"
	"os/exec"
	"syscall"
	"flag"
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

type Song struct {
	url string
}

func chk(err error){
	if err != nil {
		panic(err)
	}
}

func songSearch(query string) (song Song) {
	cmd := exec.Command("youtube-dl", "-j", query)
	stdout, err := cmd.Output()
	chk(err)

	var video map[string]interface{}
	json.Unmarshal(stdout, &video)

	if video["formats"] != nil {
		formats := video["formats"].([]interface{})
		format  := formats[0].(map[string]interface{})
		song.url = format["url"].(string)
	} else if video["url"] != nil {
		song.url = video["url"].(string)
	}

	return song
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "?") {
		return
	}
	m.Content = m.Content[1:]
	split := strings.SplitN(m.Content, " ", 2)
	cmd  := split[0]
	arg := ""
	if len(split) == 2 {
		arg = split[1]
	}

	switch cmd {
	case "ping":
		s.ChannelMessageSend(m.ChannelID, "pong")
	case "play":
		song := songSearch(arg)
		fmt.Println(song.url)
	}
}

func main() {
	var token string

	flag.StringVar(&token, "token", "", "Discord bot token")
	flag.Parse()

	dg, err := discordgo.New("Bot " + token)
	chk(err)

	dg.AddHandler(messageCreate)

	err = dg.Open()
	chk(err)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
