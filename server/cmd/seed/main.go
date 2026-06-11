// cmd/seed/main.go — one-time admin bootstrap.
// Run once from the server directory:
//
//	go run ./cmd/seed
package main

import (
	"fmt"
	"log"
	"main/config"
	"main/models"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := config.InitTestDB()
	if err != nil {
		log.Fatalf("DB: %v", err)
	}

	// AutoMigrate so the new IsAdmin / VIN columns exist
	if err := db.AutoMigrate(&models.User{}, &models.Vehicle{}, &models.PartCategory{}, &models.AgentNote{}); err != nil {
		log.Fatalf("migration: %v", err)
	}

	const (
		email    = "admin@vindecoder.local"
		username = "admin"
		password = "Admin@2026!"
	)

	// Check if already exists
	var existing models.User
	if db.Where("username = ?", username).First(&existing).Error == nil {
		// Already there — ensure the role is admin
		db.Model(&existing).Update("role", "admin")
		fmt.Printf("\n✓ User '%s' already exists — role set to admin.\n\n", username)
		printCreds(email, username, password)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	admin := models.User{
		Email:          email,
		Username:       username,
		HashedPassword: string(hash),
		IsActive:       true,
		Role:           "admin",
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("\n✓ Admin user created (ID=%d).\n\n", admin.ID)
	printCreds(email, username, password)
}

func printCreds(email, username, password string) {
	fmt.Println("┌─────────────────────────────────────┐")
	fmt.Println("│          ADMIN CREDENTIALS          │")
	fmt.Println("├─────────────────────────────────────┤")
	fmt.Printf("│  Email    : %-24s│\n", email)
	fmt.Printf("│  Username : %-24s│\n", username)
	fmt.Printf("│  Password : %-24s│\n", password)
	fmt.Println("└─────────────────────────────────────┘")
	fmt.Println("\nChange the password after first login.")
}
