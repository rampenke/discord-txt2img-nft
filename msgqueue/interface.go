package msgqueue

import (
	"github.com/bwmarrin/discordgo"
)

type ItemType int

const (
	ItemTypeTxt2Img ItemType = iota
)

type QueueItem struct {
	Prompt             string
	Type               ItemType
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
}

const (
	botID = "bot"

	initializedWidth  = 512
	initializedHeight = 512
)

type Queue interface {
	AddTxt2Img(item *QueueItem) (int, error)
	StartPolling(botSession *discordgo.Session)
}
