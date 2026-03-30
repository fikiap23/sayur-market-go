package config

import (
	"log"

	"github.com/elastic/go-elasticsearch/v7"
)

func (cfg Config) InitElasticsearch() (*elasticsearch.Client, error) {
	configElastic := elasticsearch.Config{
		Addresses: []string{cfg.ElasticSearch.Host},
	}
	es, err := elasticsearch.NewClient(configElastic)
	if err != nil {
		log.Fatalf("[InitElasticsearch-1] Error initializing Elasticsearch: %s", err)
		return nil, err
	}

	return es, nil
}
