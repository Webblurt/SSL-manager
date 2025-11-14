package main

import (
	"log"
	"net/http"
	"os"
	routes "ssl-manager/internal/api/routes"
	clients "ssl-manager/internal/clients"
	repositories "ssl-manager/internal/repositories"
	services "ssl-manager/internal/services"
	utils "ssl-manager/internal/utils"

	"github.com/joho/godotenv"
)

func main() {
	// loading .env
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("No .env file found, using system environment variables", err)
	}

	//loading configuration
	cfg, err := utils.LoadConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatal("Error loading config file", err)
	}

	// creating logger
	log := utils.NewLogger(cfg.Logger.LogLevel)

	// repository creation
	repo, err := repositories.NewRepository(cfg, log)
	if err != nil {
		log.Fatal("Error creating repository: ", err)
	}
	log.Info("Repository created successful")

	// start migrations
	if err := repo.RunMigrations(cfg); err != nil {
		log.Warn("Error running migrations: ", err)
	}
	log.Info("Migrations applied successfully")

	// creating clients for external apis
	clients, err := clients.NewClient(log, cfg)
	if err != nil {
		log.Fatal("Error creating clients: ", err)
	}
	log.Info("Clients created successful")

	// creating service
	service, err := services.NewService(cfg, clients, repo, log)
	if err != nil {
		log.Fatal("Error creating service: ", err)
	}
	log.Info("Service created successful")

	// starting scheduler
	service.StartCertificateRenewalScheduler()
	log.Info("Certificate renewal scheduler started")

	// creating routes
	router, err := routes.CreateRoutes(service, cfg, log)
	if err != nil {
		log.Fatal("Error creating routes: ", err)
	}
	log.Info("Routes created successful")

	// starting http server
	log.Info("Starting the server on port ", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, router); err != nil {
		log.Fatal("Error starting server: ", err)
	}
	log.Info("Server started successful")
}
