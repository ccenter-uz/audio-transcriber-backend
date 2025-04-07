package entity

type CreateVillage struct {
	Name         string `json:"name"`
	LangCodeName string `json:"lang_code_name"`
	OldName      string `json:"old_name"`
	LangCodeOld  string `json:"lang_code_old"`
	NewName      string `json:"new_name"`
	LangCodeNew  string `json:"lang_code_new"`
	RegionName   string `json:"region_name"`
	CityName     string `json:"city_name"`
	DistrictName string `json:"district_name"`
	Index        string `json:"index"`
	Status       int    `json:"status"`
	UpdatedAt    string `json:"updated_at"`
	StaffNumber  string `json:"staff_number"`
}

type Village struct {
	Id           int               `json:"id"`
	Status       string            `json:"status"`
	Name         MultilingualField `json:"name"`
	OldName      string            `json:"old_name"`
	NewName      string            `json:"new_name"`
	RegionName   string            `json:"region_name"`
	CityName     string            `json:"city_name"`
	DistrictName string            `json:"district_name"`
	Index        string            `json:"index"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}
