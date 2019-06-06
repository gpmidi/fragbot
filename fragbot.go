package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	bot      botConfig
	chn      channelConfig
	users    players
	platfms  platforms
	shutdown = make(chan string)
	//ServStat is the Service Status channel
	servStat = make(chan string)
)

type botConfig struct {
	Token string `json:"token"`
	Game  string `json:"game,omitempty"`
}

type channelConfig struct {
	Prefix string  `json:"prefix,omitempty"`
	LFG    lfgInfo `json:"lfg,omitempty"`
	RTD    rtdInfo `json:"rtd,omitempty"`
}

type discordCodeBlock struct {
	Header  string
	Message []string
	Footer  string
}

func init() {
	log.Printf("loading configs from files\n")
	log.Printf("loading bot config")
	err := loadInfo("config.json", &bot)
	if err != nil {
		log.Fatalf("there was an issue reading config file\n")
	}

	log.Printf("loading channel config")
	err = loadInfo("channel.json", &chn)
	if err != nil {
		log.Fatalf("there was an issue reading config file\n")
	}

	log.Printf("loading channel config")
	err = loadInfo("channel.json", &chn)
	if err != nil {
		log.Fatalf("there was an issue reading config file\n")
	}

	log.Printf("all configs loaded")

	if chn.LFG.ChannelID != "" {
		lookingForGroupInit()
	}

	if chn.RTD.ChannelID != "" {
		rollTheDiceInit()
	}
}

func main() {
	// start discord handling
	go startDiscordHandler()

	<-servStat

	log.Printf("Bot is now running.  Press CTRL-C send 'shutdown' to exit.")
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("cannot read from stdin: %s", err)
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			log.Printf("command not specified")
			continue
		}
		if line == "shutdown" {
			log.Printf("shutting down the bot.\n")
			shutdown <- "stop"
			<-shutdown
			return
		}
	}
}

func startDiscordHandler() {
	log.Printf("starting discord connections\n")

	// Initializing Discord connection
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + bot.Token)
	if err != nil {
		log.Fatalf("error creating Discord session, %s", err)
		return
	}

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(handleDiscordMessages)

	// Register ready as a callback for the ready events
	dg.AddHandler(readyDiscord)

	err = dg.Open()
	if err != nil {
		log.Fatalf("error opening connection, %s", err)
		return
	}
	log.Printf("discord service connected\n")

	bot, err := dg.User("@me")
	if err != nil {
		log.Printf("error obtaining account details, %s", err)
	}

	log.Printf("invite the bot to your server with https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot", bot.ID)

	servStat <- "discord_online"

	// ticker for lfg checks
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			// log.Printf("fifteen seconds have passed and all is well")
			// look for expired users on startup.
			response, discordUserID, send := lookingForGroupTickJob()
			if send {
				sendDiscordDirectMessage(dg, discordUserID, response)
			}
		}
	}()

	<-shutdown
	dg.Close()
	shutdown <- "stopped"
}

// discord handlers
func handleDiscordMessages(s *discordgo.Session, message *discordgo.MessageCreate) {
	messageContent := strings.ToLower(message.Message.Content)
	// ignore all bot messages
	if message.Author.Bot {
		log.Printf("user is a bot and being ignored.")
		return
	}

	// get channel info
	channel, err := s.Channel(message.ChannelID)
	if err != nil {
		log.Printf("error getting channel info, %s", err)
		return
	}

	if channel.Type == 1 {
		return
	}
	if strings.HasPrefix(messageContent, chn.Prefix) {
		var response string
		var sendToDM bool
		if strings.HasPrefix(messageContent, chn.Prefix+"lfg") && message.ChannelID == chn.LFG.ChannelID {
			if strings.TrimPrefix(messageContent, chn.Prefix+"lfg") == "" || !strings.HasPrefix(messageContent, chn.Prefix+"lfg ") {
				sendDiscordMessage(s, channel.ID, "How to use Fragfinder v1.0\n`!lfg (Game) (Platform) (Wait Time in minutes (Default is 60 if not set))\nI.E. `!lfg Rocket League PS4 60`")
				return
			}
			response, sendToDM = lookingForGroup(strings.TrimPrefix(messageContent, chn.Prefix+"lfg "), message.Author.ID, message.Author.Username)
		}
		if strings.HasPrefix(messageContent, chn.Prefix+"roll") && message.ChannelID == chn.RTD.ChannelID {
			if strings.TrimPrefix(messageContent, chn.Prefix+"roll") == "" || !strings.HasPrefix(messageContent, chn.Prefix+"roll ") {
				sendDiscordMessage(s, channel.ID, "How to use Roll the Dice\n`!roll (dice)d(sides)[+/-][proficiency]`\nI.E. `!roll 1d20+3`")
				return
			}
			response, sendToDM = rollTheDice(strings.TrimPrefix(messageContent, chn.Prefix+"roll "))
		}
		if sendToDM {
			sendDiscordDirectMessage(s, message.Author.ID, response)
		} else {
			sendDiscordMessage(s, channel.ID, response)
		}
	}
}

func readyDiscord(s *discordgo.Session, event *discordgo.Ready) {
	err := s.UpdateStatus(0, bot.Game)
	if err != nil {
		log.Printf("error setting game: %s", err)
		return
	}
	log.Printf("set game to: %s", bot.Game)
}

func handleDiscordGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}
}

func sendDiscordMessage(s *discordgo.Session, channelID string, response string) {
	s.ChannelMessageSend(channelID, response)
}

func sendDiscordDirectMessage(s *discordgo.Session, discordUserID string, response string) {
	log.Printf("sending message '%s' to %s", response, discordUserID)
	channel, err := s.UserChannelCreate(discordUserID)
	if err != nil {
		log.Printf("error creating direct message channel: %s", err)
		return
	}
	sendDiscordMessage(s, channel.ID, response)
}

func sendDiscordEmbed(s *discordgo.Session, embed *discordgo.MessageEmbed, channelID string) {
	log.Printf("sending embed to %s", channelID)
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("error creating direct message channel: %s", err)
		return
	}
}

func newCodeBlock() discordCodeBlock {
	return discordCodeBlock{"```\n", []string{}, "```\n"}
}

// helper functions
func contains(a []string, s string) bool {
	for _, n := range a {
		if s == n {
			return true
		}
	}
	return false
}

// File management
func writeJSONToFile(jdata []byte, file string) error {
	log.Printf("updating file %s", file)
	// create a file with a supplied name
	jsonFile, err := os.Create(file)
	if err != nil {
		return err
	}
	_, err = jsonFile.Write(jdata)
	if err != nil {
		return err
	}

	return nil
}

func readJSONFromFile(file string) ([]byte, error) {
	if !strings.HasSuffix(file, ".json") {
		return nil, errors.New("the file requested is not a json file")
	}

	// Open our jsonFile
	// log.Printf("opening json file\n")
	jsonFile, err := os.Open(file)

	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}

	// log.Printf("holding file open\n")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// log.Printf("reading file\n")
	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	// return the json byte value.
	return byteValue, nil
}

func doesExist(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

func loadInfo(file string, v interface{}) error {
	if !strings.HasSuffix(file, ".json") {
		return fmt.Errorf("the file specified was not a json file")
	}

	if !doesExist(file) {
		log.Printf("%s does not exist creating it", file)
		jsonFile, err := os.Create(file)
		if err != nil {
			return fmt.Errorf("there was an error loading the file")
		}
		_, err = jsonFile.Write([]byte("{}"))
		if err != nil {
			return err
		}
	}

	// Open our jsonFile
	jsonFile, err := os.Open(file)
	if err != nil {
		return err
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &v)
	if err != nil {
		return err
	}

	return nil
}

func saveInfo(file string, v interface{}) error {
	// log.Printf("converting struct data to bytesfor %s", file)
	bytes, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return fmt.Errorf("there was an error converting the user data to json")
	}

	// log.Printf("writing bytes to file")
	if err := writeJSONToFile(bytes, file); err != nil {
		return err
	}

	return nil
}
