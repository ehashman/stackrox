// Code generated by blevebindings generator. DO NOT EDIT.

package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

var OptionsMap = blevesearch.Walk(v1.SearchCategory_NODES, "node", (*storage.Node)(nil))
