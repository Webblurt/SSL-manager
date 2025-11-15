package services

import (
	"fmt"
	models "ssl-manager/internal/models"
	"time"
)

func (s *Service) GetDomains(filters models.GetDomainsReq) (models.GetDomainsResp, error) {
	s.log.Debug("Fetching list of domains started............")
	offset := (filters.Page - 1) * filters.PageSize
	repoFilters := models.DomainsFilters{
		DomainName: filters.DomainName,
		Status:     filters.Status,
		UserID:     filters.UserID,
		Limit:      &filters.PageSize,
		Offset:     &offset,
	}

	s.log.Debug("Fetching stocks count from repo...")
	totalElements, err := s.repository.GetDomainsCount(s.ctx, repoFilters)
	if err != nil {
		s.log.Error("Error while getting total stock count: ", err)
		return models.GetDomainsResp{}, err
	}
	s.log.Debug("Count: ", totalElements)

	totalPages := (totalElements + filters.Page - 1) / filters.PageSize
	hasNext := filters.Page < totalPages
	hasPrev := filters.Page > 1
	nextPage := 0
	prevPage := 0
	if hasNext {
		nextPage = filters.Page + 1
	}
	if hasPrev {
		prevPage = filters.Page - 1
	}

	s.log.Debug("Fetching list of domains from repo...")
	domains, err := s.repository.GetDomainsList(s.ctx, repoFilters)
	if err != nil {
		s.log.Error("Error while getting list of domains: ", err)
		return models.GetDomainsResp{}, err
	}
	s.log.Debug("List of domains: ", domains)

	var d []models.Domains
	for _, domain := range domains {
		d = append(d, models.ConvertDomainsDTOToDomains(domain))
	}

	return models.GetDomainsResp{
		TotalPages:    totalPages,
		Page:          filters.Page,
		PageSize:      filters.PageSize,
		TotalElements: totalElements,
		HasNext:       hasNext,
		HasPrev:       hasPrev,
		NextPage:      nextPage,
		PrevPage:      prevPage,
		Domains:       d,
	}, nil
}

func (s *Service) CreateDomain(req models.CreateDomainReq) (string, error) {
	s.log.Debug("Creating domain...............")
	tx, err := s.repository.BeginTx(s.ctx)
	if err != nil {
		s.log.Error("Error start transaction while itinerary creation: ", err)
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.log.Warn("Rollback started")
			if rollbackErr := tx.Rollback(s.ctx); rollbackErr != nil {
				s.log.Error("Rollback error: ", rollbackErr)
			}
		}
	}()

	// check for existance
	exists, err := s.repository.IsDomainExists(s.ctx, req.Domain)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("domain already exists")
	}

	// adding to db
	domainEntity := models.Entity{
		EntityName: "domains",
		StringParameters: map[string]string{
			"domain_name":         req.Domain,
			"status":              "pending",
			"verification_method": req.VerificationMethod,
			"created_by":          req.CreatedBy,
		},
		IntegerParameters: make(map[string]int),
		TimeParameters:    make(map[string]time.Time),
		BoolParameters: map[string]bool{
			"auto_renew": req.AutoRenew,
		},
	}
	domainID, err := s.repository.InsertTx(s.ctx, tx, domainEntity)
	if err != nil {
		s.log.Error("Error while creating domain: ", err)
		return "", err
	}

	// calling client to create cert
	certData, err := s.client.CreateCertificate(req.Domain)
	if err != nil {
		statusEntity := models.Entity{
			EntityName: "domains",
			StringParameters: map[string]string{
				"status":     "renewal_failed",
				"updated_by": req.CreatedBy,
			},
			IntegerParameters: make(map[string]int),
			TimeParameters:    make(map[string]time.Time),
			BoolParameters:    make(map[string]bool),
		}
		err := s.repository.UpdateTx(s.ctx, tx, statusEntity, domainID)
		if err != nil {
			s.log.Error("Error while updating domain status: ", err)
			return "", err
		}

		eventEntity := models.Entity{
			EntityName: "events",
			StringParameters: map[string]string{
				"domain_id":  domainID,
				"event_type": "failed",
				"message":    fmt.Sprintf("Certificate issuance failed: %v", err),
				"created_by": req.CreatedBy,
			},
			IntegerParameters: make(map[string]int),
			TimeParameters:    make(map[string]time.Time),
			BoolParameters:    make(map[string]bool),
		}
		_, err = s.repository.InsertTx(s.ctx, tx, eventEntity)
		if err != nil {
			s.log.Error("Error while writing new event: ", err)
			return "", err
		}

		return "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// saving files and paths
	certPaths, err := s.client.SaveCertificateFiles(req.Domain, certData)
	if err != nil {
		return "", fmt.Errorf("failed to save certificate files: %w", err)
	}

	// saving certs to db
	certEntity := models.Entity{
		EntityName: "certificates",
		StringParameters: map[string]string{
			"domain_id":  domainID,
			"issuer":     "Let's Encrypt",
			"cert_path":  certPaths.Cert,
			"key_path":   certPaths.Key,
			"chain_path": certPaths.Chain,
			"created_by": req.CreatedBy,
		},
		IntegerParameters: make(map[string]int),
		TimeParameters: map[string]time.Time{
			"valid_from": certData.ValidFrom,
			"valid_to":   certData.ValidTo,
		},
		BoolParameters: make(map[string]bool),
	}
	_, err = s.repository.InsertTx(s.ctx, tx, certEntity)
	if err != nil {
		s.log.Error("Error while saving certs to db: ", err)
		return "", err
	}

	// changing domain status
	statusEntity := models.Entity{
		EntityName: "domains",
		StringParameters: map[string]string{
			"status":     "active",
			"updated_by": req.CreatedBy,
		},
		IntegerParameters: make(map[string]int),
		TimeParameters:    make(map[string]time.Time),
		BoolParameters:    make(map[string]bool),
	}
	err = s.repository.UpdateTx(s.ctx, tx, statusEntity, domainID)
	if err != nil {
		s.log.Error("Error while updating domain status: ", err)
		return "", err
	}

	// creating new event
	eventEntity := models.Entity{
		EntityName: "events",
		StringParameters: map[string]string{
			"domain_id":  domainID,
			"event_type": "created",
			"message":    "Domain and certificate created successfully",
			"created_by": req.CreatedBy,
		},
		IntegerParameters: make(map[string]int),
		TimeParameters:    make(map[string]time.Time),
		BoolParameters:    make(map[string]bool),
	}
	_, err = s.repository.InsertTx(s.ctx, tx, eventEntity)
	if err != nil {
		s.log.Error("Error while writing new event: ", err)
		return "", err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.log.Error("Error while commit transaction: ", err)
		return "", err
	}

	s.log.Debug("Domain saved")
	return domainID, nil
}

func (s *Service) DeleteDomain(filters models.DeleteDomainReq) error {
	s.log.Debug("Deleting...............")
	tx, err := s.repository.BeginTx(s.ctx)
	if err != nil {
		s.log.Error("Error start transaction while itinerary creation: ", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.log.Warn("Rollback started")
			if rollbackErr := tx.Rollback(s.ctx); rollbackErr != nil {
				s.log.Error("Rollback error: ", rollbackErr)
			}
		}
	}()

	domainID := filters.DomainID
	// check for existance
	if filters.DomainName != "" {
		domainID, err := s.repository.GetIDByNameTx(s.ctx, tx, models.Entity{
			EntityName:       "domains",
			StringParameters: map[string]string{"domain_name": filters.DomainName},
		})
		if err != nil {
			s.log.Error("Error while getting domain id: ", err)
			return err
		}
		if domainID == "" {
			return fmt.Errorf("domain don't exists")
		}
	}

	// updating status
	statusEntity := models.Entity{
		EntityName: "domains",
		StringParameters: map[string]string{
			"status":     "deleted",
			"deleted_by": filters.UserID,
			"updated_by": filters.UserID,
		},
		TimeParameters: map[string]time.Time{
			"deleted_at": time.Now(),
		},
		IntegerParameters: make(map[string]int),
		BoolParameters:    make(map[string]bool),
	}
	err = s.repository.UpdateTx(s.ctx, tx, statusEntity, domainID)
	if err != nil {
		s.log.Error("Error updating domain status: ", err)
		return err
	}

	// fetchin certificates
	certs, err := s.repository.GetCertificatesByDomain(s.ctx, domainID)
	if err != nil {
		s.log.Error("Error fetching certificates: ", err)
		return err
	}

	// deleting files
	err = s.client.DeleteCertificateFiles(certs.CertPath, certs.KeyPath, *certs.ChainPath)
	if err != nil {
		s.log.Warn(fmt.Sprintf("Error deleting certificate files for domain %s: %v", filters.DomainName, err))
	}

	// mark certs deleted
	certEntity := models.Entity{
		EntityName: "certificates",
		StringParameters: map[string]string{
			"deleted_by": filters.UserID,
			"updated_by": filters.UserID,
		},
		TimeParameters: map[string]time.Time{
			"deleted_at": time.Now(),
		},
		IntegerParameters: make(map[string]int),
		BoolParameters:    make(map[string]bool),
	}
	err = s.repository.UpdateTx(s.ctx, tx, certEntity, certs.ID)
	if err != nil {
		s.log.Error("Error updating certificate record: ", err)
		return err
	}

	// creating new event
	eventEntity := models.Entity{
		EntityName: "events",
		StringParameters: map[string]string{
			"domain_id":  domainID,
			"event_type": "deleted",
			"message":    fmt.Sprintf("Domain '%s' and its certificates deleted", filters.DomainName),
			"created_by": filters.UserID,
		},
		IntegerParameters: make(map[string]int),
		TimeParameters:    make(map[string]time.Time),
		BoolParameters:    make(map[string]bool),
	}

	_, err = s.repository.InsertTx(s.ctx, tx, eventEntity)
	if err != nil {
		s.log.Error("Error inserting event: ", err)
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.log.Error("Error while commit transaction: ", err)
		return err
	}

	s.log.Debug("Domain deleted")
	return nil
}
