package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/bakape/boorufetch"
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

var (
	// if you search these consider suicide
	nevertags = []string{"futanari", "futa", "loli", "poop", "scat", "feces", "guro", "shota", "furry"}
	// only allowed in nsfw channels
	badtags = []string{"sex", "rape", "breasts", "penis", "pussy", "vaginal", "anal", "rating:explicit", "rating:questionable"}

	nevertag string = ""
	badtag   string = ""
)

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	for i := 0; i < len(nevertags); i++ {
		nevertag += nevertags[i]
		nevertag += " "
	}
	for i := 0; i < len(badtags); i++ {
		badtag += badtags[i]
		badtag += " "
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
			postl, errg := boorufetch.FromDanbooru(tags, 0, 100)
			// postlLen := len(postl)
			postlLen := 100
			channel, errc := s.Channel(i.ChannelID)
			if errc != nil {
				log.Fatalf("Cannot getting the channel: %v", errc)
			}
			s1 := rand.NewSource(time.Now().UnixNano())
			r1 := rand.New(s1)

			if (!channel.NSFW && isNSFW(tags)) || isNever(tags) {
				embed := &discordgo.MessageEmbed{
					Title:       "Gelbooru - Tag not allowed",
					Color:       0xBF616A,
					Description: "Tags: `" + tags + "`",
					Fields: []*discordgo.MessageEmbedField{
						&discordgo.MessageEmbedField{
							Name:   "Banned in non NSFW channels",
							Value:  badtag,
							Inline: true,
						},
						&discordgo.MessageEmbedField{
							Name:   "Banned Everywhere",
							Value:  nevertag,
							Inline: true,
						},
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
			} else if postlLen != 0 {
				if errg != nil {
					log.Fatalf("No posts found!: %v", errg)
				}
				post := postl[r1.Intn(postlLen)]
				rating, errr := post.Rating()
				if errr != nil {
					log.Fatalf("Error getting rating: %v", errr)
				}

				channel, errc := s.Channel(i.ChannelID)
				if errc != nil {
					log.Fatalf("Cannot open the session: %v", errc)
				}

				if !channel.NSFW {
					count := 0
					for rating != 0 {
						if count > (postlLen - 1) {
							log.Info("NSFW attempted in non NSFW channel, no safe post found")
							embed := &discordgo.MessageEmbed{
								Title:       "Gelbooru - Post was NSFW, no safe posts found",
								Color:       0xBF616A,
								Description: "Tags: `" + tags + "`",
								Timestamp:   time.Now().Format(time.RFC3339),
							}

							embeds := []*discordgo.MessageEmbed{embed}

							s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Embeds: embeds,
								},
							})
							break
						}
						log.Info("Wanted to post NSFW in non NSFW channel, finding new, count " + strconv.Itoa(count))
						s1 = rand.NewSource(time.Now().UnixNano())
						r1 = rand.New(s1)
						post = postl[r1.Intn(postlLen)]
						rating, errr = post.Rating()
						if errr != nil {
							log.Fatalf("Cannot open the session: %v", errr)
						}
						if rating == 0 {
							break
						}
						count++
					}
				}

				source := post.SourceURL()
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
					Description: "Tags: `" + tags + "`",
					Fields: []*discordgo.MessageEmbedField{
						&discordgo.MessageEmbedField{
							Name:   "Image Source",
							Value:  "[" + sourceSite + "](" + source + ")",
							Inline: true,
						},
						&discordgo.MessageEmbedField{
							Name:   "Dimensions",
							Value:  strconv.FormatUint(post.Height(), 10) + "x" + strconv.FormatUint(post.Width(), 10),
							Inline: true,
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: post.FileURL(),
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
			} else {
				embed := &discordgo.MessageEmbed{
					Title:       "Gelbooru - Nothing found",
					Color:       0xBF616A,
					Description: "Tags: `" + tags + "`",
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

func isNever(tags string) bool {
	for i := 0; i < len(nevertags); i++ {
		if strings.Contains(tags, nevertags[i]) {
			return true
		}
	}
	return false
}

func isNSFW(tags string) bool {
	for i := 0; i < len(badtags); i++ {
		if strings.Contains(tags, badtags[i]) {
			return true
		}
	}
	return false
}

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
