package services

import (
	"context"
	clients "ssl-manager/internal/clients"
	models "ssl-manager/internal/models"
	repositories "ssl-manager/internal/repositories"
	utils "ssl-manager/internal/utils"
	"time"
)

type ServiceInterface interface {
	Validate(token string) (string, error)
	GetDomains(filters models.GetDomainsReq) (models.GetDomainsResp, error)
	CreateDomain(req models.CreateDomainReq) (string, error)
	DeleteDomain(filters models.DeleteDomainReq) error
}

type Service struct {
	client     *clients.Client
	repository *repositories.Repository
	log        *utils.Logger
	cfg        *utils.Config
	ctx        context.Context
}

func NewService(cfg *utils.Config, client *clients.Client, repo *repositories.Repository, log *utils.Logger) (*Service, error) {
	ctx := context.Background()

	return &Service{
		client:     client,
		repository: repo,
		log:        log,
		cfg:        cfg,
		ctx:        ctx,
	}, nil
}

func (s *Service) createFailureEvent(domainID string, err error) {
	tx, err := s.repository.BeginTx(s.ctx)
	defer func() {
		if err != nil {
			s.log.Warn("Rollback started")
			if rollbackErr := tx.Rollback(s.ctx); rollbackErr != nil {
				s.log.Error("Rollback error: ", rollbackErr)
			}
		}
	}()
	event := models.Entity{
		EntityName: "events",
		StringParameters: map[string]string{
			"domain_id":  domainID,
			"event_type": "renewal_failed",
			"message":    err.Error(),
			"created_by": "system-renewal",
		},
		IntegerParameters: make(map[string]int),
		TimeParameters: map[string]time.Time{
			"created_at": time.Now(),
		},
		BoolParameters: make(map[string]bool),
	}

	_, insertErr := s.repository.InsertTx(s.ctx, tx, event)
	if insertErr != nil {
		s.log.Error("Failed to log renewal error:", insertErr)
	}
	err = tx.Commit(s.ctx)
	if err != nil {
		s.log.Error("Error while commit transaction: ", err)
	}
}
