package request

type SignInRequest struct {
	Email    string `json:"email" validate:"email,required"`
	Password string `json:"password" validate:"min=8,required"`
}

type SignUpRequest struct {
	Name                 string `json:"name" validate:"required"`
	Email                string `json:"email" validate:"email,required"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,min=8"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"password,omitempty"`
	NewPassword     string `json:"password_new" validate:"required"`
	ConfirmPassword string `json:"password_confirmation" validate:"required"`
}

type UpdateDataUserRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"email,required"`
	Phone   string `json:"phone" validate:"required"`
	Address string `json:"address" validate:"required"`
	Lat     string `json:"lat" validate:"required"`
	Lng     string `json:"lng" validate:"required"`
	Photo   string `json:"photo" validate:"required"`
}
