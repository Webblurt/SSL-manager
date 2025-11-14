package models

type GetDomainsReq struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	UserID     string
	Status     string `json:"status,omitempty"`
	DomainName string `json:"domain_name,omitempty"`
}

type CreateDomainReq struct {
	CreatedBy          string
	Domain             string `json:"domain"`
	VerificationMethod string `json:"verification_method"`
	AutoRenew          bool   `json:"auto_renew"`
}

type DeleteDomainReq struct {
	DomainID   string `json:"domain_id"`
	DomainName string `json:"domain_name"`
	UserID     string
}
