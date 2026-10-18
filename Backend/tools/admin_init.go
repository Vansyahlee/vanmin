package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"furi-backend/internal/config"
	"furi-backend/internal/database"
	"furi-backend/services"
)

func main() {
	envFile := flag.String("env", "", "Lokasi file environment")
	email := flag.String("email", "", "Email admin")
	password := flag.String("password", "", "Password admin")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("email dan password wajib diisi")
	}

	if err := config.Load(*envFile); err != nil {
		log.Fatal(err)
	}

	pool, err := database.Open(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	user, err := database.FindUserByEmail(
		context.Background(),
		pool,
		*email,
	)
	if err != nil {
		log.Fatal(err)
	}

	hash, err := services.HashPassword(*password)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.UpdateUserPassword(
		context.Background(),
		pool,
		user.ID,
		hash,
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Password berhasil dibuat untuk %s\n", user.Email)
}
