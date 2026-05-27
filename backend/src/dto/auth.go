package dto

type LoginUserDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserResponseDto struct {
	Token     string `json:"token"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

type SignupUserRequestDto struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type SignupResponseDto struct {
	Token     string `json:"token"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}
