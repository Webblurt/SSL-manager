package routes

import (
	"errors"
	"net/http"

	controllers "ssl-manager/internal/api/controllers"
	services "ssl-manager/internal/services"
	utils "ssl-manager/internal/utils"
)

func CreateRoutes(service services.ServiceInterface, cfg *utils.Config, log *utils.Logger) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("service is nil")
	}

	domains := controllers.NewController(service, cfg, log)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			domains.HandleGetDomains()(w, r)
		case http.MethodPost:
			domains.HandleCreateDomain()(w, r)
		case http.MethodDelete:
			domains.HandleDeleteDomain()(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux, nil
}
