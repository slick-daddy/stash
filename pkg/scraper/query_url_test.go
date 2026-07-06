package scraper

import (
	"reflect"
	"testing"

	"github.com/stashapp/stash/pkg/models"
)

func makeQueryURLScene(urls ...string) *models.Scene {
	return &models.Scene{
		Files: models.NewRelatedVideoFiles(nil),
		URLs:  models.NewRelatedStrings(urls),
	}
}

func TestConstructURLOrErrorReportsMissingFields(t *testing.T) {
	params := queryURLParameters{"title": "Example"}

	_, err := params.constructURLOrError("https://example.test/{title}/{url}/{checksum}/{url}")
	if err == nil {
		t.Fatal("expected missing fields error")
	}

	want := "missing fields for queryURL: checksum, url"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestQueryURLParameterCandidatesFromSceneFiltersMatchingURLs(t *testing.T) {
	scene := makeQueryURLScene("https://example.com/update/123", "https://studiox.com/update/456")
	definition := Definition{
		SceneByURL: []*ByURLDefinition{{URL: []string{"studiox.com/update"}}},
	}

	candidates, err := queryURLParameterCandidatesFromScene(scene, "{url}", ScrapeContentTypeScene, definition)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if got := candidates[0]["url"]; got != "https://studiox.com/update/456" {
		t.Fatalf("expected matching URL, got %q", got)
	}
}

func TestQueryURLParameterCandidatesFromSceneErrorsWhenNoURLsMatch(t *testing.T) {
	scene := makeQueryURLScene("https://example.com/update/123")
	definition := Definition{
		SceneByURL: []*ByURLDefinition{{URL: []string{"studiox.com/update"}}},
	}

	_, err := queryURLParameterCandidatesFromScene(scene, "{url}", ScrapeContentTypeScene, definition)
	if err == nil {
		t.Fatal("expected error")
	}

	want := "no scene URLs match this scraper"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestQueryURLParameterCandidatesFromSceneUsesAllURLsWithoutURLScrapers(t *testing.T) {
	scene := makeQueryURLScene("https://example.com/1", "https://example.com/2")

	candidates, err := queryURLParameterCandidatesFromScene(scene, "{url}", ScrapeContentTypeScene, Definition{})
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, candidate := range candidates {
		got = append(got, candidate["url"])
	}
	want := []string{"https://example.com/1", "https://example.com/2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
