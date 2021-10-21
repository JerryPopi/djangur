package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// const (
// 	channels   int = 2     // 1 for mono, 2 for stereo
// 	frameRate  int = 48000 // audio sampling rate
// 	frameSize  int = 960   // uint16 size of each audio frame 960/48KHz = 20ms
// 	bufferSize int = 1024  // max size of opus data 1K
// )

type VoiceInstance struct {
	ginst *GuildInstance
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
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	v.queue = append(v.queue, song)
}

func (v *VoiceInstance) GetSong() (song Song) {
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	if len(v.queue) != 0 {
		return v.queue[0]
	}
	return
}

func (v *VoiceInstance) PopFromQueue(i int){
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	v.queue = v.queue[i:]
}

func (v *VoiceInstance) ClearQueue(){
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	v.queue = []Song{}
}

func (v *VoiceInstance) PlayQueue(song Song) {
	
	v.AddQueue(song)
	fmt.Println("STIGNA DO 1")
	if v.speaking {
		//bota govori!
		return
	}

	go func() {
	fmt.Println("STIGNA DO 2")

		v.audioMutex.Lock()
		defer v.audioMutex.Unlock()
	fmt.Println("STIGNA DO 3")

		for {
			if len(v.queue) == 0 {
				//NQQ PESNI SHEFE
				return
			}
			v.nowPlaying = v.GetSong()
			go v.ginst.Send("Now playing " + song.title)

			v.stop = false
			v.skip = false
			v.speaking = true
			v.pause = false
			v.voice.Speaking(true)
			fmt.Println("STIGNA DO 4")

			//tuka se puska nali
			// v.DCA(song.url)
			v.DCA("https://www.youtube.com/watch?v=felyOmO6liE")
			fmt.Println("STIGNA DO 5")

			v.PopFromQueue(1)
			if v.stop {
				v.ClearQueue()
			}
			v.stop = false
			v.skip = false
			v.speaking = false
			v.voice.Speaking(false)
		}
	}()
}

func (v *VoiceInstance) DCA(url string){
	dir, err := ioutil.TempDir("/tmp", "djangur")
	chk(err)
	defer os.RemoveAll(dir)
	cmd := exec.Command("yt-dlp", "--verbose", "-x", "--audio-format", "opus", "-o", dir + "/song.opus", url)
	cmd.Output()
	
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 128
	opts.Application = "lowdelay"
	fmt.Println(dir)
	encodeSession, err := dca.EncodeFile(dir + "/song.opus", opts)
	chk(err)
	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	for {
		select {
		case err := <- done:
			if err != nil && err != io.EOF {
				fmt.Println("FATAL: an error occured\n ", err)
			}
			fmt.Println("ZASHTO SPRQ BE")
			encodeSession.Cleanup()
			return
		}
	}
}

func (v *VoiceInstance) Stop(){
	v.stop = true
	if v.encoder != nil {
		v.encoder.Cleanup()
	}
}

func (v *VoiceInstance) Skip() bool {
	if v.speaking {
		if v.pause {
			return true
		} else {
			if v.encoder != nil {
				v.encoder.Cleanup()
			}
		}
	}
	return false
}

func (v *VoiceInstance) Pause() {
	v.pause = true
	if v.stream != nil {
		v.stream.SetPaused(true)
	}
}

func (v *VoiceInstance) Resume() {
	v.pause = false
	if v.stream != nil {
		v.stream.SetPaused(false)
	}
}
