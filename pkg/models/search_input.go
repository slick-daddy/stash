package models

import (
	"context"
	"fmt"
	"io"
	"strconv"
)

type SearchEntityType string

const (
	SearchEntityScene     SearchEntityType = "SCENE"
	SearchEntityPerformer SearchEntityType = "PERFORMER"
	SearchEntityTag       SearchEntityType = "TAG"
	SearchEntityStudio    SearchEntityType = "STUDIO"
	SearchEntityGallery   SearchEntityType = "GALLERY"
	SearchEntityGroup     SearchEntityType = "GROUP"
	SearchEntityMarker    SearchEntityType = "MARKER"
)

var AllSearchEntityType = []SearchEntityType{
	SearchEntityScene,
	SearchEntityPerformer,
	SearchEntityTag,
	SearchEntityStudio,
	SearchEntityGallery,
	SearchEntityGroup,
	SearchEntityMarker,
}

func (e SearchEntityType) IsValid() bool {
	switch e {
	case SearchEntityScene, SearchEntityPerformer, SearchEntityTag,
		SearchEntityStudio, SearchEntityGallery, SearchEntityGroup, SearchEntityMarker:
		return true
	}
	return false
}

func (e SearchEntityType) String() string {
	return string(e)
}

func (e *SearchEntityType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SearchEntityType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SearchEntityType", str)
	}
	return nil
}

func (e SearchEntityType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type SearchInput struct {
	Term         string             `json:"term"`
	LimitPerType int                `json:"limit_per_type"`
	Types        []SearchEntityType `json:"types"`
}

// GetLimitPerType returns the limit per entity type, defaulting to
// 10 if unset or negative. Matches the schema default for limitPerType.
func (i SearchInput) GetLimitPerType() int {
	const defaultLimit = 10

	if i.LimitPerType <= 0 {
		return defaultLimit
	}

	return i.LimitPerType
}

type SceneSearchResult struct {
	ID         int
	Title      string
	StudioName *string
	Duration   *int
}

type PerformerSearchResult struct {
	ID   int
	Name string
}

type TagSearchResult struct {
	ID         int
	Name       string
	SceneCount int
}

type StudioSearchResult struct {
	ID   int
	Name string
}

type GallerySearchResult struct {
	ID    int
	Title string
}

type GroupSearchResult struct {
	ID   int
	Name string
}

type MarkerSearchResult struct {
	ID        int
	Title     string
	SceneID   int
	SceneName *string
	Seconds   float64
}

type SearchResultCounts struct {
	Scenes     int
	Performers int
	Tags       int
	Studios    int
	Galleries  int
	Groups     int
	Markers    int
}

type SearchResults struct {
	Scenes      []*SceneSearchResult
	Performers  []*PerformerSearchResult
	Tags        []*TagSearchResult
	Studios     []*StudioSearchResult
	Galleries   []*GallerySearchResult
	Groups      []*GroupSearchResult
	Markers     []*MarkerSearchResult
	TotalCounts SearchResultCounts
}

// Searcher provides unified search across all entity types.
type Searcher interface {
	Search(ctx context.Context, input SearchInput) (*SearchResults, error)
}
