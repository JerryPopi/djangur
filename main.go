package main

import (
	"fmt"
	"strings"
	"os"
	"os/signal"
	"syscall"
	"flag"
	"github.com/bwmarrin/discordgo"
)

func chk(err error){
	if err != nil {
		panic(err)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "?") {
		return
	}
	m.Content = m.Content[1:]

	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "pong")
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
