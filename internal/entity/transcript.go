package entity

type Transcript struct {
	Id               int     `json:"id"`
	AudioId          int     `json:"audio_id"`
	AudioName        string  `json:"audio_name"`
	SegmentId        int     `json:"segment_id"`
	UserId           *string `json:"user_id"`
	Username         *string `json:"username"`
	AIText           *string `json:"ai_text"`
	TranscriptText   *string `json:"transcribe_text"`
	ReportText       *string `json:"report_text"`
	TranscriptOption *string `json:"transcribe_option"`
	Status           string  `json:"status"`
	Emotion          *string `json:"emotion"`
	CreatedAt        string  `json:"created_at"`
}

type CreateTranscript struct {
	SegmentId int    `json:"segment_id"`
	AIText    string `json:"ai_text"`
}

type UpdateTranscript struct {
	Id                 int     `json:"id"`
	TranscriptText     string  `json:"transcribe_text"`
	ReportText         string  `json:"report_text"`
	UserID             *string `json:"user_id"`
	EntireAudioInvalid bool    `json:"entire_audio_invalid"`
	Emotion            string  `json:"emotion"`
}

type UpdateTranscriptBody struct {
	TranscriptText     string `json:"transcribe_text"`
	ReportText         string `json:"report_text"`
	EntireAudioInvalid bool   `json:"entire_audio_invalid"`
	Emotion            string `json:"emotion"`
}

type GetTranscriptReq struct {
	Status  string `json:"status"`
	AudioId string `json:"audio_id"`
	UserId  string `json:"user_id"`
	Filter  Filter `json:"filter"`
}

type TranscriptList struct {
	Transcripts []Transcript `json:"transcripts"`
	Count       int          `json:"count"`
}
