package iam_reqdto

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	FullName    string  `json:"full_name" binding:"required,min=2,max=120"`
	Email       string  `json:"email" binding:"required,email"`
	Username    string  `json:"username" binding:"required,min=3,max=32"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	Password    string  `json:"password" binding:"required,min=8"`
	RePassword  string  `json:"re_password" binding:"required,min=8"`
}
