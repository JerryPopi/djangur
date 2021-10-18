package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	// "github.com/jonas747/dca"
)

var ginsts map[string]GuildInstance

type GuildInstance struct {
	v VoiceInstance
	s *discordgo.Session
	g discordgo.Guild
	lastChannel string
}

func (g GuildInstance) Send(msg string){
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

func songSearch(query string) (song Song) {
	if !strings.HasPrefix(query, "https:") {
		query = "ytsearch:" + query
	}
	println("Downloading video...")
	cmd := exec.Command("youtube-dl", "-j", query)
	stdout, err := cmd.Output()
	chk(err)

	var video map[string]interface{}
	json.Unmarshal(stdout, &video)

	if video["formats"] != nil {
		formats := video["formats"].([]interface{})
		format  := formats[0].(map[string]interface{})
		fmt.Printf("%+v\n", formats)
		song.url = format["url"].(string)
		// song.duration = format["duration"].(string)
		// song.title = format["title"].(string)
		// song.id = format["id"].(string)
	} else if video["url"] != nil {
		song.url = video["url"].(string)
	}
	println("Done downloading video!")
	return song
}

func getUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate) (string) {
	ginst := getGinst(s, m)

	for _, vs := range ginst.g.VoiceStates {
		if m.Author.ID == vs.UserID {
			return vs.ChannelID
		}
	}

	return ""
}

func getGinst(s *discordgo.Session, m *discordgo.MessageCreate) GuildInstance {
	if ginst, ok := ginsts[m.GuildID]; !ok {
		ginsts[m.GuildID] = GuildInstance {
			v: VoiceInstance{}, 
			s: s, 
			lastChannel: m.ChannelID,
		}
		return ginsts[m.GuildID]
	} else {
		return ginst
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
		chk(err)
		song := songSearch(arg)
		dgvoice.PlayAudioFile(dgv, song.url, make(chan bool))
		fmt.Println(song)
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
