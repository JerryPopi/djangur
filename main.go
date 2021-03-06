package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var ginsts map[string]GuildInstance

type GuildInstance struct {
	v           *VoiceInstance
	s           *discordgo.Session
	g           *discordgo.Guild
	lastChannel string
}

func (g *GuildInstance) Send(msg string) {
	if g.lastChannel == "" {
		fmt.Println("lastchannel is empty")
		return
	}
	g.s.ChannelMessageSend(g.lastChannel, msg)
}

func (g *GuildInstance) SendEmbed(emb discordgo.MessageEmbed) {
	if g.lastChannel == "" {
		return
	}
	g.s.ChannelMessageSendEmbed(g.lastChannel, &emb)
}

func chk(err error) {
	if err != nil {
		for _, k := range guildFolders {
			os.RemoveAll(k)
		}
		panic(err)
	}
}

func getUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate) string {
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
		ginsts[m.GuildID] = GuildInstance{
			v:           &VoiceInstance{ginst: &ginst, timeStarted: time.Now().Unix()},
			s:           s,
			g:           guild,
			lastChannel: m.ChannelID,
		}
		ginst := ginsts[m.GuildID]
		ginst.v.ginst = &ginst

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
	cmd := split[0]
	arg := ""
	if len(split) == 2 {
		arg = split[1]
		fmt.Println(arg)
	}

	switch cmd {
	case "ping":
		ginst.Send("pong")
	case "play", "p":
		if ginst.v.pause && arg == "" {
			ginst.v.Resume()
			return
		}

		vc := getUserVoiceChannel(s, m)
		if vc == "" {
			ginst.Send("User not in voice channel!")
			return
		}
		dgv, err := s.ChannelVoiceJoin(m.GuildID, vc, false, true)
		ginst.v.voice = dgv
		chk(err)
		var song* Song
		if len(m.Attachments) > 0 {
			song, err = ginst.v.DownloadSong(m.Attachments[0].URL)
		} else {
			song, err = ginst.v.DownloadSong(arg)
		}

		// song, err := ginst.v.DownloadSong(arg)
		if err != nil {
			return
		}
		ginst.v.PlayQueue(*song)
	case "skip", "s":
		ginst.v.Skip()
	case "pause":
		ginst.v.Pause()
	case "resume":
		ginst.v.Resume()
	case "stop":
		ginst.v.Stop()
	case "queue", "q":
		ginst.v.ListQueue()
	case "remove", "r":
		if arg == "" {
			ginst.Send("Invalid remove index.")
			return
		}
		i, _ := strconv.Atoi(arg)
		removed := ginst.v.PopFromQueue(i)
		ginst.Send("Removed " + removed.title)
	case "nowplaying", "np":
		ginst.v.NowPlaying()
	case "loop", "l":
		switch ginst.v.loop {
		case 0:
			ginst.v.loop = 1
			ginst.Send("Looping current track.")
		case 1:
			ginst.v.loop = 2
			ginst.Send("Looping entire queue.")
		case 2:
			ginst.v.loop = 0
			ginst.Send("Stopped looping.")
		}
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

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	for _, k := range guildFolders {
		os.RemoveAll(k)
	}

	dg.Close()
}
