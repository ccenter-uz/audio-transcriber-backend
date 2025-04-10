package entity

type CreateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type User struct {
	Id        int    `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type UpdateUser struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type UpdateUserBody struct {
	Username string `json:"username"`
	// Role     string `json:"role"`
}

type GetUserReq struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Filter   Filter `json:"filter"`
}
type UserList struct {
	Users []User `json:"users"`
	Count int    `json:"count"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRes struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
