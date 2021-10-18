package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

const (
	channels   int = 2     // 1 for mono, 2 for stereo
	frameRate  int = 48000 // audio sampling rate
	frameSize  int = 960   // uint16 size of each audio frame 960/48KHz = 20ms
	bufferSize int = 1024  // max size of opus data 1K
)

type Song struct {
	title string
	url string
	id string
	duration string
}

type VoiceInstance struct {
	voice      *discordgo.VoiceConnection
	session    *discordgo.Session
	encoder    *dca.EncodeSession
	stream     *dca.StreamingSession
	run        *exec.Cmd
	queueMutex sync.Mutex
	audioMutex sync.Mutex
	nowPlaying Song
	queue      []Song
	recv       []int16
	guildID    string
	channelID  string
	speaking   bool
	pause      bool
	stop       bool
	skip       bool
	radioFlag  bool
}

func chk(err error){
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

func getUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate) (string){
	guild, err := s.State.Guild(m.GuildID)
	chk(err)

	for _, vs := range guild.VoiceStates {
		if m.Author.ID == vs.UserID {
			return vs.ChannelID
		}
	}

	return ""
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
		vc := getUserVoiceChannel(s, m)
		if vc == "" {
			s.ChannelMessageSend(m.ChannelID, "User not in voice channel!")
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
