//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stashapp/stash/pkg/models"
)

func TestSearchScenes(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		qb := db.Search

		// exact match (case-insensitive)
		title := getSceneStringValue(0, titleField)
		results, err := qb.Search(ctx, models.SearchInput{
			Term:         title,
			LimitPerType: 10,
		})
		if err != nil {
			t.Errorf("Error searching scenes: %s", err.Error())
			return nil
		}

		if !assert.Greater(t, len(results.Scenes), 0) {
			return nil
		}
		assert.Equal(t, sceneIDs[0], results.Scenes[0].ID)
		assert.Equal(t, title, results.Scenes[0].Title)
		assert.True(t, results.TotalCounts.Scenes >= len(results.Scenes))

		// duration is rounded from the primary video file duration
		expectedDuration := int(getSceneDuration(0) + 0.5)
		assert.Equal(t, &expectedDuration, results.Scenes[0].Duration)

		return nil
	})
}

func TestSearchPerformersAndOthers(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		qb := db.Search

		tests := []struct {
			name     string
			term     string
			typeName models.SearchEntityType
			ids      []int
		}{
			{"performers", "performer_0001_Name", models.SearchEntityPerformer, performerIDs},
			{"studios", "studio_0001_Name", models.SearchEntityStudio, studioIDs},
			{"groups", "group_0001_Name", models.SearchEntityGroup, groupIDs},
			{"galleries", "gallery_0001_Title", models.SearchEntityGallery, galleryIDs},
		}

		for _, tc := range tests {
			results, err := qb.Search(ctx, models.SearchInput{
				Term:         tc.term,
				LimitPerType: 10,
			})
			if err != nil {
				t.Errorf("Error searching %s: %s", tc.name, err.Error())
				continue
			}

			if !assert.Greater(t, len(results.Performers)+len(results.Studios)+len(results.Groups)+len(results.Galleries), 0, tc.name) {
				continue
			}

			var gotID int
			switch tc.typeName {
			case models.SearchEntityPerformer:
				gotID = results.Performers[0].ID
			case models.SearchEntityStudio:
				gotID = results.Studios[0].ID
			case models.SearchEntityGroup:
				gotID = results.Groups[0].ID
			case models.SearchEntityGallery:
				gotID = results.Galleries[0].ID
			}

			assert.Equal(t, tc.ids[1], gotID, tc.name)
		}

		return nil
	})
}

func TestSearchTagsSceneCount(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		qb := db.Search

		results, err := qb.Search(ctx, models.SearchInput{
			Term:         getTagStringValue(tagIdxWithScene, "Name"),
			LimitPerType: 10,
		})
		if err != nil {
			t.Errorf("Error searching tags: %s", err.Error())
			return nil
		}

		if !assert.Greater(t, len(results.Tags), 0) {
			return nil
		}
		assert.Equal(t, tagIDs[tagIdxWithScene], results.Tags[0].ID)
		assert.Equal(t, getTagSceneCount(tagIDs[tagIdxWithScene]), results.Tags[0].SceneCount)

		return nil
	})
}

func TestSearchLimitAndTypes(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		qb := db.Search

		// all seeded scene titles contain "_Title"
		results, err := qb.Search(ctx, models.SearchInput{
			Term:         titleField,
			LimitPerType: 3,
		})
		if err != nil {
			t.Errorf("Error searching scenes: %s", err.Error())
			return nil
		}

		assert.Equal(t, 3, len(results.Scenes))
		// all seeded scene titles contain "_Title" except the spaced-name scene
		assert.True(t, results.TotalCounts.Scenes >= totalScenes-1)

		// type filter restricts the query to tags only
		results, err = qb.Search(ctx, models.SearchInput{
			Term:         titleField,
			LimitPerType: 3,
			Types:        []models.SearchEntityType{models.SearchEntityTag},
		})
		if err != nil {
			t.Errorf("Error searching with types filter: %s", err.Error())
			return nil
		}

		assert.Equal(t, 0, len(results.Scenes))
		assert.Equal(t, 0, results.TotalCounts.Scenes)
		assert.Equal(t, 0, results.TotalCounts.Performers)

		return nil
	})
}

func TestSearchRanking(t *testing.T) {
	runWithRollbackTxn(t, "search ranking", func(t *testing.T, ctx context.Context) {
		qb := db.Performer

		names := []string{
			"zred",
			"ared",
			"Little Red Hood",
			"Red Fox",
			"Red",
		}

		for _, n := range names {
			performer := models.Performer{
				Name: n,
			}
			err := qb.Create(ctx, &models.CreatePerformerInput{
				Performer: &performer,
			})
			if err != nil {
				t.Fatalf("Error creating performer: %s", err.Error())
			}
		}

		results, err := db.Search.Search(ctx, models.SearchInput{
			Term:         "red",
			LimitPerType: 10,
		})
		if err != nil {
			t.Fatalf("Error searching performers: %s", err.Error())
		}

		// exact match first, then prefix matches, then substring matches.
		// alphabetical order as tiebreaker within each rank
		expected := []string{
			"Red",             // exact
			"Red Fox",         // prefix
			"ared",            // substring - alphabetical
			"Little Red Hood", // substring - alphabetical
			"zred",            // substring - alphabetical
		}

		var got []string
		for _, p := range results.Performers {
			if p.Name == "zred" || p.Name == "ared" || p.Name == "Little Red Hood" || p.Name == "Red Fox" || p.Name == "Red" {
				got = append(got, p.Name)
			}
		}

		assert.Equal(t, expected, got)
		assert.True(t, results.TotalCounts.Performers >= 5)
	})
}

func TestSearchWildcardRanking(t *testing.T) {
	runWithRollbackTxn(t, "search wildcard ranking", func(t *testing.T, ctx context.Context) {
		qb := db.Performer

		names := []string{
			// exact match
			"r_d",
			// true prefix - starts with the literal term
			"r_d x",
			// substring - matches the WHERE clause via the literal tail,
			// but must not rank as a prefix match. Starts with 'r' and
			// sorts before "r_d x" ('3' < '_' even under NOCASE), so a
			// prefix pattern that does not escape the term would promote
			// it to prefix rank and displace "r_d x" under limit 2
			"r3d r_d",
		}

		for _, n := range names {
			performer := models.Performer{
				Name: n,
			}
			err := qb.Create(ctx, &models.CreatePerformerInput{
				Performer: &performer,
			})
			if err != nil {
				t.Fatalf("Error creating performer: %s", err.Error())
			}
		}

		results, err := db.Search.Search(ctx, models.SearchInput{
			Term:         "r_d",
			LimitPerType: 2,
		})
		if err != nil {
			t.Fatalf("Error searching performers: %s", err.Error())
		}

		var got []string
		for _, p := range results.Performers {
			got = append(got, p.Name)
		}

		// exact then prefix; with limit 2 the genuine prefix match must
		// not be displaced by alphabetically earlier substring matches
		assert.Equal(t, []string{"r_d", "r_d x"}, got)
	})
}

func TestSearchMarkers(t *testing.T) {
	runWithRollbackTxn(t, "search markers", func(t *testing.T, ctx context.Context) {
		mqb := db.SceneMarker

		marker := models.SceneMarker{
			Title:        "findme marker",
			Seconds:      12.34,
			SceneID:      sceneIDs[sceneIdxWithMarkers],
			PrimaryTagID: tagIDs[tagIdxWithPrimaryMarkers],
		}
		if err := mqb.Create(ctx, &marker); err != nil {
			t.Fatalf("Error creating marker: %s", err.Error())
		}

		results, err := db.Search.Search(ctx, models.SearchInput{
			Term:         "findme",
			LimitPerType: 10,
		})
		if err != nil {
			t.Fatalf("Error searching markers: %s", err.Error())
		}

		if !assert.Len(t, results.Markers, 1) {
			return
		}

		found := results.Markers[0]
		assert.Equal(t, marker.ID, found.ID)
		assert.Equal(t, "findme marker", found.Title)
		assert.Equal(t, sceneIDs[sceneIdxWithMarkers], found.SceneID)
		assert.Equal(t, getSceneTitle(sceneIdxWithMarkers), *found.SceneName)
		assert.Equal(t, marker.Seconds, found.Seconds)
	})
}

func TestSearchEmptyTerm(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		qb := db.Search

		results, err := qb.Search(ctx, models.SearchInput{
			Term:         "   ",
			LimitPerType: 10,
		})
		if err != nil {
			t.Errorf("Error searching empty term: %s", err.Error())
			return nil
		}

		assert.Empty(t, results.Scenes)
		assert.Empty(t, results.Performers)
		assert.Empty(t, results.Tags)
		assert.Empty(t, results.Studios)
		assert.Empty(t, results.Galleries)
		assert.Empty(t, results.Groups)
		assert.Empty(t, results.Markers)
		assert.Equal(t, models.SearchResultCounts{}, results.TotalCounts)

		return nil
	})
}
