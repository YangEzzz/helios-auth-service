package audit

import (
	"context"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	LogAction(ctx context.Context, userID *uuid.UUID, action, resource, details, ipAddress string) error
}

type service struct {
	dao Dao
}

func NewService(dao Dao) Service {
	return &service{dao: dao}
}

func (s *service) LogAction(ctx context.Context, userID *uuid.UUID, action, resource, details, ipAddress string) error {
	log := models.NewAuditLog(userID, action, resource, details, ipAddress)
	return s.dao.CreateAuditLog(log)
}
