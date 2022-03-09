package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	// "io"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	embed "github.com/Clinet/discordgo-embed"
	// "github.com/bwmarrin/dgvoice"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

var guildFolders []string

type VoiceInstance struct {
	ginst              *GuildInstance
	voice              *discordgo.VoiceConnection
	encoder            *dca.EncodeSession
	stream             *dca.StreamingSession
	queueMutex         sync.Mutex
	audioMutex         sync.Mutex
	nowPlaying         Song
	queue              []Song
	loop               uint8
	speaking           bool
	pause              bool
	stop               bool
	skip               bool
	// folder             string
	timeStarted        int64
	pausedAndResumedTS []int64
	pausedTime         int
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
		// fmt.Printf("queue at getsong %+v\n", v.queue[0])
		return v.queue[0]
	}
	return
}

func (v *VoiceInstance) PopFromQueue(i int) Song {
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	//v.queue = v.queue[i:]
	i = i - 1
	el := v.queue[i]
	v.queue = append(v.queue[:i], v.queue[i+1:]...)
	return el
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
		list += strconv.Itoa(i) + ": " + k.title + "\n"
	}

	list += "```"

	emb.AddField("Songs", list)

	v.ginst.SendEmbed(*emb.MessageEmbed)
}

func (v *VoiceInstance) NowPlaying() {
	if len(v.queue) == 0 {
		emb := embed.Embed{MessageEmbed: embed.NewGenericEmbed("Queue empty!", "Use ?play to play a song.")}
		v.ginst.SendEmbed(*emb.MessageEmbed)
		return
	}
	emb := embed.Embed{MessageEmbed: embed.NewGenericEmbedAdvanced(v.nowPlaying.title, "", 0x09b6e6)}
	emb.SetURL(v.nowPlaying.url)
	emb.SetThumbnail(v.nowPlaying.thumbnail)

	for i := 0; i < len(v.pausedAndResumedTS); i += 2 {
		fmt.Println(int(v.pausedAndResumedTS[i]))
		if i+1 < len(v.pausedAndResumedTS) {

			v.pausedTime = int(v.pausedAndResumedTS[i+1]) - int(v.pausedAndResumedTS[i])
			fmt.Println(int(v.pausedAndResumedTS[i+1]) - int(v.pausedAndResumedTS[i]))
			fmt.Println("---")
		} else {
			fmt.Println("laina")
			// v.pausedAndResumedTS = append(v.pausedAndResumedTS, time.Now().Unix())
			v.pausedTime = int(time.Now().Unix()) - int(v.pausedAndResumedTS[i])
		}
	}

	fmt.Println(v.pausedAndResumedTS)
	fmt.Println(v.pausedTime)
	timeDifference := (time.Now().Unix() - int64(v.pausedTime)) - v.timeStarted
	fmt.Println(v.timeStarted)
	fmt.Println(timeDifference)
	displayTimestamp := int(math.Round((float64(timeDifference) / v.nowPlaying.duration) * 30))
	displayTimestampEmoji := ""
	for i := 0; i < 30; i++ {
		if i == displayTimestamp {
			displayTimestampEmoji += "🔴"
		} else {
			displayTimestampEmoji += "▬"
		}
	}
	emb.AddField("`"+displayTimestampEmoji+"`", "`"+TimeFormat(float64(timeDifference))+"/"+TimeFormat(v.nowPlaying.duration)+"`")

	v.ginst.SendEmbed(*emb.MessageEmbed)
}

func (v *VoiceInstance) DownloadSong(query string) (*Song, error) {
	if query == "" {
		emb := embed.Embed{MessageEmbed: embed.NewGenericEmbed("Nothing to search for!", "")}
		v.ginst.SendEmbed(*emb.MessageEmbed)
		return new(Song), errors.New("no-query")
	}
	if !strings.HasPrefix("https://", query) {
		query = "ytsearch:" + query
	}
	fmt.Println(query)

	// var video map[string]interface{}
	// json.Unmarshal(out, &video)
	// song := new(Song)
	// song.title = video["title"].(string)
	// song.id = video["id"].(string)
	// song.duration = video["duration"].(float64)
	// song.thumbnail = video["thumbnails"].([]interface{})[0].(map[string]interface{})["url"].(string)
	// song.url = "https://www.youtube.com/watch?v=" + song.id
	// return song, nil


	// NEW IMPLEMENTATION (OLD IMPLEMENTATION)
	cmd := exec.Command("yt-dlp", "--print", "\"%()j\"", "-x", "--get-duration", query)
	stdout, err := cmd.Output()
	chk(err)

	err = os.WriteFile("temp.json", stdout, 0644)
	chk(err)

	output := strings.Split(string(stdout), "\n")

	var video map[string]interface{}
	json.Unmarshal([]byte(output[0]), &video)
	song := new(Song)

	if video["formats"] != nil {
		formats := video["formats"].([]interface{})
		for _, k := range formats {
			format := k.(map[string]interface{})
			if !strings.Contains(format["format_id"].(string), "sb") {
				song.url = format["url"].(string)
				break
			}
		}
	} else if video["url"] != nil {
		song.url = video["url"].(string)
	}

	// song.duration, err = strconv.ParseFloat(output[1], 64)
	// chk(err)
	println("TIMESTAMP: " + output[1])
	time, _ := time.ParseDuration(output[1])
	println("TIME: " + time.String())

	return song, nil
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
			v.timeStarted = time.Now().Unix()
			v.pausedTime = 0

			v.AudioPlayer(v.nowPlaying)

			if v.loop != 1 {
				v.PopFromQueue(1)
			}

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

// func (v *VoiceInstance) AudioPlayer(song Song) {
// 	opts := dca.StdEncodeOptions
// 	opts.RawOutput = true
// 	opts.Bitrate = 128
// 	opts.Application = "lowdelay"

// 	// encodeSession, err := dca.EncodeFile(v.folder+"/"+song.id+".opus", opts)
// 	encodeSession, err := dca.EncodeFile(song.url, opts)
// 	chk(err)

// 	v.encoder = encodeSession
// 	done := make(chan error)
// 	stream := dca.NewStream(encodeSession, v.voice, done)
// 	v.stream = stream

// 	for {
// 		select {
// 		case err := <-done:
// 			if err != nil && err != io.EOF {
// 				fmt.Println("FATAL: an error occured\n ", err)
// 			}
// 			fmt.Println("End of track")
// 			encodeSession.Cleanup()
// 			return
// 		}
// 	}
// }

func (v *VoiceInstance) AudioPlayer(song Song) {
	PlayAudioFile(v.voice, song.url, make(chan bool))
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
	if !v.pause {
		v.pausedAndResumedTS = append(v.pausedAndResumedTS, time.Now().Unix())
	}
	v.pause = true
	if v.stream != nil {
		v.stream.SetPaused(true)
	}
}

func (v *VoiceInstance) Resume() {
	if v.pause {
		// v.pausedAndResumedTS = v.pausedAndResumedTS[:len(v.pausedAndResumedTS)-1]
		v.pausedAndResumedTS = append(v.pausedAndResumedTS, time.Now().Unix())
	}
	v.pause = false
	if v.stream != nil {
		v.stream.SetPaused(false)
	}
}
