package entity

type CreateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserInfo struct {
	AgentID      string  `json:"agent_id"`
	Login        string  `json:"login"`
	Password     string  `json:"password"`
	ServiceName  string  `json:"service_name"`
	Name         string  `json:"name"`
	FirstNumber  string  `json:"first_number"`
	Role         string  `json:"role"`
	Image        *string `json:"image"`
	CreateDate   string  `json:"create_date"`
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

type User struct {
	AgentID      string  `json:"agent_id"`
	ServiceName  string  `json:"service_name"`
	Name         string  `json:"name"`
	FirstNumber  string  `json:"first_number"`
	CreateDate   string  `json:"create_date"`
}

type GetUserReq struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Filter   Filter `json:"filter"`
}
type UserList struct {
	Users []User `json:"users"`
	Count int        `json:"count"`
}

type LoginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginRes struct {
	Message string `json:"message"`
	Role    string `json:"role"`
	Token   string `json:"token"`
}
