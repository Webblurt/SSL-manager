package models

type DomainsFilters struct {
	Limit      *int
	Offset     *int
	DomainName string
	Status     string
	UserID     string
}
