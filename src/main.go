package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"

	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"

	openai "github.com/sashabaranov/go-openai"
)

const version string = "Test build 002"

var debug_channel string
var talking_channel string

var GSession *discordgo.Session
var aiClient *openai.Client

func main() {
	viper.SetDefault("gpttoken", 0)
	viper.SetDefault("token", 0)
	viper.SetDefault("debugChannel", 0)
	viper.SetDefault("talkingChannel", 0)
	viper.SetDefault("oldversion", "unknown")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("/config")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	gpttoken := viper.Get("gpttoken").(string)
	fmt.Println("gpttoken=" + gpttoken)

	token := viper.Get("token").(string)
	fmt.Println("token=" + token)

	debug_channel = viper.Get("debugChannel").(string)
	fmt.Println("debugChannel=" + debug_channel)

	talking_channel = viper.Get("talkingChannel").(string)
	fmt.Println("talkingChannel=" + talking_channel)

	oldVersion := viper.Get("oldversion").(string)

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	GSession = dg

	aiClient = openai.NewClient(gpttoken)

	fmt.Println("Bot online")
	if oldVersion != version {
		dg.ChannelMessageSend(debug_channel, "升级成功！\n旧版本:"+oldVersion+"\n当前版本:"+version)
		viper.Set("oldversion", version)
		viper.WriteConfig()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("offline.")
	dg.ChannelMessageSend(debug_channel, "Bot OFFLINE")
	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.ChannelMessageSend(debug_channel, "Bot ONLINE")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Author.Bot {
		return
	}
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}
	fmt.Printf("[" + channel.Name + "]" + m.Author.Username + ":" + m.Content + "\n")

	// 愿此bot寿与天齐
	if m.Content == "苟利国家生死以" {
		s.ChannelMessageSend(m.ChannelID, "岂因祸福避趋之")
		return
	}
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "pong")
		return
	}
	if len(m.Content) > 0 && m.ChannelID == talking_channel {
		resp, err := aiClient.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: m.Content,
					},
				},
			},
		)
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			s.ChannelMessageSend(m.ChannelID, "GPT response Error...")
			return
		}

		fmt.Printf("GPT: %s\n", resp.Choices[0].Message.Content)
		s.ChannelMessageSend(m.ChannelID, resp.Choices[0].Message.Content)
	}
}
