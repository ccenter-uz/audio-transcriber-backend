package entity

type CreateAudioSegment struct {
	AudioId   int    `json:"audio_id"`
	FileName  string `json:"filename_name"`
}

type AudioSegment struct {
	Id        int    `json:"id"`
	AudioId   int    `json:"audio_id"`
	AudioName string `json:"audio_name"`
	Status    string `json:"status"`
	FilePath  string `json:"file_path"`
	CreatedAt string `json:"created_at"`
}

type GetAudioSegmentReq struct {
	Status  string `json:"status"`
	AudioId string `json:"audio_id"`
	Filter  Filter `json:"filter"`
}
type AudioSegmentList struct {
	AudioSegments []AudioSegment `json:"audio_segments"`
	Count         int            `json:"count"`
}

type TranscriptPersent struct {
	AudioFileId       int     `json:"audio_file_id"`
	Filename          string  `json:"filename"`
	TotalSegments     int     `json:"total_segments"`
	CompletedSegments int     `json:"completed_segments"`
	Percent           float64 `json:"percent"`
}

type UserTranscriptCount struct {
	UserId        int    `json:"user_id"`
	Username      string `json:"username"`
	TotalSegments int    `json:"total_segments"`
}
