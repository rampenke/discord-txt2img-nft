package discordbot

import (
	"discord-txt2img-nft/msgqueue"
	"log"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BotToken       string `envconfig:"BOT_TOKEN" required:"true"`
	ApiHost        string `envconfig:"API_HOST" required:"true"`
	Password       string `envconfig:"PASSWORD" required:"true"`
	Txt2imgQueue   msgqueue.Queue
	RemoveCommands bool
}

var cfg Config
var once sync.Once

func LoadConfig() *Config {
	once.Do(func() {
		_ = godotenv.Overload()

		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatal(err.Error())
		}
	})
	return &cfg
}
