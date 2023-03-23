package bot

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/Fancy11111/mattermost-xkcd-bot/xkcd"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Settings struct {
	ServerUrl    string
	Team         string
	LogChannel   string
	BotToken     string
	BotId        string
	Prefix       string
	WebsocketUrl string
}

type Bot struct {
	client           *model.Client4
	botUser          *model.User
	webSocketClient  *model.WebSocketClient
	botTeam          *model.Team
	debuggingChannel *model.Channel
	settings         *Settings
}

func NewBot() (*Bot, error) {
	bot := new(Bot)
	bot.readSettings()
	log.Debug().Interface("settings", bot.settings).Msg("Loaded settings")
	bot.client = model.NewAPIv4Client((bot.settings.ServerUrl))
	bot.client.SetToken(bot.settings.BotToken)

	if err := bot.getBotUser(); err != nil {
		return nil, err
	}
	log.Debug().Str("bot-user", bot.botUser.Username).Msg("Loaded bot user")

	bot.setupGracefulShutdown()

	if err := bot.findBotTeam(); err != nil {
		return nil, err
	}
	log.Debug().Str("bot-team", bot.botTeam.Name).Msg("Loaded bot team")

	if err := bot.createBotDebuggingChannelIfNeeded(); err != nil {
		return nil, err
	}
	log.Debug().Str("debug-channel", bot.debuggingChannel.Name).Msg("Loaded debug channel")

	return bot, nil
}

func (bot *Bot) Start() (err error) {
	if err = bot.checkServerConnection(); err != nil {
		return
	}

	err = bot.sendMsgToDebuggingChannel("bot "+bot.botUser.Username+" has **started** running", "")
	if err != nil {
		return
	}
	err = nil
	bot.webSocketClient, err = model.NewWebSocketClient4(bot.settings.WebsocketUrl, bot.client.AuthToken)
	if err != nil && bot.webSocketClient == nil {
		log.Info().Interface("error", err).Msg("error info")
		log.Info().Str("error", err.Error()).Msg("error info")
		log.Error().Err(err).Msg("Failed to connect to web socket")
		// log.Error().Err(err).Str("websocket-url", bot.settings.WebsocketUrl).Msg("Failed to connect to web socket")
		return
	}
	//
	bot.webSocketClient.Listen()
	//
	go func() {
		for {
			select {
			case resp := <-bot.webSocketClient.EventChannel:
				bot.handleWebSocketResponse(resp)
			}
		}
	}()
	//
	// // You can block forever with
	select {}
}

func (bot *Bot) readSettings() error {
	bot.settings = new(Settings)
	err := viper.Unmarshal(bot.settings)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to decode setting")
		return err
	}

	checkURL, err := url.Parse(bot.settings.ServerUrl)
	if err != nil {
		log.Error().Err(err).Msg("Server URL is not valid")
		return err
	}

	bot.settings.WebsocketUrl = "ws://" + checkURL.Host
	if checkURL.Scheme == "https" {
		bot.settings.WebsocketUrl = "wss://" + checkURL.Host
	}

	return nil
}

func (b *Bot) checkServerConnection() error {
	// Check api connection
	if _, resp := b.client.GetOldClientConfig(""); resp.Error != nil {
		log.Error().Err(resp.Error).Msg("Server could not be reached")
		return resp.Error
	}

	// Get channel list
	// _, resp := b.client.GetTeamByName(b.settings.Team, "")
	// if resp.Error != nil {
	// 	log.Error().Err(resp.Error).Msg("Server could not be reached")
	// 	return resp.Error
	// }
	return nil
}

func (bot *Bot) handleWebSocketResponse(event *model.WebSocketEvent) {
	bot.handleMsgFromDebuggingChannel(event)
}

func (bot *Bot) handleMsgFromDebuggingChannel(event *model.WebSocketEvent) {
	// Lets only reponded to messaged posted events
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}
	log.Debug().Interface("event", event).Msg("New event")

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
	if post != nil {

		if !strings.HasPrefix(post.Message, bot.settings.Prefix) {
			return
		}
		log.Debug().Msg("Responding to msg")

		// dm, resp := bot.client.CreateDirectChannel(bot.settings.BotId, post.UserId)
		// if resp.Error != nil {
		// 	log.Error().Err(resp.Error).Msg("There was a problem getting direct message channel")
		// }

		// ignore my events
		if post.UserId == bot.settings.BotId {
			return
		}

		if strings.Contains(post.Message, "daily") {
			xkcdPost, err := xkcd.GetDailyPost()
			if err != nil {
				bot.replyToPost(post, "Sorry, I could not find the requested XKCD!")
				return
			}
			bot.replyWithXKCD(post, xkcdPost)	
		} else {
			parts := strings.Split(post.Message, " ")
			log.Debug().Str("msg", post.Message).Msg("specific post")
			if len(parts) == 2 {
				number, err := strconv.ParseInt(parts[1], 0, 64)
				if err != nil {
					bot.replyToPost(post, "Could not find parse post number: "+parts[1])
					return
				}
				xkcdPost, err := xkcd.GetPost(number)
				if err != nil {
					bot.replyToPost(post, "Sorry, I could not find the requested XKCD!")
					return
				}
				bot.replyWithXKCD(post, xkcdPost)
			}
		}
	}
}

func (bot *Bot) replyWithXKCD(post *model.Post, xkcdPost *xkcd.XKCDPost) {
	if len(xkcdPost.AltText) > 0 {
		bot.replyToPost(post, fmt.Sprintf("Title: %s\nAlt Text: %s\nImage: %s", xkcdPost.SafeTitle, xkcdPost.AltText, xkcdPost.Image))
	} else {
		bot.replyToPost(post, fmt.Sprintf("Title: %s\nImage: %s", xkcdPost.SafeTitle, xkcdPost.Image))
	}
}

func (bot *Bot) replyToPost(post *model.Post, msg string) {
	bot.sendMsgToChannel(post.ChannelId, msg, post.Id)
}

func (bot *Bot) findBotTeam() error {
	log.Debug().Str("bot-team", bot.settings.Team).Msg("Trying to load bot team")
	botTeams, resp := bot.client.SearchTeams(&model.TeamSearch{
		Term: bot.settings.Team,
	})
	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("Error trying to find bot user")
		return resp.Error
	}

	if len(botTeams) == 0 {
		err := fmt.Errorf("Could not find team with name: %s", bot.settings.Team)
		log.Error().Err(err).Msg("Could not find team")
		return err
	}
	bot.botTeam = botTeams[0]
	// if team, resp := bot.client.GetTeamByName(bot.settings.Team, ""); resp.Error != nil {
	// 	log.Error().Err(resp.Error).Str("team_name", bot.settings.Team).Msg("Failed to find bot team")
	// 	return resp.Error
	// } else {
	// 	bot.botTeam = team
	// }
	return nil
}

func (bot *Bot) createBotDebuggingChannelIfNeeded() error {
	if rchannel, resp := bot.client.GetChannelByName(bot.settings.LogChannel, bot.botTeam.Id, ""); resp.Error != nil {
		log.Warn().Str("logChannel", bot.settings.LogChannel).Msg("Log Channel does not exist")
	} else {
		bot.debuggingChannel = rchannel
		return nil
	}

	// Looks like we need to create the logging channel
	channel := &model.Channel{}
	channel.Name = bot.settings.LogChannel
	channel.DisplayName = "Debugging For Sample Bot"
	channel.Purpose = "This is used as a test channel for logging bot debug messages"
	channel.Type = model.CHANNEL_OPEN
	channel.TeamId = bot.botTeam.Id
	if rchannel, resp := bot.client.CreateChannel(channel); resp.Error != nil {
		log.Error().Err(resp.Error).Msg("Failed to create the log channel")
		return resp.Error
	} else {
		bot.debuggingChannel = rchannel
		log.Info().Str("channel_name", bot.settings.LogChannel).Msg("Created the log channel")
	}
	return nil
}

func (bot *Bot) sendMsgToChannel(channelId string, msg string, replyToId string) error {
	channel, resp := bot.client.GetChannel(channelId, "")
	if resp.Error != nil {
		log.Error().Err(resp.Error).Str("channel_id", channelId).Msg("Could not get channel")
		return resp.Error
	}

	post := &model.Post{}
	post.ChannelId = channelId
	post.Message = msg

	post.RootId = replyToId

	if _, resp := bot.client.CreatePost(post); resp.Error != nil {
		log.Error().Err(resp.Error).Interface("message", post).Str("channel_name", channel.Name).Msg("Could not send message")
		return resp.Error
	}
	return nil
}

func (bot *Bot) sendMsgToDebuggingChannel(msg string, replyToId string) error {
	post := &model.Post{}
	post.ChannelId = bot.debuggingChannel.Id
	post.Message = msg

	post.RootId = replyToId

	if _, resp := bot.client.CreatePost(post); resp.Error != nil {
		log.Error().Err(resp.Error).Interface("message", post).Msg("Failed to post message to log channel")
		return resp.Error
	}
	return nil
}

func (bot *Bot) getBotUser() error {
	if user, resp := bot.client.GetUser(bot.settings.BotId, ""); resp.Error != nil {
		log.Error().Err(resp.Error).Str("bot_id", bot.settings.BotId).Msg("Could not get bot user")
		return resp.Error
	} else {
		bot.botUser = user
	}
	return nil
}

func (bot *Bot) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if bot.webSocketClient != nil {
				bot.webSocketClient.Close()
			}

			bot.sendMsgToDebuggingChannel("bot "+bot.botUser.Username+" has **stopped** running", "")
			os.Exit(0)
		}
	}()
}
