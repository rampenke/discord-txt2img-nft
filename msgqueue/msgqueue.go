package msgqueue

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"discord-txt2img-nft/stable_diffusion_api"

	"discord-txt2img-nft/domain/model"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	StableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
}

type msgQueueImpl struct {
	botSession         *discordgo.Session
	stableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
	queue              chan *QueueItem
	currentTxt2Img     *QueueItem
	mu                 sync.Mutex
}

func NewMsgQueue(cfg Config) (Queue, error) {

	return &msgQueueImpl{
		stableDiffusionAPI: cfg.StableDiffusionAPI,
		queue:              make(chan *QueueItem, 100),
	}, nil
}

func (q *msgQueueImpl) AddTxt2Img(item *QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *msgQueueImpl) StartPolling(botSession *discordgo.Session) {
	q.botSession = botSession

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	stopPolling := false

	for {
		select {
		case <-stop:
			stopPolling = true
		case <-time.After(1 * time.Second):
			//log.Info("Polling for msg")newGeneration
			if q.currentTxt2Img == nil {
				q.pullNextInQueue()
			}
		}

		if stopPolling {
			break
		}
	}
	log.Printf("Polling stopped...\n")
}

func (q *msgQueueImpl) pullNextInQueue() {
	if len(q.queue) > 0 {
		element := <-q.queue

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentTxt2Img = element

		q.processCurrentTxt2Img()
	}
}

func (q *msgQueueImpl) processCurrentTxt2Img() {
	go func() {
		defer func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			q.currentTxt2Img = nil
		}()

		enableHR := false
		// TODO: Configure
		defaultWidth := 512
		defaultHeight := 512
		hiresWidth := defaultWidth
		hiresHeight := defaultHeight

		// new generation with defaults
		newGeneration := &model.Txt2img{
			Prompt: q.currentTxt2Img.Prompt,
			NegativePrompt: "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
				"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
				"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy",
			Width:             defaultWidth,
			Height:            defaultHeight,
			RestoreFaces:      true,
			EnableHR:          enableHR,
			HiresWidth:        hiresWidth,
			HiresHeight:       hiresHeight,
			DenoisingStrength: 0.7,
			Seed:              -1,
			Subseed:           -1,
			SubseedStrength:   0,
			SamplerName:       "Euler a",
			CfgScale:          9,
			Steps:             20,
			Processed:         false,
		}

		err := q.processTxt2img(newGeneration, q.currentTxt2Img)
		if err != nil {
			log.Printf("Error processing imagine grid: %v", err)

			return
		}
	}()
}

func (q *msgQueueImpl) processTxt2img(newGeneration *model.Txt2img, txt2img *QueueItem) error {
	log.Printf("Processing Tx2img #%s: %v\n", txt2img.DiscordInteraction.ID, newGeneration.Prompt)

	newContent := txt2imgMessageContent(newGeneration, txt2img.DiscordInteraction.Member.User, 0)

	message, err := q.botSession.InteractionResponseEdit(txt2img.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
	}

	defaultBatchCount := 1

	defaultBatchSize := 1

	newGeneration.InteractionID = txt2img.DiscordInteraction.ID
	newGeneration.MessageID = message.ID
	newGeneration.MemberID = txt2img.DiscordInteraction.Member.User.ID
	newGeneration.SortOrder = 0
	newGeneration.BatchCount = defaultBatchCount
	newGeneration.BatchSize = defaultBatchSize
	newGeneration.Processed = true
	newGeneration.CreatedAt = time.Now()

	resp, err := q.stableDiffusionAPI.TextToImage(&stable_diffusion_api.TextToImageRequest{
		Prompt:            newGeneration.Prompt,
		NegativePrompt:    newGeneration.NegativePrompt,
		Width:             newGeneration.Width,
		Height:            newGeneration.Height,
		RestoreFaces:      newGeneration.RestoreFaces,
		EnableHR:          newGeneration.EnableHR,
		HRResizeX:         newGeneration.HiresWidth,
		HRResizeY:         newGeneration.HiresHeight,
		DenoisingStrength: newGeneration.DenoisingStrength,
		BatchSize:         newGeneration.BatchSize,
		Seed:              newGeneration.Seed,
		Subseed:           newGeneration.Subseed,
		SubseedStrength:   newGeneration.SubseedStrength,
		SamplerName:       newGeneration.SamplerName,
		CfgScale:          newGeneration.CfgScale,
		Steps:             newGeneration.Steps,
		NIter:             newGeneration.BatchCount,
	})
	if err != nil {
		log.Printf("Error processing image: %v\n", err)

		errorContent := "There is a problem generating your image."

		_, err = q.botSession.InteractionResponseEdit(txt2img.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &errorContent,
		})

		return err
	}

	finishedContent := txt2imgMessageContent(newGeneration, txt2img.DiscordInteraction.Member.User, 1)

	log.Printf("Seeds: %v Subseeds:%v", resp.Seeds, resp.Subseeds)

	imageBufs := make([]*bytes.Buffer, len(resp.Images))

	for idx, image := range resp.Images {
		decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
		if decodeErr != nil {
			log.Printf("Error decoding image: %v\n", decodeErr)
		}

		imageBuf := bytes.NewBuffer(decodedImage)

		imageBufs[idx] = imageBuf
	}

	_, err = q.botSession.InteractionResponseEdit(txt2img.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				Name:        "imagine.png",
				Reader:      imageBufs[0],
			},
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)

		return err
	}

	return nil
}

func txt2imgMessageContent(generation *model.Txt2img, user *discordgo.User, progress float64) string {
	if progress >= 0 && progress < 1 {
		return fmt.Sprintf("<@%s> asked me to generate image for \"%s\". Currently generating...",
			user.ID, generation.Prompt)
	} else {
		return fmt.Sprintf("<@%s> asked me to generate image for \"%s\", here is generated image.",
			user.ID,
			generation.Prompt,
		)
	}
}
