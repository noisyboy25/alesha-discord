package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var counter = 0
var discord *discordgo.Session
var discordToken string

type Todo struct {
	Completed bool   `json:"completed"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	UserID    int    `json:"userId"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("error loading .env file")
	}
	discordToken = os.Getenv("DISCORD_TOKEN")
}

func init() {
	var err error
	discord, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func init() {
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "basic-command",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Basic command",
		},
		{
			Name:        "avatar",
			Description: "Show your avatar",
		},
		{
			Name:        "todo",
			Description: "Get one todo",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "todo-id",
					Description: "Todo ID",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"basic-command": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Hey there! Congratulations, you just executed your first slash command",
				},
			})
		},
		"avatar": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			content := ""
			if i.Interaction.Member != nil {
				content = i.Interaction.Member.AvatarURL("2048")
			} else if i.Interaction.User != nil {
				content = i.Interaction.User.AvatarURL("2048")
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
		"todo": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			content := ""
			url := fmt.Sprintf("https://jsonplaceholder.typicode.com/todos/%d", i.ApplicationCommandData().Options[0].IntValue())

			res, err := http.Get(url)
			if err != nil {
				log.Println("error getting json")
				return
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println("error reading body")
			}

			var todo Todo
			err = json.Unmarshal(body, &todo)
			if err != nil {
				log.Println("error unmarshalling json")
			} else {
				status := ""
				if todo.Completed {
					status = "Completed"
				} else {
					status = "InProgress"
				}
				content = fmt.Sprintf("**%s**\nstatus: %s", todo.Title, status)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}
)

func init() {
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	discord.AddHandler(messageCreate)

	err := discord.Open()
	if err != nil {
		log.Println("error opening connection, ", err)
		return
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer discord.Close()

	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func messageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.ID == s.State.User.ID {
		return
	}

	cmd := strings.Split(msg.Content, " ")

	log.Printf("cmd %s, %s: %v", msg.ChannelID, msg.Author.Username, cmd)

	switch cmd[0] {
	case "ping":
		s.ChannelMessageSend(msg.ChannelID, "Pong!")

	case "c":
		s.UpdateGameStatus(0, fmt.Sprint(counter))
		counter++
	}
}
