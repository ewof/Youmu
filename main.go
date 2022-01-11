package main

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: new(logrus.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}

// Bot parameters
var (
	GuildID        = flag.String("guild", "924047241096876044", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", os.Getenv("YOUMU_TOKEN"), "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session

func init() { flag.Parse() }

// if you search these consider suicide, leave a space in front
var bannedtags string = " -futanari -futa -loli -poop -scat -feces -guro -shota -furry"

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "gelbooru",
			Description: "Search gelbooru",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tags",
					Description: "tags to search with",
					Required:    true,
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gelbooru": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Booru tags
			tags := i.ApplicationCommandData().Options[0].StringValue()
			tags += bannedtags
			channel, errc := s.Channel(i.ChannelID)
			if errc != nil {
				log.Fatalf("Error getting the channel: %v", errc)
			}

			post, found, errg := Gelbooru(tags, channel.NSFW)
			if errg != nil {
				log.Fatalf("Error getting Gelbooru post: %v", errg)
			}
			if !found {
				embed := &discordgo.MessageEmbed{
					Title:       "Gelbooru - Nothing found",
					Color:       0xBF616A,
					Description: "Tags: `" + i.ApplicationCommandData().Options[0].StringValue() + "`",
					Timestamp:   time.Now().Format(time.RFC3339),
				}

				embeds := []*discordgo.MessageEmbed{embed}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: embeds,
					},
				})
			} else {
				source := post.Source
				sourceSite := "Source"
				if strings.Contains(source, "pixiv") || strings.Contains(source, "pximg") {
					sourceSite = "Pixiv"
				} else if strings.Contains(source, "twitter") {
					sourceSite = "Twitter"
				} else if strings.Contains(source, "nicovideo") {
					sourceSite = "NicoNico"
				} else if strings.Contains(source, "deviantart") {
					sourceSite = "DeviantArt"
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Gelbooru",
					Color:       0xA3BE8C,
					Description: "Tags: `" + i.ApplicationCommandData().Options[0].StringValue() + "`",
					Fields: []*discordgo.MessageEmbedField{
						&discordgo.MessageEmbedField{
							Name:   "Image Source",
							Value:  "[" + sourceSite + "](" + source + ")",
							Inline: true,
						},
						&discordgo.MessageEmbedField{
							Name:   "Dimensions",
							Value:  strconv.Itoa(post.Height) + "x" + strconv.Itoa(post.Width),
							Inline: true,
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: post.FileURL,
					},
					Timestamp: time.Now().Format(time.RFC3339),
				}

				embeds := []*discordgo.MessageEmbed{embed}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: embeds,
					},
				})
			}
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Bot is up!")
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Errorln("Cannot create " + v.Name + " command: " + err.Error())
		}
	}

	defer s.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Info("\nGracefully shutdowning")
}
