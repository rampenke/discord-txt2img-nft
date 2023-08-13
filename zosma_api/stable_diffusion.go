package zosma_api

import (
	"time"

	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	"discord-txt2img-nft/stable_diffusion_api"
	"log"

	"github.com/rampenke/zosma-api-server/tasks"
)

type apiImpl struct {
	cfg Config
}

type Config struct {
	Host     string
	Password string
}

func waitForResult(ctx context.Context, i *asynq.Inspector, queue, taskID string) (*asynq.TaskInfo, error) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			taskInfo, err := i.GetTaskInfo(queue, taskID)
			if err != nil {
				return nil, err
			}
			if taskInfo.CompletedAt.IsZero() {
				continue
			}
			return taskInfo, nil
		case <-ctx.Done():
			return nil, fmt.Errorf("context closed")
		}
	}
}

func New(cfg Config) (stable_diffusion_api.StableDiffusionAPI, error) {
	if cfg.Host == "" {
		return nil, errors.New("missing host")
	}

	return &apiImpl{
		cfg: cfg,
	}, nil
}

func (api *apiImpl) TextToImage(_request *stable_diffusion_api.TextToImageRequest) (*stable_diffusion_api.TextToImageResponse, error) {

	conn := asynq.RedisClientOpt{Addr: api.cfg.Host, Password: api.cfg.Password}
	client := asynq.NewClient(conn)
	defer client.Close()
	inspector := asynq.NewInspector(conn)
	request := &tasks.TextToImageRequest{
		Prompt:            _request.Prompt,
		NegativePrompt:    _request.NegativePrompt,
		Width:             _request.Width,
		Height:            _request.Height,
		RestoreFaces:      _request.RestoreFaces,
		EnableHR:          _request.EnableHR,
		HRResizeX:         _request.HRResizeX,
		HRResizeY:         _request.HRResizeY,
		DenoisingStrength: _request.DenoisingStrength,
		BatchSize:         _request.BatchSize,
		Seed:              _request.Seed,
		Subseed:           _request.Subseed,
		SubseedStrength:   _request.SubseedStrength,
		SamplerName:       _request.SamplerName,
		CfgScale:          _request.CfgScale,
		Steps:             _request.Steps,
		NIter:             _request.NIter,
	}
	task, err := tasks.NewTxt2imgTask(request)
	if err != nil {
		log.Fatalf("could not create task: %v", err)
	}
	info, err := client.Enqueue(task, asynq.MaxRetry(10), asynq.Timeout(3*time.Minute), asynq.Retention(2*time.Hour))
	if err != nil {
		log.Fatalf("could not enqueue task: %v", err)
	}
	log.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	res, err := waitForResult(ctx, inspector, "default", info.ID)
	if err != nil {
		log.Fatalf("unable to wait for resilt: %v", err)
	}
	var respStruct = &tasks.TextToImageResponse{}
	err = json.Unmarshal(res.Result, respStruct)
	if err != nil {
		log.Fatalf("Unexpected API response: %v", err)
	}

	return &stable_diffusion_api.TextToImageResponse{
		Images:   respStruct.Images,
		Seeds:    respStruct.Seeds,
		Subseeds: respStruct.Subseeds,
	}, nil

}

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	EtaRelative float64 `json:"eta_relative"`
}
