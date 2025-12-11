package loaders

import (
	"context"
	"fmt"
	"log"
	"tmazleo/api-instagram/models"

	"cloud.google.com/go/bigquery"
)

// LoadProfileToBigQuery carrega um perfil para a tabela especificada no BigQuery.
func LoadProfileToBigQuery(ctx context.Context, projectID, datasetID, tableName string, data *models.InstagramProfile) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(datasetID).Table(tableName).Inserter()

	items := []interface{}{data}
	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção de Perfil na linha %d: %v", i, e)
			}
			return fmt.Errorf("erros parciais ou totais na inserção de Perfil")
		}
		return fmt.Errorf("erro fatal ao inserir Perfil no BigQuery: %w", err)
	}

	return nil
}

// LoadStoriesToBigQuery carrega uma lista de stories para a tabela especificada no BigQuery.
func LoadStoriesToBigQuery(ctx context.Context, projectID, datasetID, tableName string, data []*models.InstagramStory) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(datasetID).Table(tableName).Inserter()

	items := make([]interface{}, len(data))
	for i, s := range data {
		items[i] = s
	}

	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção de Story na linha %d: %v", i, e)
			}
			return fmt.Errorf("erros parciais ou totais na inserção de Stories")
		}
		return fmt.Errorf("erro fatal ao inserir Stories no BigQuery: %w", err)
	}

	return nil
}

// LoadMediaToBigQuery carrega mídias (posts) para a tabela especificada no BigQuery.
func LoadMediaToBigQuery(ctx context.Context, projectID, datasetID, tableName string, data []*models.InstagramMedia) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	defer client.Close()

	inserter := client.Dataset(datasetID).Table(tableName).Inserter()

	items := make([]interface{}, len(data))
	for i, m := range data {
		items[i] = m
	}

	if err := inserter.Put(ctx, items); err != nil {
		if multiErr, ok := err.(bigquery.MultiError); ok {
			for i, e := range multiErr {
				log.Printf("Erro de inserção na linha %d: %v", i, e)
			}
			return fmt.Errorf("erros parciais ou totais na inserção. Detalhes acima.")
		}
		return fmt.Errorf("erro fatal ao inserir dados no BigQuery: %w", err)
	}

	return nil
}
