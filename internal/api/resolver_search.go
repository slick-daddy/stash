package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

// SearchPreview types are defined here rather than generated so that the
// search resolver can map directly from the store results. They are bound
// to the schema types of the same names via autobind.

type ScenePreview struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	ThumbnailPath *string `json:"thumbnailPath"`
	StudioName    *string `json:"studioName"`
	Duration      *int    `json:"duration"`
}

type PerformerPreview struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ImagePath *string `json:"imagePath"`
}

type TagPreview struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SceneCount *int   `json:"sceneCount"`
}

type StudioPreview struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ImagePath *string `json:"imagePath"`
}

type GalleryPreview struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	CoverPath *string `json:"coverPath"`
}

type GroupPreview struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ThumbnailPath *string `json:"thumbnailPath"`
}

type MarkerPreview struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	SceneID   string   `json:"sceneId"`
	SceneName *string  `json:"sceneName"`
	Seconds   *float64 `json:"seconds"`
}

type SearchResultCounts struct {
	Scenes     int `json:"scenes"`
	Performers int `json:"performers"`
	Tags       int `json:"tags"`
	Studios    int `json:"studios"`
	Galleries  int `json:"galleries"`
	Groups     int `json:"groups"`
	Markers    int `json:"markers"`
}

type SearchResults struct {
	Scenes      []*ScenePreview     `json:"scenes"`
	Performers  []*PerformerPreview `json:"performers"`
	Tags        []*TagPreview       `json:"tags"`
	Studios     []*StudioPreview    `json:"studios"`
	Galleries   []*GalleryPreview   `json:"galleries"`
	Groups      []*GroupPreview     `json:"groups"`
	Markers     []*MarkerPreview    `json:"markers"`
	TotalCounts *SearchResultCounts `json:"totalCounts"`
}

func (r *queryResolver) Search(ctx context.Context, input models.SearchInput) (*SearchResults, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)

	var ret *SearchResults
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		results, err := r.repository.Search.Search(ctx, input)
		if err != nil {
			return err
		}

		ret = makeSearchResults(baseURL, results)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func makeSearchResults(baseURL string, results *models.SearchResults) *SearchResults {
	previews := &SearchResults{
		Scenes:      make([]*ScenePreview, len(results.Scenes)),
		Performers:  make([]*PerformerPreview, len(results.Performers)),
		Tags:        make([]*TagPreview, len(results.Tags)),
		Studios:     make([]*StudioPreview, len(results.Studios)),
		Galleries:   make([]*GalleryPreview, len(results.Galleries)),
		Groups:      make([]*GroupPreview, len(results.Groups)),
		Markers:     make([]*MarkerPreview, len(results.Markers)),
		TotalCounts: makeSearchResultCounts(results.TotalCounts),
	}

	for i, r := range results.Scenes {
		thumbnailPath := previewPath(baseURL, "scene", r.ID, "screenshot")
		previews.Scenes[i] = &ScenePreview{
			ID:            strconv.Itoa(r.ID),
			Title:         r.Title,
			ThumbnailPath: &thumbnailPath,
			StudioName:    r.StudioName,
			Duration:      r.Duration,
		}
	}

	for i, r := range results.Performers {
		imagePath := previewPath(baseURL, "performer", r.ID, "image")
		previews.Performers[i] = &PerformerPreview{
			ID:        strconv.Itoa(r.ID),
			Name:      r.Name,
			ImagePath: &imagePath,
		}
	}

	for i, r := range results.Tags {
		sceneCount := r.SceneCount
		previews.Tags[i] = &TagPreview{
			ID:         strconv.Itoa(r.ID),
			Name:       r.Name,
			SceneCount: &sceneCount,
		}
	}

	for i, r := range results.Studios {
		imagePath := previewPath(baseURL, "studio", r.ID, "image")
		previews.Studios[i] = &StudioPreview{
			ID:        strconv.Itoa(r.ID),
			Name:      r.Name,
			ImagePath: &imagePath,
		}
	}

	for i, r := range results.Galleries {
		coverPath := previewPath(baseURL, "gallery", r.ID, "cover")
		previews.Galleries[i] = &GalleryPreview{
			ID:        strconv.Itoa(r.ID),
			Title:     r.Title,
			CoverPath: &coverPath,
		}
	}

	for i, r := range results.Groups {
		thumbnailPath := previewPath(baseURL, "group", r.ID, "frontimage")
		previews.Groups[i] = &GroupPreview{
			ID:            strconv.Itoa(r.ID),
			Name:          r.Name,
			ThumbnailPath: &thumbnailPath,
		}
	}

	for i, r := range results.Markers {
		previews.Markers[i] = &MarkerPreview{
			ID:        strconv.Itoa(r.ID),
			Title:     r.Title,
			SceneID:   strconv.Itoa(r.SceneID),
			SceneName: r.SceneName,
			Seconds:   &r.Seconds,
		}
	}

	return previews
}

// previewPath returns the URL for an entity's image endpoint. It returns
// the empty string if baseURL is empty.
func previewPath(baseURL string, pathPrefix string, id int, endpoint string) string {
	if baseURL == "" {
		return ""
	}

	return baseURL + "/" + pathPrefix + "/" + strconv.Itoa(id) + "/" + endpoint
}
func makeSearchResultCounts(counts models.SearchResultCounts) *SearchResultCounts {
	return &SearchResultCounts{
		Scenes:     counts.Scenes,
		Performers: counts.Performers,
		Tags:       counts.Tags,
		Studios:    counts.Studios,
		Galleries:  counts.Galleries,
		Groups:     counts.Groups,
		Markers:    counts.Markers,
	}
}
