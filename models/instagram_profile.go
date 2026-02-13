package models

import "time"

type InstagramProfile struct {
	AccountID           string    `json:"id" bigquery:"id"`
	IgID                string    `json:"ig_id" bigquery:"ig_id"` 
	Username            string    `json:"username" bigquery:"username"`
	FollowersCount      int32     `json:"followers_count" bigquery:"followers_count"`
	FollowsCount        int32     `json:"follows_count" bigquery:"follows_count"`
	MediaCount          int32     `json:"media_count" bigquery:"media_count"`
	ProfilePictureURL   string    `json:"profile_picture_url" bigquery:"profile_picture_url"`
	ExtractionTimestamp time.Time `json:"-" bigquery:"extraction_timestamp"`
	Date                time.Time `json:"-" bigquery:"date"`
}
