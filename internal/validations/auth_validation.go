package validations

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"task-management/internal/dtos"
)

// ValidateRegisterRequest validates the registration payload.
func ValidateRegisterRequest(req dtos.RegisterRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Name, validation.Required.Error("name is required"), validation.Length(1, 100)),
		validation.Field(&req.Email, validation.Required.Error("email is required"), is.Email.Error("email must be a valid email address"), validation.Length(1, 255)),
		validation.Field(&req.Password, validation.Required.Error("password is required"), validation.Length(8, 72).Error("password must be between 8 and 72 characters")),
		validation.Field(&req.TeamID, validation.Required.Error("team_id is required"), validation.By(func(value interface{}) error {
			if id, ok := value.(uuid.UUID); !ok || id == uuid.Nil {
				return validation.NewError("validation_required", "team_id is required")
			}
			return nil
		})),
	)
}

// ValidateLoginRequest validates the login payload.
func ValidateLoginRequest(req dtos.LoginRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Email, validation.Required.Error("email is required"), is.Email.Error("email must be a valid email address"), validation.Length(1, 255)),
		validation.Field(&req.Password, validation.Required.Error("password is required"), validation.Length(8, 72).Error("password must be between 8 and 72 characters")),
	)
}

