package entity

type AudioSegment struct {
	Id        int    `json:"id"`
	AudioId   int    `json:"audio_id"`
	AudioName string `json:"audio_name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type GetAudioSegmentReq struct {
	Status  string `json:"status"`
	AudioId string `json:"audio_id"`
	Filter  Filter `json:"filter"`
}
type AudioSegmentList struct {
	AudioSegments []AudioSegment `json:"audo_segments"`
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
