package audit

import (
	"context"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	LogAction(ctx context.Context, userID *uuid.UUID, action, resource, details, ipAddress string) error
	ListAuditLogs(ctx context.Context, resource string, page, pageSize int) ([]models.AuditLog, int64, error)
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

func (s *service) ListAuditLogs(ctx context.Context, resource string, page, pageSize int) ([]models.AuditLog, int64, error) {
	logs, total, err := s.dao.ListAuditLogs(resource, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	for i := range logs {
		logs[i].ActionName = constant.GetActionName(logs[i].Action)
		logs[i].ResourceName = constant.GetResourceName(logs[i].Resource)
		logs[i].DetailsName = constant.GetDetailName(logs[i].Action, logs[i].Details)
	}

	return logs, total, nil
}

