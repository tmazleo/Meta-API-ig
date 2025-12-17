package models

type MediaResponse struct {
	Data []InstagramMedia `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}