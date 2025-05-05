package entity

type CreateAudioFile struct {
	Filename string `json:"filename"`
	FilePath string `json:"file_path"`
}

type AudioFile struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	UserID   string `json:"user_id"`
}
