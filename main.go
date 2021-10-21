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

	// "github.com/bwmarrin/dgvoice"
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

func songSearch(query string) (song Song) {
	if !strings.HasPrefix(query, "https:") {
		query = "ytsearch:" + query
	}
	println("Downloading video...")
	cmd := exec.Command("youtube-dl", "-j", query)
	stdout, err := cmd.Output()
	cmd.Run()
	chk(err)

	var video map[string]interface{}
	json.Unmarshal(stdout, &video)

	if video["formats"] != nil {
		formats := video["formats"].([]interface{})
		format  := formats[0].(map[string]interface{})
		// fmt.Printf("%+v\n", formats)
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
	// ginst := getGinst(s, m)

	guild, err := s.State.Guild(m.GuildID)
	chk(err)

	// fmt.Printf("%+v\n", ginst)

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
		song := songSearch(arg)
		// PlayAudioFile(dgv, song.url, make(chan bool))
		ginst.v.PlayQueue(song)
		// fmt.Println(song)
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
