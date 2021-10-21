package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var ginsts map[string]GuildInstance

type GuildInstance struct {
	v *VoiceInstance
	s *discordgo.Session
	g *discordgo.Guild
	lastChannel string
}

func (g *GuildInstance) Send(msg string){
	if g.lastChannel == "" {
		return
	}
	g.s.ChannelMessageSend(g.lastChannel, msg)
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func getUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate) (string) {
	guild, err := s.State.Guild(m.GuildID)
	chk(err)

	for _, vs := range guild.VoiceStates {
		if m.Author.ID == vs.UserID {
			return vs.ChannelID
		}
	}

	return ""
}

func getGinst(s *discordgo.Session, m *discordgo.MessageCreate) (ginst *GuildInstance) {
	if ginst, ok := ginsts[m.GuildID]; !ok {
		guild, err := s.State.Guild(m.GuildID)
		chk(err)
		ginsts[m.GuildID] = GuildInstance {
			v: &VoiceInstance{ginst: &ginst}, 
			s: s, 
			g: guild,
			lastChannel: m.ChannelID,
		}
		ginst := ginsts[m.GuildID]
		return &ginst	
	} else {
		return &ginst
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "?") {
		return
	}
	ginst := getGinst(s, m)
	m.Content = m.Content[1:]
	split := strings.SplitN(m.Content, " ", 2)
	cmd  := split[0]
	arg := ""
	if len(split) == 2 {
		arg = split[1]
		fmt.Println(arg)
	}

	switch cmd {
	case "ping":
		ginst.Send("pong")
	case "play":
		vc := getUserVoiceChannel(s, m)
		if vc == "" {
			ginst.Send("User not in voice channel!")
			return
		}
		dgv, err := s.ChannelVoiceJoin(m.GuildID, vc, false, true)
		ginst.v.voice = dgv
		chk(err)
		// song := songSearch(arg)
		// ginst.v.PlayQueue(song)
		ginst.v.AudioPlayer(arg)
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

	ginsts = make(map[string]GuildInstance)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
