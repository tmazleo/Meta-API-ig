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
		// Log como aviso, pois em produção (Cloud Run) as variáveis virão do ambiente
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
	profileData, err := fetchProfileData(APIVersion, instagramID, apiToken)
	if err != nil {
		log.Fatalf("Erro fatal na extração de Perfil: %v", err)
	}

	if profileData != nil {
		err = loadProfileToBigQuery(context.Background(), profileData)
		if err != nil {
			log.Fatalf("Erro ao carregar Perfil no BigQuery: %v", err)
		}
		fmt.Println("Carga de Perfil em BigQuery concluída com sucesso.")
	}

	fmt.Println("\nProcesso ETL de Instagram concluído.")

	mediaData, err := fetchInstagramData(APIVersion, instagramID, apiToken)
	if err != nil {
		log.Fatalf("Erro fatal na extração: %v", err)
	}

	fmt.Printf("Extração de Mídias Concluída. Total de Mídias (Posts/Reels) encontradas: %d\n", len(mediaData))

	if len(mediaData) > 0 {
		err = loadToBigQuery(context.Background(), mediaData) // <<< ADICIONAR ESTA LINHA
		if err != nil {
			log.Fatalf("Erro ao carregar Mídias (Posts/Reels) no BigQuery: %v", err)
		}
		fmt.Println("Carga de Mídias (Posts/Reels) em BigQuery concluída com sucesso.") // <<< E ESTA
	} else {
		fmt.Println("Nenhuma Mídia (Post/Reel) encontrada para carregar.")
	}
	fmt.Println("Extração e Carga de Mídias concluídas.")

	storiesData, err := fetchInstagramStories(APIVersion, instagramID, apiToken)
	if err != nil {
		log.Fatalf("Erro fatal na extração de Stories: %v", err)
	}

	if len(storiesData) > 0 {
		err = loadStoriesToBigQuery(context.Background(), storiesData)
		if err != nil {
			log.Fatalf("Erro ao carregar Stories no BigQuery: %v", err)
		}
		fmt.Println("Carga de Stories em BigQuery concluída com sucesso.")
	} else {
		fmt.Println("Nenhum Story encontrado para carregar.")
	}

	runBigQueryTransformations(ctx)
	fmt.Println("\nProcesso ETL de Instagram concluído.")
}
func fetchProfileData(apiVersion, igUserID, token string) (*models.InstagramProfile, error) {
	fmt.Println("\n   >>> Iniciando Fluxo de Extração de Perfil <<<")

	// Campos necessários: followers_count, media_count e username
	fields := "followers_count,media_count,username"
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var profile models.InstagramProfile
	if err := makeAPICall(url, &profile); err != nil {
		return nil, fmt.Errorf("erro na requisição de Perfil: %w", err)
	}

	now := time.Now()
	profile.ExtractionTimestamp = now
	profile.AccountID = igUserID
	profile.Date = now
	profile.IgID = igUserID

	fmt.Printf("   > Dados de Perfil (%s) coletados com sucesso. Seguidores: %d\n", profile.Username, profile.FollowersCount)

	return &profile, nil
}
func fetchInstagramStories(apiVersion, igUserID, token string) ([]*models.InstagramStory, error) {
	fmt.Println("\n   >>> Iniciando Fluxo de Extração de Stories <<<")

	// 1. PRIMEIRA CHAMADA: Obter lista de Stories Ativos (Stories só duram 24h na API)
	fields := "id,media_type,media_url,timestamp,username" // Campos básicos do Story
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/stories?fields=%s&access_token=%s",
		apiVersion, igUserID, fields, token)

	var storiesResponse models.MediaResponse // Reutilizando a struct MediaResponse (que contém Data []InstagramMedia)
	if err := makeAPICall(url, &storiesResponse); err != nil {
		return nil, fmt.Errorf("erro na requisição de Stories: %w", err)
	}

	if len(storiesResponse.Data) == 0 {
		fmt.Println("   > Nenhuma Story encontrada.")
		return nil, nil
	}

	// 2. SEGUNDA CHAMADA: Loop para Insights de Cada Story
	var finalStoriesData []*models.InstagramStory

	fmt.Printf("   > Coletando Insights para %d Stories...\n", len(storiesResponse.Data))

	for i, storyMedia := range storiesResponse.Data {
		fmt.Printf("      - Processando Story %d/%d: %s\n", i+1, len(storiesResponse.Data), storyMedia.MediaID)

		// Métricas de Story: impressions, reach, replies, exits, taps_forward, taps_back
		insightMetrics := "impressions,reach,replies,exits,taps_forward,taps_back"
		insightURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/insights?metric=%s&access_token=%s",
			apiVersion, storyMedia.MediaID, insightMetrics, token)

		var insightResponse models.InsightResponse
		if err := makeAPICall(insightURL, &insightResponse); err != nil {
			log.Printf("Aviso: Falha ao obter insights para Story %s: %v. Story será carregado sem insights.", storyMedia.MediaID, err)
		}

		// Mapeia os dados do MediaResponse para a struct InstagramStory final
		story := &models.InstagramStory{
			StoryID:   storyMedia.MediaID,
			AccountID: igUserID,

			OwnerID:   igUserID,
			Username:  storyMedia.Username,
			MediaURL:  storyMedia.MediaURL,
			MediaType: storyMedia.MediaType,
			Timestamp: storyMedia.Timestamp,
		}

		// Combina os Insights
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

func fetchInstagramData(apiVersion, igUserID, token string) ([]*models.InstagramMedia, error) {
	var allMedia []*models.InstagramMedia

	// URL inicial para a primeira página
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

		for i, media := range mediaResponse.Data {
			// Log de progresso simplificado para não inundar o terminal
			if (i+1)%20 == 0 {
				fmt.Printf("      - Processando... (%d mídias coletadas até agora)\n", len(allMedia)+i+1)
			}

			// Lógica de Insights (Permanece a mesma que já funciona)
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
				insightErr = makeAPICall(insightURL, &insightResponse)
				if insightErr == nil {
					break
				}
			}

			// Preenchimento dos campos para o BigQuery
			media.AccountID = igUserID
			media.OwnerID = igUserID
			media.MediaID = media.MediaID
			media.ExtractionTimestamp = time.Now()
			media.TimestampBQ = media.Timestamp.Value // Sua lógica de conversão para BQ

			if insightErr == nil {
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

			// Adiciona ao slice final
			allMedia = append(allMedia, &media)
		}

		// Verifica se existe uma próxima página no objeto Paging
		// Nota: Você precisará atualizar a struct MediaResponse para incluir o campo Paging
		nextURL = mediaResponse.Paging.Next
	}

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

	// Aponta para a tabela 'perfil'
	inserter := client.Dataset(DatasetID).Table(TablePerfil).Inserter()

	// O Inserter aceita diretamente a struct, mas precisamos envolvê-la em uma slice de interface{}
	items := []interface{}{data}

	// Carrega os dados (Inserção de Streaming)
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

	// Aponta para a tabela 'Story'
	inserter := client.Dataset(DatasetID).Table(TableStories).Inserter()

	// Converte a slice []*models.InstagramStory para a interface{}
	items := make([]interface{}, len(data))
	for i, story := range data {
		items[i] = story
	}

	// Carrega os dados (Inserção de Streaming)
	if err := inserter.Put(ctx, items); err != nil {
		// Usando a lógica de tratamento de erro simples (sem InsertError)
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

	// 1. Cria o Cliente BigQuery
	client, err := bigquery.NewClient(ctx, ProjectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	// 2. Define a Tabela de Destino
	inserter := client.Dataset(DatasetID).Table(TablePosts).Inserter()

	// 3. Converte a slice []*models.InstagramMedia para o tipo genérico aceito pelo Inserter
	// O Inserter aceita diretamente slices de structs, graças às tags 'bigquery:"..."'

	// Cria uma slice de interface{} para o Inserter
	items := make([]interface{}, len(data))
	for i, media := range data {
		items[i] = media
	}

	// ...
	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				// Loga o erro de forma genérica, sem tentar acessar RowIndex/Errors,
				// que está causando o problema de compilação.
				log.Printf("Erro de inserção na linha %d: %v", i, e.Error())
			}
			return fmt.Errorf("erros parciais ou totais na inserção. Detalhes acima.")
		}
		// Retorna se o erro não for MultiError (ex: erro de conexão)
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

	// Usamos as constantes do pacote sql
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

		// Verificação de segurança: checa se status não é nulo antes de chamar Err()
		if status != nil && status.Err() != nil {
			log.Printf("❌ Erro na execução da Query %d: %v", i+1, status.Err())
			continue
		}

		fmt.Printf("   ✅ Sucesso: Tabela %d processada.\n", i+1)
	}
}
