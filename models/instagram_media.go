package models

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
)

const APITimeLayout = "2006-01-02T15:04:05-0700"

type MetaTime struct {
	Value time.Time
}

func (mt *MetaTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)
	if len(s) > 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	t, err := time.Parse(APITimeLayout, s)
	if err != nil {
		return fmt.Errorf("falha ao decodificar MetaTime '%s': %w", s, err)
	}
	mt.Value = t
	return nil
}
func (mt MetaTime) ValueTime() (bigquery.Value, error) {
    // Retorna o time.Time interno como o valor BQ
    return mt.Value, nil
}

type InstagramMedia struct {
	MediaID          string   `json:"id" bigquery:"media_id"` // ID do post/reel (ID da Mídia)
	AccountID        string   `json:"-" bigquery:"account_id"`
	OwnerID          string   `json:"-" bigquery:"owner"`
	Username         string   `json:"username" bigquery:"username"`                               // Nome de Usuário (username) - Não vem no JSON, será preenchido pelo código
	Caption          string   `json:"caption" bigquery:"caption"`                                 // Legenda
	Shortcode        string   `json:"shortcode,omitempty" bigquery:"shortcode"`                   // Shortcode (ID curto)
	MediaURL         string   `json:"media_url,omitempty" bigquery:"media_url"`                   // URL da Mídia
	Permalink        string   `json:"permalink" bigquery:"permalink"`                             // Link direto para o post
	ThumbnailURL     string   `json:"thumbnail_url,omitempty" bigquery:"thumbnail_url"`           // URL da miniatura
	MediaType        string   `json:"media_type" bigquery:"media_type"`                           // Tipo (IMAGE, VIDEO, CAROUSEL_ALBUM)
	MediaProductType string   `json:"media_product_type,omitempty" bigquery:"media_product_type"` // Tipo de produto (FEED, REEL, STORY)
	Timestamp        MetaTime `json:"timestamp" bigquery:"timestamp"`                             // Data/hora da publicação
	TimestampBQ time.Time `json:"-" bigquery:"timestamp"`

	LikeCount     int32 `json:"like_count" bigquery:"like_count"`         // Contagem de curtidas
	CommentsCount int32 `json:"comments_count" bigquery:"comments_count"` // Contagem de comentários

	Impressions         int32     `json:"-" bigquery:"impressions"`                  // Impressões
	Reach               int32     `json:"-" bigquery:"reach"`                        // Alcance
	Saved               int32     `json:"-" bigquery:"saved"`                        // Salvos
	Views               int32     `json:"-" bigquery:"views"`                        // Views (para vídeos/reels)
	ExtractionTimestamp time.Time `json:"-" bigquery:"dataddo_extraction_timestamp"` // Timestamp de quando foi feita a extração
}
type Insight struct {
	Name  string `json:"name"`
	Value int32  `json:"value"`
}
type InsightResponse struct {
	Data []Insight `json:"data"`
}

type InstagramStory struct {
	StoryID   string   `json:"id" bigquery:"media_id"`
	AccountID string   `json:"-" bigquery:"account_id"`
	OwnerID   string   `json:"-" bigquery:"owner"`
	Username  string   `json:"username" bigquery:"username"`
	MediaURL  string   `json:"media_url,omitempty" bigquery:"media_url"`
	MediaType string   `json:"media_type" bigquery:"media_type"`
	Timestamp MetaTime `json:"timestamp" bigquery:"timestamp"`

	Impressions int32 `json:"-" bigquery:"impressions"`
	Reach       int32 `json:"-" bigquery:"reach"`
	TapsForward int32 `json:"-" bigquery:"taps_forward"`
	TapsBack    int32 `json:"-" bigquery:"taps_back"`
	Exits       int32 `json:"-" bigquery:"exits"`
	Replies     int32 `json:"-" bigquery:"replies"` // Corresponde ao seu campo 'replies' no BQ

}
