package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

var guildFolders []string

type VoiceInstance struct {
	ginst      *GuildInstance
	voice      *discordgo.VoiceConnection
	encoder    *dca.EncodeSession
	stream     *dca.StreamingSession
	queueMutex sync.Mutex
	audioMutex sync.Mutex
	nowPlaying Song
	queue      []Song
	speaking   bool
	pause      bool
	stop       bool
	skip       bool
	folder     string
	timer      *time.Timer
}

type Song struct {
	title     string
	url       string
	id        string
	duration  float64
	thumbnail string
}

func TimeFormat(duration float64) string {
	hrs := math.Floor(duration / 3600)
	mins := math.Floor(math.Mod(duration, 3600) / 60)
	secs := math.Mod(math.Floor(duration), 60)

	var out string
	var minOut string
	var secOut string

	if mins < 10 {
		minOut = "0"
	} else {
		minOut = ""
	}

	if secs < 10 {
		secOut = "0"
	} else {
		secOut = ""
	}

	if hrs > 0 {
		out += "" + strconv.FormatFloat(hrs, 'f', 0, 64) + ":" + minOut
	}

	out += "" + strconv.FormatFloat(mins, 'f', 0, 64) + ":" + secOut
	out += "" + strconv.FormatFloat(secs, 'f', 0, 64)
	return out
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
		fmt.Printf("queue at getsong %+v\n", v.queue[0])
		return v.queue[0]
	}
	return
}

func (v *VoiceInstance) PopFromQueue(i int) {
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	v.queue = v.queue[i:]
}

func (v *VoiceInstance) ClearQueue() {
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	v.queue = []Song{}
}

func (v *VoiceInstance) ListQueue() {
	if len(v.queue) == 0 {
		emb := embed.Embed{MessageEmbed: embed.NewGenericEmbed("Queue empty!", "Use ?play to play a song.")}
		v.ginst.SendEmbed(*emb.MessageEmbed)
		return
	}

	emb := embed.Embed{MessageEmbed: embed.NewGenericEmbedAdvanced("Queue", "", 0x09b6e6)}

	var list string

	list = "```"

	for i, k := range v.queue {
		//fmt.Println(strconv.Itoa(i) + ": " + k.title)
		list += strconv.Itoa(i) + ": " + k.title + "\n"
	}

	list += "```"

	//emb.AddField("Songs", strconv.Itoa(i+1) + ": " + k.title)
	emb.AddField("Songs", list)

	v.ginst.SendEmbed(*emb.MessageEmbed)
}

func (v *VoiceInstance) DownloadSong(query string) (song Song) {
	if !strings.HasPrefix("https://", query) {
		query = "ytsearch:" + query
	}
	fmt.Println(query)
	if v.folder == "" {
		dir, err := ioutil.TempDir("/tmp", "djangur")
		chk(err)
		v.folder = dir
		guildFolders = append(guildFolders, dir)
	}

	fmt.Println("Downloading video...")
	cmd := exec.Command("yt-dlp", "--quiet", "-j", "--no-simulate", "-x", "--audio-format", "opus", "-o", v.folder+"/%(id)s.opus", query)
	out, err := cmd.Output()
	//fmt.Println(string(out))
	chk(err)

	fmt.Println("Video downloaded!")

	var video map[string]interface{}
	json.Unmarshal(out, &video)
	song.title = video["title"].(string)
	song.id = video["id"].(string)
	song.duration = video["duration"].(float64)
	song.thumbnail = video["thumbnails"].([]interface{})[0].(map[string]interface{})["url"].(string)
	song.url = "https://www.youtube.com/watch?v=" + song.id
	return song
}

func (v *VoiceInstance) PlayQueue(song Song) {
	v.AddQueue(song)
	if v.speaking {
		return
	}

	go func() {
		v.audioMutex.Lock()
		defer v.audioMutex.Unlock()
		for {
			if len(v.queue) == 0 {
				return
			}

			v.nowPlaying = v.GetSong()
			emb := embed.Embed{MessageEmbed: embed.NewGenericEmbedAdvanced(v.nowPlaying.title, v.nowPlaying.title, 0x09b6e6)}
			emb.SetThumbnail(v.nowPlaying.thumbnail)
			emb.SetURL(v.nowPlaying.url)
			dur := TimeFormat(v.nowPlaying.duration)
			emb.AddField("Duration", dur)
			emb.SetFooter("You have played this bruh times.")

			fmt.Println(v.nowPlaying.thumbnail)

			go v.ginst.SendEmbed(*emb.MessageEmbed)

			v.stop = false
			v.skip = false
			v.speaking = true
			v.pause = false
			v.voice.Speaking(true)

			v.AudioPlayer(v.nowPlaying)

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

func (v *VoiceInstance) AudioPlayer(song Song) {
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 128
	opts.Application = "lowdelay"

	encodeSession, err := dca.EncodeFile(v.folder+"/"+song.id+".opus", opts)
	chk(err)

	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream

	for {
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				fmt.Println("FATAL: an error occured\n ", err)
			}
			fmt.Println("End of track")
			encodeSession.Cleanup()
			return
		}
	}
}

func (v *VoiceInstance) Stop() {
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
