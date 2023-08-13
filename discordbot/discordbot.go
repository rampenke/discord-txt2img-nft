package discordbot

import (
	"discord-txt2img-nft/msgqueue"
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type botImpl struct {
	botSession         *discordgo.Session
	registeredCommands []*discordgo.ApplicationCommand
	guildID            string // Empty in production or test guild id in dev
	txt2imgQueue       msgqueue.Queue
	removeCommands     bool
}

var (
	tx2imgCommand = "dream"
)

func NewBot(cfg *Config) (Bot, error) {
	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	err = botSession.Open()
	if err != nil {
		return nil, err
	}
	bot := &botImpl{
		botSession:     botSession,
		guildID:        "",
		txt2imgQueue:   cfg.Txt2imgQueue,
		removeCommands: cfg.RemoveCommands,
	}

	err = bot.addTxt2ImgCommand()
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case tx2imgCommand:
				bot.processTxt2imgCommand(s, i)
			default:
				log.Printf("Unknown command '%v'", i.ApplicationCommandData().Name)
			}

		case discordgo.InteractionMessageComponent:
			switch customID := i.MessageComponentData().CustomID; {
			case customID == "imagine_reroll":
				_ = customID
			}
		}
	})

	return bot, nil
}

func (b *botImpl) addTxt2ImgCommand() error {
	log.Printf("Adding command '%s'...", tx2imgCommand)

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        tx2imgCommand,
		Description: "Ask the bot to generate image",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "The text prompt to image",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", tx2imgCommand, err)
		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) processTxt2imgCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var position int
	var queueError error
	var prompt string

	if option, ok := optionMap["prompt"]; ok {
		prompt = option.StringValue()

		position, queueError = b.txt2imgQueue.AddTxt2Img(&msgqueue.QueueItem{
			Prompt:             prompt,
			Type:               msgqueue.ItemTypeTxt2Img,
			DiscordInteraction: i.Interaction,
		})
		if queueError != nil {
			log.Printf("Error adding imagine to queue: %v\n", queueError)
		}

		_ = prompt
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(
				"Processing request. Waiting position #%d.\n<@%s> asked me to generate image for \"%s\".",
				position,
				i.Member.User.ID,
				prompt),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) Start() {
	b.txt2imgQueue.StartPolling(b.botSession)

	err := b.teardown()
	if err != nil {
		log.Printf("Error tearing down bot: %v", err)
	}
}

func (b *botImpl) teardown() error {
	// Delete all commands added by the bot
	if b.removeCommands {
		log.Printf("Removing all commands added by bot...")

		for _, v := range b.registeredCommands {
			log.Printf("Removing command '%v'...", v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.guildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}
	return b.botSession.Close()
}
