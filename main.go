package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var counter = 0

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	discordToken := os.Getenv("DISCORD_TOKEN")

	discord, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("error creating Discord session, ", err)
		return
	}

	discord.AddHandler(messageCreate)

	discord.Identify.Intents = discordgo.IntentGuildMessages

	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection, ", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}

func messageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.ID == s.State.User.ID {
		return
	}

	cmd := strings.Split(msg.Content, " ")

	switch cmd[0] {
	case "ping":
		s.ChannelMessageSend(msg.ChannelID, "Pong!")
	case "avatar":
		author := msg.Author
		avatarImg, err := s.UserAvatarDecode(author)
		if err != nil {
			fmt.Println("error loading avatar image")
			return
		}

		buf := new(bytes.Buffer)
		err = png.Encode(buf, avatarImg)
		if err != nil {
			fmt.Println("error encoding avatar image")
			return
		}

		file := &discordgo.File{Name: "avatar.png", ContentType: "image/png", Reader: buf}
		s.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{Files: []*discordgo.File{file}})
	case "avatarUrl":
		avatarUrl := msg.Author.AvatarURL("2048")
		s.ChannelMessageSend(msg.ChannelID, avatarUrl)

	case "c":
		s.UpdateGameStatus(0, fmt.Sprint(counter))
		counter++
	case "todo":
		var todoIndex uint64
		if len(cmd) > 1 {
			todoIndex, _ = strconv.ParseUint(cmd[1], 10, 64)
		}

		res, err := http.Get("https://jsonplaceholder.typicode.com/todos")
		if err != nil {
			fmt.Println("error getting json")
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println("error reading body")
			return
		}

		var todos []interface{}
		json.Unmarshal(body, &todos)

		if uint64(len(todos)) < todoIndex {
			return
		}
		todo, err := json.MarshalIndent(todos[todoIndex], "", " ")
		if err != nil {
			fmt.Println("error marshalling todos")
			return
		}

		_, err = s.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("```json\n%s```", todo))
		if err != nil {
			fmt.Printf("error sending message to channel %s: %s\n", msg.ChannelID, err)
		}
	}
}
