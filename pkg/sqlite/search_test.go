//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// TestSearchCountsMatchListQuery verifies that unified search counts
// agree with the corresponding list page query for the same term. The
// "see all" link navigates from the header search to the list page, so
// the two must not disagree.
func TestSearchCountsMatchListQuery(t *testing.T) {
	runWithRollbackTxn(t, "search counts match list query", func(t *testing.T, ctx context.Context) {
		const term = "nata"
		aliasOnly := []string{"nata alias"}

		// performers: two matching by name, four by alias only
		for _, n := range []string{"nata lee", "zed natalie"} {
			err := db.Performer.Create(ctx, &models.CreatePerformerInput{
				Performer: &models.Performer{Name: n},
			})
			if err != nil {
				t.Fatalf("Error creating performer: %s", err.Error())
			}
		}
		for i := range 4 {
			err := db.Performer.Create(ctx, &models.CreatePerformerInput{
				Performer: &models.Performer{
					Name:    fmt.Sprintf("unrelated performer %d", i),
					Aliases: models.NewRelatedStrings(aliasOnly),
				},
			})
			if err != nil {
				t.Fatalf("Error creating performer: %s", err.Error())
			}
		}

		// studios: one matching by name, one by alias only
		for _, s := range []*models.Studio{
			{Name: "nata studio"},
			{Name: "unrelated studio", Aliases: models.NewRelatedStrings(aliasOnly)},
		} {
			err := db.Studio.Create(ctx, &models.CreateStudioInput{Studio: s})
			if err != nil {
				t.Fatalf("Error creating studio: %s", err.Error())
			}
		}

		// tags: one matching by name, one by alias only, one by sort name
		for _, tg := range []*models.Tag{
			{Name: "nata tag"},
			{Name: "unrelated tag", Aliases: models.NewRelatedStrings(aliasOnly)},
			{Name: "another unrelated tag", SortName: "nata sort"},
		} {
			err := db.Tag.Create(ctx, &models.CreateTagInput{Tag: tg})
			if err != nil {
				t.Fatalf("Error creating tag: %s", err.Error())
			}
		}

		// groups: aliases are a plain column; one name match, one alias
		for _, g := range []*models.Group{
			{Name: "nata group"},
			{Name: "unrelated group", Aliases: "nata alias"},
		} {
			err := db.Group.Create(ctx, g)
			if err != nil {
				t.Fatalf("Error creating group: %s", err.Error())
			}
		}

		results, err := db.Search.Search(ctx, models.SearchInput{
			Term:         term,
			LimitPerType: 10,
		})
		if err != nil {
			t.Fatalf("Error searching: %s", err.Error())
		}

		q := term
		findFilter := &models.FindFilterType{Q: &q}

		listCount := func(count int, err error) int {
			if err != nil {
				t.Fatalf("Error querying list count: %s", err.Error())
			}
			return count
		}

		assert.Equal(t,
			listCount(db.Performer.QueryCount(ctx, nil, findFilter)),
			results.TotalCounts.Performers,
			"performers search count must match the performers list page")

		assert.Equal(t,
			listCount(db.Studio.QueryCount(ctx, nil, findFilter)),
			results.TotalCounts.Studios,
			"studios search count must match the studios list page")

		_, tagsCount, err := db.Tag.Query(ctx, nil, findFilter)
		if err != nil {
			t.Fatalf("Error querying tags: %s", err.Error())
		}

		assert.Equal(t,
			tagsCount,
			results.TotalCounts.Tags,
			"tags search count must match the tags list page")

		assert.Equal(t,
			listCount(db.Group.QueryCount(ctx, nil, findFilter)),
			results.TotalCounts.Groups,
			"groups search count must match the groups list page")
	})
}

// TestSearchSceneCountsMatchListQuery verifies that the unified scene
// search matches the same sources as the scenes list page q filter:
// title, details, file path, fingerprints and marker titles.
func TestSearchSceneCountsMatchListQuery(t *testing.T) {
	runWithRollbackTxn(t, "search scene counts match list query", func(t *testing.T, ctx context.Context) {
		sqb := db.Scene

		// three scenes matching by title
		for i := range 3 {
			scene := &models.Scene{
				Title: fmt.Sprintf("elena scene %d", i),
			}
			err := sqb.Create(ctx, scene, nil)
			if err != nil {
				t.Fatalf("Error creating scene: %s", err.Error())
			}
		}

		// three scenes matching only via their details text
		for i := range 3 {
			scene := &models.Scene{
				Title:   fmt.Sprintf("unrelated scene %d", i),
				Details: "features elena",
			}
			err := sqb.Create(ctx, scene, nil)
			if err != nil {
				t.Fatalf("Error creating scene: %s", err.Error())
			}
		}

		// a scene matching only via a marker title
		tag := &models.Tag{Name: "elena marker tag"}
		err := db.Tag.Create(ctx, &models.CreateTagInput{Tag: tag})
		if err != nil {
			t.Fatalf("Error creating tag: %s", err.Error())
		}

		markerScene := &models.Scene{Title: "unrelated marker scene"}
		err = sqb.Create(ctx, markerScene, nil)
		if err != nil {
			t.Fatalf("Error creating scene: %s", err.Error())
		}

		marker := models.SceneMarker{
			Title:        "elena marker",
			Seconds:      1,
			SceneID:      markerScene.ID,
			PrimaryTagID: tag.ID,
		}
		if err := db.SceneMarker.Create(ctx, &marker); err != nil {
			t.Fatalf("Error creating marker: %s", err.Error())
		}

		// a scene matching only via its file path
		folder := &models.Folder{
			Path: "/elena folder",
			DirEntry: models.DirEntry{
				ModTime: time.Now(),
			},
		}
		if err := db.Folder.Create(ctx, folder); err != nil {
			t.Fatalf("Error creating folder: %s", err.Error())
		}

		pathFile := &models.VideoFile{
			BaseFile: &models.BaseFile{
				Basename:       "some other name.mp4",
				ParentFolderID: folder.ID,
				DirEntry: models.DirEntry{
					ModTime: time.Now(),
				},
				Fingerprints: []models.Fingerprint{
					{
						Type:        models.FingerprintTypeOshash,
						Fingerprint: "unrelated hash",
					},
				},
			},
		}
		if err := db.File.Create(ctx, pathFile); err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		pathScene := &models.Scene{Title: "unrelated path scene"}
		err = sqb.Create(ctx, pathScene, []models.FileID{pathFile.ID})
		if err != nil {
			t.Fatalf("Error creating scene: %s", err.Error())
		}

		// a scene matching only via a file fingerprint
		fpFolder := &models.Folder{
			Path: "/plain folder",
			DirEntry: models.DirEntry{
				ModTime: time.Now(),
			},
		}
		if err := db.Folder.Create(ctx, fpFolder); err != nil {
			t.Fatalf("Error creating folder: %s", err.Error())
		}

		fpFile := &models.VideoFile{
			BaseFile: &models.BaseFile{
				Basename:       "another video.mp4",
				ParentFolderID: fpFolder.ID,
				DirEntry: models.DirEntry{
					ModTime: time.Now(),
				},
				Fingerprints: []models.Fingerprint{
					{
						Type:        models.FingerprintTypeOshash,
						Fingerprint: "elena oshash",
					},
				},
			},
		}
		if err := db.File.Create(ctx, fpFile); err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		fpScene := &models.Scene{Title: "unrelated fingerprint scene"}
		err = sqb.Create(ctx, fpScene, []models.FileID{fpFile.ID})
		if err != nil {
			t.Fatalf("Error creating scene: %s", err.Error())
		}

		const term = "elena"

		results, err := db.Search.Search(ctx, models.SearchInput{
			Term:         term,
			LimitPerType: 10,
		})
		if err != nil {
			t.Fatalf("Error searching scenes: %s", err.Error())
		}

		q := term
		listCount, err := sqb.QueryCount(ctx, nil, &models.FindFilterType{Q: &q})
		if err != nil {
			t.Fatalf("Error querying scenes: %s", err.Error())
		}

		// nine scenes: three by title, three via details, and one each
		// via marker title, file path and fingerprint
		assert.Equal(t, 9, listCount)
		assert.Equal(t,
			listCount,
			results.TotalCounts.Scenes,
			"scenes search count must match the scenes list page count for the same term")
		assert.Len(t, results.Scenes, 9)
	})
}
