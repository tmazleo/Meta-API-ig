package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/joho/godotenv"

	"tmazleo/api-instagram/models"
	"tmazleo/api-instagram/sql"
)

const (
	ProjectID    = "worlddata-439415"
	DatasetID    = "Midias_Instagram"
	TablePosts   = "postagens_meta"
	TableStories = "story_meta"
	TablePerfil  = "perfil_meta"
)
const (
	APIVersion = "v24.0"
)

var apiToken string
var instagramID string

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Aviso: Não foi possível carregar o arquivo .env:", err)
	}

	apiToken = os.Getenv("API_TOKEN")
	instagramID = os.Getenv("USER_ID")

	if apiToken == "" || instagramID == "" {
		log.Fatal("As variáveis de ambiente API_TOKEN e/ou USER_ID não estão definidas.")
	}
}

func main() {
    ctx := context.Background()
    fmt.Println("Iniciando extração de dados do Instagram...")

    // 1. Extração de Dados Básicos do Perfil
    profileData, err := fetchProfileData(APIVersion, instagramID, apiToken)
    if err != nil {
        log.Fatalf("Erro fatal na extração de Perfil: %v", err)
    }

    // 2. Extração de Insights do Perfil (Impressões, Alcance, Cliques, etc.)
    if profileData != nil {
        fmt.Println("   > Coletando Insights de performance da conta...")
        

        // 3. Carga do Perfil Completo no BigQuery
        err = loadProfileToBigQuery(ctx, profileData)
        if err != nil {
            log.Fatalf("Erro ao carregar Perfil no BigQuery: %v", err)
        }
        fmt.Println("Carga de Perfil em BigQuery concluída com sucesso.")
    }

    // 4. Extração e Carga de Mídias (Posts/Reels)
    mediaData, err := fetchInstagramData(APIVersion, instagramID, apiToken)
    if err != nil {
        log.Fatalf("Erro fatal na extração de mídias: %v", err)
    }

    fmt.Printf("Extração de Mídias Concluída. Total: %d\n", len(mediaData))

    if len(mediaData) > 0 {
        err = loadToBigQuery(ctx, mediaData)
        if err != nil {
            log.Fatalf("Erro ao carregar Mídias no BigQuery: %v", err)
        }
        fmt.Println("Carga de Mídias em BigQuery concluída com sucesso.")
    }

    // 5. Extração e Carga de Stories
    storiesData, err := fetchInstagramStories(APIVersion, instagramID, apiToken)
    if err != nil {
        log.Fatalf("Erro fatal na extração de Stories: %v", err)
    }

    if len(storiesData) > 0 {
        err = loadStoriesToBigQuery(ctx, storiesData)
        if err != nil {
            log.Fatalf("Erro ao carregar Stories no BigQuery: %v", err)
        }
        fmt.Println("Carga de Stories em BigQuery concluída com sucesso.")
    }

    // 6. Execução das Transformações SQL (Merge)
    fmt.Println("\nIniciando transformações SQL no BigQuery...")
    runBigQueryTransformations(ctx)
    
    fmt.Println("\nProcesso ETL de Instagram concluído com sucesso.")
}
func fetchProfileData(apiVersion, igUserID, token string) (*models.InstagramProfile, error) {
    fmt.Printf("\n   >>> Iniciando Extração de Perfil para: %s <<<\n", igUserID)

    fields := "id,ig_id,username,followers_count,follows_count,media_count,profile_picture_url"
    url := fmt.Sprintf("https://graph.facebook.com/%s/%s?fields=%s&access_token=%s", apiVersion, igUserID, fields, token)

    // Usamos um mapa temporário para evitar erros de tipo no JSON
    var result map[string]interface{}
    if err := makeAPICall(url, &result); err != nil {
        return nil, fmt.Errorf("erro na requisição de Perfil: %w", err)
    }

    // Função auxiliar para converter com segurança interface{} para int32
    toInt32 := func(v interface{}) int32 {
        if v == nil {
            return 0
        }
        if val, ok := v.(float64); ok {
            return int32(val)
        }
        return 0
    }

    // Função auxiliar para converter interface{} para string
    toString := func(v interface{}) string {
        if v == nil {
            return ""
        }
        return fmt.Sprintf("%v", v)
    }

    profile := &models.InstagramProfile{
        AccountID:           toString(result["id"]),
        IgID:                toString(result["ig_id"]),
        Username:            toString(result["username"]),
        FollowersCount:      toInt32(result["followers_count"]),
        FollowsCount:        toInt32(result["follows_count"]),
        MediaCount:          toInt32(result["media_count"]),
        ProfilePictureURL:   toString(result["profile_picture_url"]),
        ExtractionTimestamp: time.Now(),
        Date:                time.Now(),
    }


    fmt.Println("   --------------------------------------------------")
    fmt.Printf("   [DEBUG PERFIL] Username: %s\n", profile.Username)
    fmt.Printf("   [DEBUG PERFIL] ID Meta: %s | IG ID: %s\n", profile.AccountID, profile.IgID)
    fmt.Printf("   [DEBUG PERFIL] Seguidores: %d\n", profile.FollowersCount)
    fmt.Printf("   [DEBUG PERFIL] Seguindo: %d\n", profile.FollowsCount)
    fmt.Printf("   [DEBUG PERFIL] Total de Mídias: %d\n", profile.MediaCount)
    fmt.Println("   --------------------------------------------------")

    return profile, nil
}


func fetchInstagramStories(apiVersion, igUserID, token string) ([]*models.InstagramStory, error) {
	fmt.Println("\n   >>> Iniciando Fluxo de Extração de Stories <<<")

	fields := "id,media_type,media_url,timestamp,username"
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/stories?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var storiesResponse models.MediaResponse
	if err := makeAPICall(url, &storiesResponse); err != nil {
		return nil, fmt.Errorf("erro na requisição de Stories: %w", err)
	}

	if len(storiesResponse.Data) == 0 {
		fmt.Println("   > Nenhuma Story encontrada.")
		return nil, nil
	}

	var finalStoriesData []*models.InstagramStory

	fmt.Printf("   > Coletando Insights para %d Stories...\n", len(storiesResponse.Data))

	for i, storyMedia := range storiesResponse.Data {
		fmt.Printf("      - Processando Story %d/%d: %s\n", i+1, len(storiesResponse.Data), storyMedia.MediaID)

		insightMetrics := "impressions,reach,replies,exits,taps_forward,taps_back"
		insightURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/insights?metric=%s&access_token=%s",
			apiVersion, storyMedia.MediaID, insightMetrics, token)

		var insightResponse models.InsightResponse
		if err := makeAPICall(insightURL, &insightResponse); err != nil {
			log.Printf("Aviso: Falha ao obter insights para Story %s: %v. Story será carregado sem insights.", storyMedia.MediaID, err)
		}

		story := &models.InstagramStory{
			StoryID:   storyMedia.MediaID,
			AccountID: igUserID,
			OwnerID:   igUserID,
			Username:  storyMedia.Username,
			MediaURL:  storyMedia.MediaURL,
			MediaType: storyMedia.MediaType,
			Timestamp: storyMedia.Timestamp,
		}

		for _, insight := range insightResponse.Data {
			if len(insight.Values) > 0 {
				valorReal := insight.Values[0].Value

				switch insight.Name {
				case "impressions":
					story.Impressions = valorReal
				case "reach":
					story.Reach = valorReal
				case "replies":
					story.Replies = valorReal
				case "exits":
					story.Exits = valorReal
				case "taps_forward":
					story.TapsForward = valorReal
				case "taps_back":
					story.TapsBack = valorReal
				}
			}
		}
		
		finalStoriesData = append(finalStoriesData, story)
	}

	return finalStoriesData, nil
}

func fetchInstagramData(apiVersion, igUserID, token string) ([]*models.InstagramMedia, error) {
    var allMedia []*models.InstagramMedia

    fields := "id,caption,media_type,media_product_type,timestamp,like_count,comments_count,permalink,shortcode,media_url,username"
    nextURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/media?fields=%s&limit=100&access_token=%s",
        apiVersion, igUserID, fields, token)

    fmt.Println("   > Iniciando busca exaustiva de todas as mídias...")

    for nextURL != "" {
        var mediaResponse models.MediaResponse
        if err := makeAPICall(nextURL, &mediaResponse); err != nil {
            return nil, fmt.Errorf("erro na requisição de Mídias: %w", err)
        }

        if len(mediaResponse.Data) == 0 {
            break
        }

        fmt.Printf("   > Coletando Insights para lote de %d mídias...\n", len(mediaResponse.Data))

        for i := range mediaResponse.Data {
            media := &mediaResponse.Data[i] 

            if (i+1)%20 == 0 {
                fmt.Printf("      - Processando... (%d mídias coletadas até agora)\n", len(allMedia)+i+1)
            }

            metricAttempts := []string{
                "impressions,reach,saved,video_views,replies,shares",
                "impressions,reach,saved,replies,shares",
                "reach,saved,shares",
                "reach,saved",
            }

            var insightResponse models.InsightResponse
            var insightErr error
            for _, metrics := range metricAttempts {
                insightURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/insights?metric=%s&access_token=%s",
                    apiVersion, media.MediaID, metrics, token)
                insightErr = makeAPICall(insightURL, &insightResponse)
                if insightErr == nil {
                    break
                }
            }

            media.AccountID = igUserID
            media.OwnerID = igUserID
            media.ExtractionTimestamp = time.Now()
            media.TimestampBQ = media.Timestamp.Value

            // ATENÇÃO: Bloco de atribuição corrigido
            if insightErr == nil {
                for _, insight := range insightResponse.Data {
                    if len(insight.Values) > 0 {
                        valorReal := insight.Values[0].Value

                        switch insight.Name {
                        case "impressions":
                            media.Impressions = valorReal
                        case "reach":
                            media.Reach = valorReal
                        case "saved":
                            media.Saved = valorReal
                        case "video_views":
                            media.Views = valorReal
                        case "replies":
                            media.Replies = valorReal
                        case "shares":
                            media.Shares = valorReal
                        }
                    }
                }
            } // Chave que fecha o 'if insightErr == nil'

            // CÁLCULO: Realizado para cada mídia
            media.TotalInteractions = media.LikeCount + media.CommentsCount + media.Replies + media.Saved + media.Shares
            
            // DEBUG
            fmt.Printf("   [DEBUG] ID: %s | L: %d | C: %d | R: %d | S: %d | TOTAL: %d\n", 
                media.MediaID, media.LikeCount, media.CommentsCount, media.Replies, media.Shares, media.TotalInteractions)
            
            allMedia = append(allMedia, media)
        } // Chave que fecha o loop 'for i := range'

        nextURL = mediaResponse.Paging.Next
    } // Chave que fecha o loop 'for nextURL != ""'

    fmt.Printf("   > Total final de mídias extraídas: %d\n", len(allMedia))
    return allMedia, nil
}

func makeAPICall(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("erro na requisição GET: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler a resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro na API. Status: %s. Resposta: %s", resp.Status, string(bodyBytes))
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	return nil
}

func loadProfileToBigQuery(ctx context.Context, data *models.InstagramProfile) error {

	client, err := bigquery.NewClient(ctx, ProjectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(DatasetID).Table(TablePerfil).Inserter()

	items := []interface{}{data}

	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção de Perfil na linha %d: %v", i, e.Error())
			}
			return fmt.Errorf("erros parciais ou totais na inserção de Perfil.")
		}
		return fmt.Errorf("erro fatal ao inserir Perfil no BigQuery: %w", err)
	}

	return nil
}
func loadStoriesToBigQuery(ctx context.Context, data []*models.InstagramStory) error {

	client, err := bigquery.NewClient(ctx, ProjectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(DatasetID).Table(TableStories).Inserter()

	items := make([]interface{}, len(data))
	for i, story := range data {
		items[i] = story
	}

	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção de Story na linha %d: %v", i, e.Error())
			}
			return fmt.Errorf("erros parciais ou totais na inserção de Stories.")
		}
		return fmt.Errorf("erro fatal ao inserir Stories no BigQuery: %w", err)
	}

	return nil
}
func loadToBigQuery(ctx context.Context, data []*models.InstagramMedia) error {

	client, err := bigquery.NewClient(ctx, ProjectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(DatasetID).Table(TablePosts).Inserter()

	items := make([]interface{}, len(data))
	for i, media := range data {
		items[i] = media
	}

	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção na linha %d: %v", i, e.Error())
			}
			return fmt.Errorf("erros parciais ou totais na inserção. Detalhes acima.")
		}
		return fmt.Errorf("erro fatal ao inserir dados no BigQuery: %w", err)
	}

	return nil
}
func runBigQueryTransformations(ctx context.Context) {
	client, err := bigquery.NewClient(ctx, ProjectID)
	if err != nil {
		log.Printf("Falha ao conectar BigQuery para tratamento: %v", err)
		return
	}
	defer client.Close()

	queries := []string{
		sql.MergePerfil,
		sql.MergePostagens,
		sql.MergeStories,
	}

	fmt.Println("\n   >>> Iniciando Tratamento e Deduplicação via MERGE <<<")
	for i, queryStr := range queries {
		q := client.Query(queryStr)
		job, err := q.Run(ctx)
		if err != nil {
			log.Printf("❌ Erro ao iniciar Job da Query %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   > Aguardando processamento da Query %d...\n", i+1)
		status, err := job.Wait(ctx)
		if err != nil {
			log.Printf("❌ Falha na espera do Job %d: %v", i+1, err)
			continue
		}

		if status != nil && status.Err() != nil {
			log.Printf("❌ Erro na execução da Query %d: %v", i+1, status.Err())
			continue
		}

		fmt.Printf("   ✅ Sucesso: Tabela %d processada.\n", i+1)
	}
}
