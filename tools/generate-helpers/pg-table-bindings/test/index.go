// Code generated by pg-bindings generator. DO NOT EDIT.

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	mappings "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/test/options"
)

func init() {
	mapping.RegisterCategoryToTable(v1.SearchCategory_SEARCH_UNSET, table)
}

func NewIndexer(db *pgxpool.Pool) *indexerImpl {
	return &indexerImpl{
		db: db,
	}
}

type indexerImpl struct {
	db *pgxpool.Pool
}

func (b *indexerImpl) Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "TestSingleKeyStruct")

	return postgres.RunCountRequest(v1.SearchCategory_SEARCH_UNSET, q, b.db, mappings.OptionsMap)
}

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "TestSingleKeyStruct")

	return postgres.RunSearchRequest(v1.SearchCategory_SEARCH_UNSET, q, b.db, mappings.OptionsMap)
}

func (b *indexerImpl) SearchTestSingleKeyStructs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return nil, nil
}

func (b *indexerImpl) SearchRawTestSingleKeyStructs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return nil, nil
}

func (b *indexerImpl) SearchListTestSingleKeyStructs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return nil, nil
}

//// Stubs for satisfying interfaces

func (b *indexerImpl) AddTestSingleKeyStruct(deployment *storage.TestSingleKeyStruct) error {
	return nil
}

func (b *indexerImpl) AddTestSingleKeyStructs(_ []*storage.TestSingleKeyStruct) error {
	return nil
}

func (b *indexerImpl) DeleteTestSingleKeyStruct(id string) error {
	return nil
}

func (b *indexerImpl) DeleteTestSingleKeyStructs(_ []string) error {
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}
