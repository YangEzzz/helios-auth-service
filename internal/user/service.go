package user

import (
	"context"
	"helios-auth-service/internal/models"
)

type Service interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
}

type service struct {
	dao       Dao
	jwtSecret string
}
