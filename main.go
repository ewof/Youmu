package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
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
	GuildID        = flag.String("guild", "929748108802412595", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", os.Getenv("YOUMU_TOKEN"), "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session

func init() { flag.Parse() }

// if you search these consider suicide. leave a space in front
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
			Description: "Search Gelbooru",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tags",
					Description: "tags to search with",
					Required:    true,
				},
			},
		},
		{
			Name:        "characterlist",
			Description: "List all of the characters for `/character`",
		},
		{
			Name:        "character",
			Description: "Search Gelbooru for character",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "charcter",
					Description: "character to search for (do /characterlist for options)",
					Required:    true,
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gelbooru": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			sendGelbooru(s, i, i.ApplicationCommandData().Options[0].StringValue())
		},
		"characterlist": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			embed := &discordgo.MessageEmbed{
				Title:       "Character List",
				Color:       0xA3BE8C,
				Description: characterlist,
				Timestamp:   time.Now().Format(time.RFC3339),
			}

			embeds := []*discordgo.MessageEmbed{embed}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: embeds,
				},
			})
		},
		// Wanted to make this autocomplete but discord only allows for 25 choices
		"character": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if val, ok := characters[i.ApplicationCommandData().Options[0].StringValue()]; ok {
				sendGelbooru(s, i, val)
			} else {
				embed := &discordgo.MessageEmbed{
					Title:       "Character - not found",
					Color:       0xBF616A,
					Description: "Do `/characterlist` for a list of characters",
					Timestamp:   time.Now().Format(time.RFC3339),
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

func sendGelbooru(s *discordgo.Session, i *discordgo.InteractionCreate, tags string) {
	orig_tags := tags
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
		description := fmt.Sprintf("Tags: `%s`", i.ApplicationCommandData().Options[0].StringValue())
		if !channel.NSFW {
			description += "\nMaybe you searched an nsfw tag in a non-nsfw channel?"
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Gelbooru - Nothing found",
			Color:       0xBF616A,
			Description: description,
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
			Description: "Tags: `" + orig_tags + "`",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Image Source",
					Value:  fmt.Sprintf("[%s](%s)", sourceSite, source),
					Inline: false,
				},
				&discordgo.MessageEmbedField{
					Name:   "Gelbooru ID",
					Value:  strconv.Itoa(post.ID),
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
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		return
	}

	// for testing
	// if m.GuildID != "924047241096876044" {
	// return
	// }

	if regexp.MustCompile("blocked message").MatchString(m.Content) {
		s.ChannelMessageSend(m.ChannelID, "https://media.discordapp.net/attachments/764447332288561152/947480845038538812/wow.gif")
	}

	if m.Content == "gasoline" && m.Author.ID == "489371664430268446" {
		s.MessageReactionAdd(m.ChannelID, m.ID, ":witness:941500786037366846")
	}

	if len(m.Content) >= 28 && m.Content[0:28] == "https://media.discordapp.net" && m.Content[len(m.Content)-4:len(m.Content)] == ".mp4" {
		s.ChannelMessageSendReply(m.ChannelID, "You are stupid, I fixed it\n"+m.Content[0:7]+"/cdn.discordapp.com/attachments"+m.Content[40:len(m.Content)], m.Reference())
	}

}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(messageCreate)
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Bot is up!")
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	// for _, v := range commands {
	// 	_, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
	// 	if err != nil {
	// 		log.Errorln("Cannot create " + v.Name + " command: " + err.Error())
	// 	}
	// }

	defer s.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Info("Shutting down...")
}
