package stable_diffusion_api

//go:generate mockgen -destination=../mocks/mock_stable_diffusion_api.go -package=mock_stable_diffusion_api . StableDiffusionAPI

type JsonTextToImageResponse struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
}

type JsonInfoResponse struct {
	Seed        int   `json:"seed"`
	AllSeeds    []int `json:"all_seeds"`
	AllSubseeds []int `json:"all_subseeds"`
}

type TextToImageResponse struct {
	Images   []string `json:"images"`
	Seeds    []int    `json:"seeds"`
	Subseeds []int    `json:"subseeds"`
}

type TextToImageRequest struct {
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	RestoreFaces      bool    `json:"restore_faces"`
	EnableHR          bool    `json:"enable_hr"`
	HRResizeX         int     `json:"hr_resize_x"`
	HRResizeY         int     `json:"hr_resize_y"`
	DenoisingStrength float64 `json:"denoising_strength"`
	BatchSize         int     `json:"batch_size"`
	Seed              int     `json:"seed"`
	Subseed           int     `json:"subseed"`
	SubseedStrength   float64 `json:"subseed_strength"`
	SamplerName       string  `json:"sampler_name"`
	CfgScale          float64 `json:"cfg_scale"`
	Steps             int     `json:"steps"`
	NIter             int     `json:"n_iter"`
}

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	EtaRelative float64 `json:"eta_relative"`
}

type StableDiffusionAPI interface {
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
}
