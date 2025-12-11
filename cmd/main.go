package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"tmazleo/api-instagram/config"
	"tmazleo/api-instagram/handlers"
	"tmazleo/api-instagram/loaders"
	"tmazleo/api-instagram/routers"

	"github.com/gin-gonic/gin"
)

const (
	APIVersion   = "v24.0"
	ProjectID    = "worlddata-439415"
	DatasetID    = "Midias_Instagram"
	TablePosts   = "postagens2"
	TableStories = "Story"
	TablePerfil  = "perfil"
)

var apiToken string
var instagramID string

func init() {
	config.LoadDotEnv(&apiToken, &instagramID)
}

// ExtractProfileData executa a extração de dados do perfil do Instagram
func ExtractProfileData(ctx context.Context, client *http.Client) error {
	profile, err := handlers.FetchProfileData(ctx, client, APIVersion, instagramID, apiToken)
	if err != nil {
		return fmt.Errorf("erro ao extrair dados do perfil: %w", err)
	}

	// Carrega no BigQuery
	if err := loaders.LoadProfileToBigQuery(ctx, ProjectID, DatasetID, TablePerfil, profile); err != nil {
		return fmt.Errorf("erro ao carregar perfil no BigQuery: %w", err)
	}

	fmt.Printf("✓ Perfil extraído e carregado com sucesso: %s (Seguidores: %d)\n", profile.Username, profile.FollowersCount)
	return nil
}

// ExtractStoriesData executa a extração de dados de Stories do Instagram
func ExtractStoriesData(ctx context.Context, client *http.Client) error {
	stories, err := handlers.FetchInstagramStories(ctx, client, APIVersion, instagramID, apiToken)
	if err != nil {
		return fmt.Errorf("erro ao extrair dados de Stories: %w", err)
	}

	if stories == nil {
		fmt.Println("✓ Nenhuma Story ativa para extrair.")
		return nil
	}

	// Carrega no BigQuery
	if err := loaders.LoadStoriesToBigQuery(ctx, ProjectID, DatasetID, TableStories, stories); err != nil {
		return fmt.Errorf("erro ao carregar stories no BigQuery: %w", err)
	}

	fmt.Printf("✓ %d Stories extraídas e carregadas com sucesso\n", len(stories))
	return nil
}

// ExtractMediaData executa a extração de dados de Mídias do Instagram
func ExtractMediaData(ctx context.Context, client *http.Client) error {
	medias, err := handlers.FetchInstagramMedia(ctx, client, APIVersion, instagramID, apiToken)
	if err != nil {
		return fmt.Errorf("erro ao extrair dados de Mídias: %w", err)
	}

	if medias == nil {
		fmt.Println("✓ Nenhuma mídia ativa para extrair.")
		return nil
	}
	// Carrega no BigQuery
	if err := loaders.LoadMediaToBigQuery(ctx, ProjectID, DatasetID, TablePosts, medias); err != nil {
		return fmt.Errorf("erro ao carregar mídias no BigQuery: %w", err)
	}

	fmt.Printf("✓ %d Mídias extraídas e carregadas com sucesso\n", len(medias))
	return nil
}

// InitializeRouter inicia o servidor Gin com as rotas registradas
func InitializeRouter() {
	router := gin.Default()
	routers.RegisterRoutes(router)
	router.Run()
}

// main executa a extração de dados e inicia o servidor
func main() {
	// Cria contexto com timeout de 2 minutos para todas as extrações
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Client HTTP com timeout padrão
	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Println("========== INICIANDO PIPELINE DE EXTRAÇÃO DE DADOS ==========")
	fmt.Println()

	// Executa a extração de perfil
	if err := ExtractProfileData(ctx, client); err != nil {
		fmt.Printf("✗ %v\n", err)
	}

	// Executa a extração de Stories
	if err := ExtractStoriesData(ctx, client); err != nil {
		fmt.Printf("✗ %v\n", err)
	}

	// Executa a extração de Mídias
	if err := ExtractMediaData(ctx, client); err != nil {
		fmt.Printf("✗ %v\n", err)
	}

	fmt.Println()
	fmt.Println("========== PIPELINE CONCLUÍDO - INICIANDO SERVIDOR ==========")

	// Inicia o servidor Gin
	InitializeRouter()
}
