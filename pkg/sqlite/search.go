package sqlite

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/guregu/null.v4"

	"github.com/stashapp/stash/pkg/models"
)

type searchSceneRow struct {
	ID         int         `db:"id"`
	Title      null.String `db:"title"`
	StudioName null.String `db:"studio_name"`
	Duration   null.Int    `db:"duration"`
}

type searchNameRow struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type searchTagRow struct {
	ID         int    `db:"id"`
	Name       string `db:"name"`
	SceneCount int    `db:"scene_count"`
}

type searchGalleryRow struct {
	ID    int         `db:"id"`
	Title null.String `db:"title"`
}

type searchMarkerRow struct {
	ID        int         `db:"id"`
	Title     null.String `db:"title"`
	Seconds   float64     `db:"seconds"`
	SceneID   int         `db:"scene_id"`
	SceneName null.String `db:"scene_name"`
}

type SearchStore struct{}

func NewSearchStore() *SearchStore {
	return &SearchStore{}
}

// escapeLike escapes LIKE wildcard characters so that they are matched
// literally within a LIKE ... ESCAPE '\' pattern.
func escapeLike(v string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(v)
}

// likeTerm escapes the term for literal use within a LIKE clause.
// Wildcard characters in the term are escaped so they are matched
// literally.
func likeTerm(term string) string {
	return "%" + escapeLike(term) + "%"
}

// searchRankExpr returns a CASE expression ranking exact matches first,
// then prefix matches, then substring matches. Takes the lower-cased
// term and its prefix as arguments.
func searchRankExpr(column string) string {
	return fmt.Sprintf(
		`CASE WHEN LOWER(%[1]s) = LOWER(?) THEN 0 WHEN LOWER(%[1]s) LIKE ? ESCAPE '\' THEN 1 ELSE 2 END`,
		column,
	)
}

// requestedTypeSet returns a set of entity types to search for. An empty
// input list means all types.
func requestedTypeSet(types []models.SearchEntityType) map[models.SearchEntityType]bool {
	if len(types) == 0 {
		set := make(map[models.SearchEntityType]bool, len(models.AllSearchEntityType))
		for _, t := range models.AllSearchEntityType {
			set[t] = true
		}
		return set
	}

	set := make(map[models.SearchEntityType]bool, len(types))
	for _, t := range types {
		if t.IsValid() {
			set[t] = true
		}
	}
	return set
}

func (qb *SearchStore) Search(ctx context.Context, input models.SearchInput) (*models.SearchResults, error) {
	term := strings.TrimSpace(input.Term)
	ret := &models.SearchResults{}

	if term == "" {
		return ret, nil
	}

	limit := input.GetLimitPerType()
	requested := requestedTypeSet(input.Types)

	var err error

	if requested[models.SearchEntityScene] {
		ret.Scenes, ret.TotalCounts.Scenes, err = qb.searchScenes(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching scenes: %w", err)
		}
	}

	if requested[models.SearchEntityPerformer] {
		ret.Performers, ret.TotalCounts.Performers, err = qb.searchPerformers(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching performers: %w", err)
		}
	}

	if requested[models.SearchEntityTag] {
		ret.Tags, ret.TotalCounts.Tags, err = qb.searchTags(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching tags: %w", err)
		}
	}

	if requested[models.SearchEntityStudio] {
		ret.Studios, ret.TotalCounts.Studios, err = qb.searchStudios(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching studios: %w", err)
		}
	}

	if requested[models.SearchEntityGallery] {
		ret.Galleries, ret.TotalCounts.Galleries, err = qb.searchGalleries(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching galleries: %w", err)
		}
	}

	if requested[models.SearchEntityGroup] {
		ret.Groups, ret.TotalCounts.Groups, err = qb.searchGroups(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching groups: %w", err)
		}
	}

	if requested[models.SearchEntityMarker] {
		ret.Markers, ret.TotalCounts.Markers, err = qb.searchMarkers(ctx, term, limit)
		if err != nil {
			return nil, fmt.Errorf("searching markers: %w", err)
		}
	}

	return ret, nil
}

// rankArgs returns the arguments for the rank expression in a query's
// ORDER BY clause. The prefix pattern escapes the term before appending
// the ranking wildcard so wildcard characters in the term are matched
// literally.
func rankArgs(term string) []interface{} {
	lower := strings.ToLower(term)
	return []interface{}{lower, escapeLike(lower) + "%"}
}

func (qb *SearchStore) searchScenes(ctx context.Context, term string, limit int) ([]*models.SceneSearchResult, int, error) {
	const whereClause = `WHERE COALESCE(scenes.title, '') LIKE ? ESCAPE '\'`

	countQuery := `SELECT COUNT(*) FROM scenes ` + whereClause

	var count int
	if err := dbWrapper.Get(ctx, &count, countQuery, likeTerm(term)); err != nil {
		return nil, 0, err
	}

	query := `
SELECT scenes.id AS id, scenes.title AS title, studios.name AS studio_name,
	CASE WHEN video_files.duration IS NULL THEN NULL ELSE CAST(ROUND(video_files.duration) AS INTEGER) END AS duration
FROM scenes
LEFT JOIN studios ON studios.id = scenes.studio_id
LEFT JOIN scenes_files ON scenes_files.scene_id = scenes.id AND scenes_files."primary" = 1
LEFT JOIN video_files ON video_files.file_id = scenes_files.file_id
` + whereClause + `
ORDER BY ` + searchRankExpr(`COALESCE(scenes.title, '')`) + `, COALESCE(scenes.title, '') COLLATE NOCASE ASC, scenes.id ASC
LIMIT ?`

	args := []interface{}{likeTerm(term)}
	args = append(args, rankArgs(term)...)
	args = append(args, limit)

	var rows []*searchSceneRow
	if err := dbWrapper.Select(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}

	ret := make([]*models.SceneSearchResult, len(rows))
	for i, r := range rows {
		var duration *int
		if r.Duration.Valid {
			v := int(r.Duration.Int64)
			duration = &v
		}

		ret[i] = &models.SceneSearchResult{
			ID:         r.ID,
			Title:      r.Title.String,
			StudioName: r.StudioName.Ptr(),
			Duration:   duration,
		}
	}

	return ret, count, nil
}

// namedSearchOptions configures additional term matching for
// searchByName. The extra columns and alias tables mirror what the
// corresponding list page's q filter matches, so that the "see all"
// count shown in the header search agrees with the list it opens.
type namedSearchOptions struct {
	// extraColumns are additional columns on the entity table that are
	// matched against the term.
	extraColumns []string

	// aliasTable, when non-empty, adds an EXISTS clause matching the
	// table's alias column against the term, joined on aliasFK equal to
	// the entity table's id.
	aliasTable string
	aliasFK    string
}

// nameMatchClauses returns the WHERE conditions matching the term
// against the given name column plus anything configured in opts.
func nameMatchClauses(table string, nameColumn string, opts namedSearchOptions) []string {
	clauses := []string{nameColumn + ` LIKE ? ESCAPE '\'`}
	for _, c := range opts.extraColumns {
		clauses = append(clauses, c+` LIKE ? ESCAPE '\'`)
	}
	if opts.aliasTable != "" {
		clauses = append(clauses, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM %[1]s WHERE %[1]s.%[2]s = %[3]s.id AND %[1]s.alias LIKE ? ESCAPE '\')`,
			opts.aliasTable, opts.aliasFK, table,
		))
	}
	return clauses
}

func (qb *SearchStore) searchPerformers(ctx context.Context, term string, limit int) ([]*models.PerformerSearchResult, int, error) {
	rows, count, err := searchByName(ctx, "performers", "performers.name", term, limit, namedSearchOptions{
		aliasTable: "performer_aliases",
		aliasFK:    "performer_id",
	})
	if err != nil {
		return nil, 0, err
	}

	ret := make([]*models.PerformerSearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.PerformerSearchResult{
			ID:   r.ID,
			Name: r.Name,
		}
	}

	return ret, count, nil
}

func (qb *SearchStore) searchStudios(ctx context.Context, term string, limit int) ([]*models.StudioSearchResult, int, error) {
	rows, count, err := searchByName(ctx, "studios", "studios.name", term, limit, namedSearchOptions{
		aliasTable: "studio_aliases",
		aliasFK:    "studio_id",
	})
	if err != nil {
		return nil, 0, err
	}

	ret := make([]*models.StudioSearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.StudioSearchResult{
			ID:   r.ID,
			Name: r.Name,
		}
	}

	return ret, count, nil
}

func (qb *SearchStore) searchGroups(ctx context.Context, term string, limit int) ([]*models.GroupSearchResult, int, error) {
	rows, count, err := searchByName(ctx, "groups", "groups.name", term, limit, namedSearchOptions{
		extraColumns: []string{"groups.aliases"},
	})
	if err != nil {
		return nil, 0, err
	}

	ret := make([]*models.GroupSearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.GroupSearchResult{
			ID:   r.ID,
			Name: r.Name,
		}
	}

	return ret, count, nil
}

// searchByName performs a ranked name search on the given table. The
// table must have id and name columns. Matching beyond the name column
// is configured via opts, mirroring the list page's q filter so that
// the count agrees with the list the "see all" action opens. Rows are
// ranked on the name column only: alias matches rank after exact and
// prefix name matches.
func searchByName(ctx context.Context, table string, nameColumn string, term string, limit int, opts namedSearchOptions) ([]*searchNameRow, int, error) {
	clauses := nameMatchClauses(table, nameColumn, opts)
	whereClause := ""
	var matchArgs []interface{}
	if len(clauses) > 0 {
		whereClause = "WHERE " + strings.Join(clauses, " OR ")
		for range clauses {
			matchArgs = append(matchArgs, likeTerm(term))
		}
	}

	countQuery := `SELECT COUNT(*) FROM ` + table + ` ` + whereClause

	var count int
	if err := dbWrapper.Get(ctx, &count, countQuery, matchArgs...); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, ` + nameColumn + ` AS name FROM ` + table + ` ` + whereClause +
		` ORDER BY ` + searchRankExpr(nameColumn) + `, ` + nameColumn + ` COLLATE NOCASE ASC, id ASC LIMIT ?`

	args := append(append([]interface{}{}, matchArgs...), rankArgs(term)...)
	args = append(args, limit)

	var rows []*searchNameRow
	if err := dbWrapper.Select(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}

	return rows, count, nil
}

func (qb *SearchStore) searchTags(ctx context.Context, term string, limit int) ([]*models.TagSearchResult, int, error) {
	// mirrors the tags list page q filter: name, aliases and sort name
	whereClause := `WHERE tags.name LIKE ? ESCAPE '\'
		OR tags.sort_name LIKE ? ESCAPE '\'
		OR EXISTS (SELECT 1 FROM tag_aliases WHERE tag_aliases.tag_id = tags.id AND tag_aliases.alias LIKE ? ESCAPE '\')`

	countQuery := `SELECT COUNT(*) FROM tags ` + whereClause

	matchArgs := []interface{}{likeTerm(term), likeTerm(term), likeTerm(term)}

	var count int
	if err := dbWrapper.Get(ctx, &count, countQuery, matchArgs...); err != nil {
		return nil, 0, err
	}

	query := `
SELECT tags.id AS id, tags.name AS name,
	(SELECT COUNT(*) FROM scenes_tags WHERE scenes_tags.tag_id = tags.id) AS scene_count
FROM tags
` + whereClause + `
ORDER BY ` + searchRankExpr(`tags.name`) + `, tags.name COLLATE NOCASE ASC, tags.id ASC
LIMIT ?`

	args := append(append([]interface{}{}, matchArgs...), rankArgs(term)...)
	args = append(args, limit)

	var rows []*searchTagRow
	if err := dbWrapper.Select(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}

	ret := make([]*models.TagSearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.TagSearchResult{
			ID:         r.ID,
			Name:       r.Name,
			SceneCount: r.SceneCount,
		}
	}

	return ret, count, nil
}

func (qb *SearchStore) searchGalleries(ctx context.Context, term string, limit int) ([]*models.GallerySearchResult, int, error) {
	const whereClause = `WHERE COALESCE(galleries.title, '') LIKE ? ESCAPE '\'`

	countQuery := `SELECT COUNT(*) FROM galleries ` + whereClause

	var count int
	if err := dbWrapper.Get(ctx, &count, countQuery, likeTerm(term)); err != nil {
		return nil, 0, err
	}

	query := `
SELECT galleries.id AS id, galleries.title AS title
FROM galleries
` + whereClause + `
ORDER BY ` + searchRankExpr(`COALESCE(galleries.title, '')`) + `, COALESCE(galleries.title, '') COLLATE NOCASE ASC, galleries.id ASC
LIMIT ?`

	args := []interface{}{likeTerm(term)}
	args = append(args, rankArgs(term)...)
	args = append(args, limit)

	var rows []*searchGalleryRow
	if err := dbWrapper.Select(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}

	ret := make([]*models.GallerySearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.GallerySearchResult{
			ID:    r.ID,
			Title: r.Title.String,
		}
	}

	return ret, count, nil
}

func (qb *SearchStore) searchMarkers(ctx context.Context, term string, limit int) ([]*models.MarkerSearchResult, int, error) {
	const whereClause = `WHERE scene_markers.title LIKE ? ESCAPE '\'`

	countQuery := `SELECT COUNT(*) FROM scene_markers ` + whereClause

	var count int
	if err := dbWrapper.Get(ctx, &count, countQuery, likeTerm(term)); err != nil {
		return nil, 0, err
	}

	query := `
SELECT scene_markers.id AS id, scene_markers.title AS title, scene_markers.seconds AS seconds,
	scene_markers.scene_id AS scene_id, scenes.title AS scene_name
FROM scene_markers
LEFT JOIN scenes ON scenes.id = scene_markers.scene_id
` + whereClause + `
ORDER BY ` + searchRankExpr(`scene_markers.title`) + `, scene_markers.title COLLATE NOCASE ASC, scene_markers.id ASC
LIMIT ?`

	args := []interface{}{likeTerm(term)}
	args = append(args, rankArgs(term)...)
	args = append(args, limit)

	var rows []*searchMarkerRow
	if err := dbWrapper.Select(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}

	ret := make([]*models.MarkerSearchResult, len(rows))
	for i, r := range rows {
		ret[i] = &models.MarkerSearchResult{
			ID:        r.ID,
			Title:     r.Title.String,
			SceneID:   r.SceneID,
			SceneName: r.SceneName.Ptr(),
			Seconds:   r.Seconds,
		}
	}

	return ret, count, nil
}
