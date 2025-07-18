package entity

type CreateAudioSegment struct {
	AudioId          int     `json:"audio_id"`
	FileName         string  `json:"filename_name"`
	Duration         float32 `json:"duration"`
	TranscribeOption string  `json:"transcribe_option"`
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
	UIserId string `json:"user_id"`
	UserID  string `json:"user_Id"`
	Filter  Filter `json:"filter"`
}
type AudioSegmentList struct {
	AudioSegments []AudioSegment `json:"audio_segments"`
	Count         int            `json:"count"`
}

type TranscriptPersent struct {
	TotalAudioFiles   int `json:"total_audio_files"`
	DoneAudioFiles    int `json:"completed_audio_files"`
	ProcessingAudio   int `json:"processing_audio"`
	PendingAudioFiles int `json:"pending_audio_files"`
	ErrorAudioFiles   int `json:"error_audio_files"`
	TotalSegments     int `json:"total_segments"`
	CompletedSegments int `json:"completed_segments"`
	ReportSegments    int `json:"report_segments"`
}

// type UserTranscriptCount struct {
// 	UserId        int    `json:"user_id"`
// 	Username      string `json:"username"`
// 	TotalSegments int    `json:"total_segments"`
// }

type UserTranscriptStatictics struct {
	Username         string  `json:"username"`
	TotalAudioFiles  int     `json:"total_audio_files"`
	TotalChunks      int     `json:"total_chunks"`
	TotalMinutes     float64 `json:"total_minutes"`
	WeeklyAudioFiles int     `json:"weekly_audio_files"`
	WeeklyChunks     int     `json:"weekly_chunks"`
	DailyChunks      string  `json:"daily_chunks"`
}

type TranscriptStatictics struct {
	StateDate            string             `json:"state_date"`
	DoneChunks           int                `json:"done_chunks"`
	InvalidChunks        int                `json:"invalid_chunks"`
	DoneAudioFiles       int                `json:"done_audio_files"`
	ErrorAudioFiles      int                `json:"error_audio_files"`
	ActiveOperators      float32            `json:"active_operators"`
	ActiveOperatorsBlock []DailyActiveBlock `json:"active_operators_block"`
}

type DailyActiveBlock struct {
	StatDate     string  `json:"stat_date"`
	OperatorID   string  `json:"operator_id"`
	Username     string  `json:"username"`
	ActiveBlocks float64 `json:"active_blocks"`
}

type DatasetViewerList struct {
	AudioID       int      `json:"audio_id"`
	AudioUrl      string   `json:"audio_url"`
	ChunkID       int      `json:"chunk_id"`
	ChunkUrl      string   `json:"chunk_url"`
	Duration      float32  `json:"duration"`
	PreviouText   *string  `json:"previous_text"`
	ChunkText     *string  `json:"text"`
	NextText      *string  `json:"next_text"`
	Sentence      *string  `json:"sentence"`
	ReportText    *string  `json:"report_text"`
	Transcriber   *string  `json:"transcriber"`
	TranscriberID *string  `json:"transcriber_id"`
	MinutesSpent  *float32 `json:"minutes_spent"`
	Emotion       *string  `json:"emotion"`
}

type DatasetViewerListResponse struct {
	Total int                 `json:"total"`
	Data  []DatasetViewerList `json:"data"`
}

type Statistics struct {
	Duration    map[string]int `json:"duration"`
	Text        map[string]int `json:"text"`
	PreviouText map[string]int `json:"previous_text"`
	NextText    map[string]int `json:"next_text"`
	Transcriber map[string]int `json:"transcriber"`
}

type DailyTranscript struct {
	HourRange string `json:"hour_range"`
	Count     int    `json:"count"`
}

type DailyTranscriptResponse struct {
	UserId           string            `json:"user_id"`
	Username         string            `json:"username"`
	DailyTranscripts []DailyTranscript `json:"daily_transcripts"`
	TotalCount       int               `json:"total_count"`
}

type ListDailyTranscriptResponse struct {
	Data []DailyTranscriptResponse `json:"data"`
}
