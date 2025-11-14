package models

import "time"

type GetDomainsDTO struct {
	TotalPages    int
	Page          int
	PageSize      int
	TotalElements int
	HasNext       bool
	HasPrev       bool
	NextPage      int
	PrevPage      int
	Domains       []DomainsDTO
}

type DomainsDTO struct {
	ID         string
	DomainName string
	Details    DetailsDTO
}

type DetailsDTO struct {
	Status              string
	AutoRenew           bool
	VerificationMethod  string
	CreatedAt           time.Time
	CreatedBy           string
	DomainLastUpdate    *time.Time
	NginxContainerName  string
	CertValidTo         *time.Time
	CertLastRenewal     *time.Time
	CertRenewalAttempts *int
}
