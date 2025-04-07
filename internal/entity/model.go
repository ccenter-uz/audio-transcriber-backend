package entity

type Filter struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type RowsEffected struct {
	RowsEffected int `json:"rows_effected"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type MultilingualField struct {
	Uz string `json:"uz" example:"Uzbek"`
	Ru string `json:"ru" example:"Русский"`
	Cy string `json:"cy" example:"Cyril"`
}
