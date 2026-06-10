package main

import (
	"main/config"
	"main/models"
	"main/routes"
	"main/services"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (dev/Air). In production containers env vars
	// are injected directly — missing .env file is not an error there.
	_ = godotenv.Load()
	db, err := config.InitDB()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	// Migrate your Vin model automatically
	err = db.AutoMigrate(
		&models.Vehicle{},
		&models.PartCategory{},
		&models.User{},
		&models.AgentNote{},
		&models.FieldPermission{},
		&models.VehicleFieldHistory{},
		&models.CatalogPart{},
		&models.PartFitmentRule{},
		// --- build-number fork/range system ---
		&models.ForkField{},
		&models.FieldRange{},
		&models.FieldPoint{},
		&models.KnownVIN{},
	)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Seed the default build-number fork fields (idempotent).
	if err := services.SeedDefaultForkFields(db); err != nil {
		log.Printf("warning: seeding default fork fields failed: %v", err)
	}

	r := gin.Default()
	routes.Setup(r, db)
	r.Run(":8080")
}
