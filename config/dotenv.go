package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)
func LoadDotEnv(apiToken *string, instagramID *string) {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Aviso: Não foi possível carregar o arquivo .env:", err)
	}

	*apiToken = os.Getenv("API_TOKEN")
	*instagramID = os.Getenv("USER_ID")

	if *apiToken == "" || *instagramID == "" {
		log.Fatal("As variáveis de ambiente API_TOKEN e/ou USER_ID não estão definidas.")
	}
}