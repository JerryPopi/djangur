package main

import (
	"os/exec"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

const (
	channels   int = 2     // 1 for mono, 2 for stereo
	frameRate  int = 48000 // audio sampling rate
	frameSize  int = 960   // uint16 size of each audio frame 960/48KHz = 20ms
	bufferSize int = 1024  // max size of opus data 1K
)

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

type Song struct {
	title    string
	url      string
	id       string
	duration string
}

func (v *VoiceInstance) AddQueue(song Song) {
	//todo

}

func (v *VoiceInstance) GetSong() Song {
	//todo
	return Song{}
}

func (v *VoiceInstance) PlayQueue(song Song) {
	v.AddQueue(song)

	if v.speaking {
		//bota govori!
		return
	}

	go func() {
		v.audioMutex.Lock()
		defer v.audioMutex.Unlock()
		for {
			if len(v.queue) == 0 {
				//NQQ PESNI SHEFE
				return
			}
			v.nowPlaying = v.GetSong()
			// go
		}
	}()
}
