package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"tmazleo/api-instagram/models"
	"tmazleo/api-instagram/routers"
)

// FetchProfileData extrai dados de perfil do Instagram via Graph API.
func FetchProfileData(ctx context.Context, client *http.Client, apiVersion, igUserID, token string) (*models.InstagramProfile, error) {
	fmt.Println("\n   >>> Iniciando Fluxo de Extração de Perfil <<<")

	// Campos necessários: followers_count, media_count e username
	fields := "followers_count,media_count,username"
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var profile models.InstagramProfile
	_, err := routers.MakeAPICall(ctx, client, url, &profile)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição de Perfil: %w", err)
	}

	// Adiciona metadados de ETL antes de carregar
	profile.ExtractionTimestamp = time.Now()
	profile.AccountID = igUserID // Garante que o ID do usuário está no struct

	fmt.Printf("   > Dados de Perfil (%s) coletados com sucesso. Seguidores: %d\n", profile.Username, profile.FollowersCount)

	return &profile, nil
}

// FetchInstagramStories extrai dados de Stories do Instagram via Graph API.
func FetchInstagramStories(ctx context.Context, client *http.Client, apiVersion, igUserID, token string) ([]*models.InstagramStory, error) {
	fmt.Println("\n   >>> Iniciando Fluxo de Extração de Stories <<<")

	// 1. PRIMEIRA CHAMADA: Obter lista de Stories Ativos
	fields := "id,media_type,media_url,timestamp,username"
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/stories?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var storiesResponse models.MediaResponse
	_, err := routers.MakeAPICall(ctx, client, url, &storiesResponse)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição de Stories: %w", err)
	}

	if len(storiesResponse.Data) == 0 {
		fmt.Println("   > Nenhuma Story encontrada.")
		return nil, nil
	}

	// 2. SEGUNDA CHAMADA: Loop para Insights de Cada Story
	var finalStoriesData []*models.InstagramStory
	now := time.Now()

	fmt.Printf("   > Coletando Insights para %d Stories...\n", len(storiesResponse.Data))

	for i, storyMedia := range storiesResponse.Data {
		fmt.Printf("      - Processando Story %d/%d: %s\n", i+1, len(storiesResponse.Data), storyMedia.MediaID)

		insightMetrics := "impressions,reach,replies,exits,taps_forward,taps_back"
		insightURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/insights?metric=%s&access_token=%s",
			apiVersion, storyMedia.MediaID, insightMetrics, token)

		var insightResponse models.InsightResponse
		if _, err := routers.MakeAPICall(ctx, client, insightURL, &insightResponse); err != nil {
			log.Printf("Aviso: Falha ao obter insights para Story %s: %v. Story será carregado sem insights.", storyMedia.MediaID, err)
		}

		story := &models.InstagramStory{
			StoryID:             storyMedia.MediaID,
			AccountID:           igUserID,
			Owner:               storyMedia.Owner,
			MediaURL:            storyMedia.MediaURL,
			MediaType:           storyMedia.MediaType,
			Timestamp:           storyMedia.Timestamp,
			ExtractionTimestamp: now,
		}

		for _, insight := range insightResponse.Data {
			switch insight.Name {
			case "impressions":
				story.Impressions = insight.Value
			case "reach":
				story.Reach = insight.Value
			case "replies":
				story.Replies = insight.Value
			case "exits":
				story.Exits = insight.Value
			case "taps_forward":
				story.TapsForward = insight.Value
			case "taps_back":
				story.TapsBack = insight.Value
			}
		}
		finalStoriesData = append(finalStoriesData, story)
	}

	return finalStoriesData, nil
}

// FetchInstagramMedia extrai dados de Mídias (posts, reels, etc) do Instagram via Graph API.
func FetchInstagramMedia(ctx context.Context, client *http.Client, apiVersion, igUserID, token string) ([]*models.InstagramMedia, error) {
	fmt.Println("   > Buscando IDs de Mídias...")

	fields := "id,caption,media_type,media_product_type,timestamp,like_count,comments_count,permalink,shortcode,media_url,username"
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/media?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var mediaResponse models.MediaResponse
	_, err := routers.MakeAPICall(ctx, client, url, &mediaResponse)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição de Mídias: %w", err)
	}

	if len(mediaResponse.Data) == 0 {
		fmt.Println("   > Nenhuma mídia encontrada para análise.")
		return nil, nil
	}

	var finalMediaData []*models.InstagramMedia

	fmt.Printf("   > Coletando Insights para %d mídias...\n", len(mediaResponse.Data))

	for i, media := range mediaResponse.Data {
		fmt.Printf("      - Processando mídia %d/%d: %s\n", i+1, len(mediaResponse.Data), media.MediaID)

		metricAttempts := []string{
			"impressions,reach,saved,video_views",
			"impressions,reach,saved",
			"reach,saved",
		}

		var insightResponse models.InsightResponse
		var insightErr error

		for _, metrics := range metricAttempts {
			insightURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/insights?metric=%s&access_token=%s",
				apiVersion, media.MediaID, metrics, token)

			_, insightErr = routers.MakeAPICall(ctx, client, insightURL, &insightResponse)

			if insightErr == nil {
				break
			}
		}

		if insightErr != nil {
			log.Printf("Aviso: Falha PERMANENTE ao obter insights para %s. Mídia será carregada sem insights.", media.MediaID)
		} else {
			for _, insight := range insightResponse.Data {
				switch insight.Name {
				case "impressions":
					media.Impressions = insight.Value
				case "reach":
					media.Reach = insight.Value
				case "saved":
					media.Saved = insight.Value
				case "video_views":
					media.Views = insight.Value
				}
			}
		}

		media.AccountID = igUserID
		media.ExtractionTimestamp = time.Now()

		finalMediaData = append(finalMediaData, &media)
	}

	return finalMediaData, nil
}
