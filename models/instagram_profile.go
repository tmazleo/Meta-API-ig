package models

import "time"
type InstagramProfile struct {
    AccountID        string          `json:"id" bigquery:"account_id"`
    FollowersCount   int32           `json:"followers_count" bigquery:"followers_count"`
    MediaCount       int32           `json:"media_count" bigquery:"media_count"`
    Username         string          `json:"username" bigquery:"username"`
    ExtractionTimestamp time.Time    `json:"-" bigquery:"extraction_timestamp"`
    // Nota: O campo 'follows_count' (seguindo) não é fornecido pelo endpoint do Facebook Graph, então ele será omitido no struct.
}