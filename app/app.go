package app

import (
	"discord-txt2img-nft/discordbot"
	"discord-txt2img-nft/msgqueue"
	"discord-txt2img-nft/stable_diffusion_api"

	"discord-txt2img-nft/zosma_api"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "discord-txt2img-nft",
	Short: "Text to image generatoin",
	Long:  `A text to image genrator driven by discord bot`,
	Run: func(cmd *cobra.Command, args []string) {
		//ctx := context.Background()
		var (
			stableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
			err                error
		)
		cfg := discordbot.LoadConfig()

		stableDiffusionAPI, err = zosma_api.New(zosma_api.Config{
			Host:     cfg.ApiHost,
			Password: cfg.Password,
		})

		txt2imgQueue, err := msgqueue.NewMsgQueue(msgqueue.Config{
			StableDiffusionAPI: stableDiffusionAPI,
		})
		if err != nil {
			log.Fatalf("Failed to create imagine queue: %v", err)
		}

		cfg.Txt2imgQueue = txt2imgQueue

		cfg.RemoveCommands = true
		bot, err := discordbot.NewBot(cfg)
		if err != nil {
			log.Fatal(err.Error())
		}
		bot.Start()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
