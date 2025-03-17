package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"time"
)

var ctx = context.Background()

type Test struct {
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

func Insert() error {
	indexName := "products" + time.Now().Format("20060102")
	req := resource.ElasticSearchClient.Bulk().Index(indexName)
	doc := elastic.NewBulkCreateRequest().Doc(map[string]interface{}{
		"title":      "Cross body Bag with Tassel",
		"created_at": time.Now(),
	})
	req.Add(doc)

	_, err := req.Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func Search() error {
	indexName := "products*"
	res, err := resource.ElasticSearchClient.Search().
		Index(indexName).
		//降序
		Sort("created_at", false).
		Size(1).
		Do(ctx)
	if err != nil {
		return err
	}

	if res.Hits.TotalHits.Value == 0 {
		return nil
	}

	for _, hit := range res.Hits.Hits {
		var test Test
		if err = json.Unmarshal(hit.Source, &test); err != nil {
			return err
		}
		fmt.Println(test)
	}

	return nil
}
