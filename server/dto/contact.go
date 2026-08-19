package dto

type ListContactsQuery struct {
	InstanceID string `param:"instance" validate:"required" swaggerignore:"true"`
}

type FindContactRequest struct {
	InstanceID string `param:"instance" validate:"required" swaggerignore:"true"`
	Number     string `json:"number" validate:"required" example:"5511999999999"`
}

