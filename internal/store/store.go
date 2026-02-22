package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/robertmeta/twi-map/internal/model"
)

// Store manages all data persistence via DuckDB.
type Store struct {
	DB      *sql.DB
	DataDir string
}

// New opens (or creates) a DuckDB database in the given data directory.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "twi-map.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}

	s := &Store{DB: db, DataDir: dataDir}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS chapters (
			idx INTEGER PRIMARY KEY,
			web_title TEXT NOT NULL,
			url TEXT NOT NULL,
			volume TEXT NOT NULL,
			book_number INTEGER,
			audiobook_chapter TEXT,
			ebook_chapter TEXT,
			slug TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chapter_text (
			chapter_idx INTEGER PRIMARY KEY REFERENCES chapters(idx),
			body TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS extraction_meta (
			chapter_idx INTEGER PRIMARY KEY REFERENCES chapters(idx),
			model TEXT NOT NULL,
			extracted_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS extracted_locations (
			id INTEGER PRIMARY KEY DEFAULT nextval('extracted_locations_seq'),
			chapter_idx INTEGER NOT NULL REFERENCES chapters(idx),
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			aliases TEXT,
			description TEXT,
			visual_description TEXT,
			context_quotes TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS extracted_relationships (
			id INTEGER PRIMARY KEY DEFAULT nextval('extracted_relationships_seq'),
			chapter_idx INTEGER NOT NULL REFERENCES chapters(idx),
			from_loc TEXT NOT NULL,
			to_loc TEXT NOT NULL,
			type TEXT NOT NULL,
			detail TEXT,
			quote TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS extracted_containment (
			id INTEGER PRIMARY KEY DEFAULT nextval('extracted_containment_seq'),
			chapter_idx INTEGER NOT NULL REFERENCES chapters(idx),
			child TEXT NOT NULL,
			parent TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS locations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			aliases TEXT,
			description TEXT,
			visual_description TEXT,
			first_chapter_idx INTEGER NOT NULL,
			mention_count INTEGER NOT NULL,
			chapter_indices TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS relationships (
			id INTEGER PRIMARY KEY DEFAULT nextval('relationships_seq'),
			from_loc TEXT NOT NULL,
			to_loc TEXT NOT NULL,
			type TEXT NOT NULL,
			detail TEXT,
			first_chapter_idx INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS containment (
			id INTEGER PRIMARY KEY DEFAULT nextval('containment_seq'),
			child TEXT NOT NULL,
			parent TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS coordinates (
			location_id TEXT PRIMARY KEY,
			x DOUBLE NOT NULL,
			y DOUBLE NOT NULL,
			confidence TEXT NOT NULL DEFAULT 'estimated',
			manual BOOLEAN NOT NULL DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	// Create sequences first (ignore errors if they already exist)
	seqs := []string{
		"CREATE SEQUENCE IF NOT EXISTS extracted_locations_seq",
		"CREATE SEQUENCE IF NOT EXISTS extracted_relationships_seq",
		"CREATE SEQUENCE IF NOT EXISTS extracted_containment_seq",
		"CREATE SEQUENCE IF NOT EXISTS relationships_seq",
		"CREATE SEQUENCE IF NOT EXISTS containment_seq",
	}
	for _, seq := range seqs {
		if _, err := s.DB.Exec(seq); err != nil {
			return fmt.Errorf("creating sequence: %w", err)
		}
	}

	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return fmt.Errorf("executing migration %q: %w", stmt[:60], err)
		}
	}

	// Add columns to existing tables (ignore errors if they already exist)
	alters := []string{
		"ALTER TABLE extracted_locations ADD COLUMN visual_description TEXT",
		"ALTER TABLE locations ADD COLUMN visual_description TEXT",
	}
	for _, alt := range alters {
		s.DB.Exec(alt) // ignore errors (column already exists)
	}

	return nil
}

// WriteTOC inserts or replaces all chapter metadata.
func (s *Store) WriteTOC(toc *model.TOC) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM chapters"); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO chapters (idx, web_title, url, volume, book_number, audiobook_chapter, ebook_chapter, slug)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ch := range toc.Chapters {
		if _, err := stmt.Exec(ch.Index, ch.WebTitle, ch.URL, ch.Volume, ch.BookNumber, ch.AudiobookChapter, ch.EbookChapter, ch.Slug); err != nil {
			return fmt.Errorf("inserting chapter %d: %w", ch.Index, err)
		}
	}

	if _, err := tx.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES ('toc_scraped_at', ?)", toc.ScrapedAt); err != nil {
		return err
	}

	return tx.Commit()
}

// ReadTOC loads all chapter metadata.
func (s *Store) ReadTOC() (*model.TOC, error) {
	rows, err := s.DB.Query("SELECT idx, web_title, url, volume, book_number, audiobook_chapter, ebook_chapter, slug FROM chapters ORDER BY idx")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var toc model.TOC
	for rows.Next() {
		var ch model.Chapter
		if err := rows.Scan(&ch.Index, &ch.WebTitle, &ch.URL, &ch.Volume, &ch.BookNumber, &ch.AudiobookChapter, &ch.EbookChapter, &ch.Slug); err != nil {
			return nil, err
		}
		toc.Chapters = append(toc.Chapters, ch)
	}

	var scrapedAt sql.NullString
	s.DB.QueryRow("SELECT value FROM meta WHERE key = 'toc_scraped_at'").Scan(&scrapedAt)
	toc.ScrapedAt = scrapedAt.String

	return &toc, rows.Err()
}

// WriteChapterText stores a chapter's plaintext body.
func (s *Store) WriteChapterText(chapterIdx int, text string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO chapter_text (chapter_idx, body) VALUES (?, ?)", chapterIdx, text)
	return err
}

// ReadChapterText retrieves a chapter's plaintext body.
func (s *Store) ReadChapterText(chapterIdx int) (string, error) {
	var body string
	err := s.DB.QueryRow("SELECT body FROM chapter_text WHERE chapter_idx = ?", chapterIdx).Scan(&body)
	return body, err
}

// ChapterTextExists checks if a chapter's text has been scraped.
func (s *Store) ChapterTextExists(chapterIdx int) bool {
	var n int
	s.DB.QueryRow("SELECT 1 FROM chapter_text WHERE chapter_idx = ?", chapterIdx).Scan(&n)
	return n == 1
}

// WriteExtraction saves a chapter's extraction results.
func (s *Store) WriteExtraction(ext *model.ChapterExtraction) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear any previous extraction for this chapter
	for _, tbl := range []string{"extracted_locations", "extracted_relationships", "extracted_containment", "extraction_meta"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE chapter_idx = ?", tbl), ext.ChapterIndex); err != nil {
			return err
		}
	}

	// Insert meta
	if _, err := tx.Exec("INSERT INTO extraction_meta (chapter_idx, model, extracted_at) VALUES (?, ?, ?)",
		ext.ChapterIndex, ext.Model, ext.ExtractedAt); err != nil {
		return err
	}

	// Insert locations
	for _, loc := range ext.Locations {
		aliases, _ := json.Marshal(loc.Aliases)
		quotes, _ := json.Marshal(loc.ContextQuotes)
		if _, err := tx.Exec("INSERT INTO extracted_locations (chapter_idx, name, type, aliases, description, visual_description, context_quotes) VALUES (?, ?, ?, ?, ?, ?, ?)",
			ext.ChapterIndex, loc.Name, loc.Type, string(aliases), loc.Description, loc.VisualDescription, string(quotes)); err != nil {
			return err
		}
	}

	// Insert relationships
	for _, rel := range ext.Relationships {
		if _, err := tx.Exec("INSERT INTO extracted_relationships (chapter_idx, from_loc, to_loc, type, detail, quote) VALUES (?, ?, ?, ?, ?, ?)",
			ext.ChapterIndex, rel.From, rel.To, rel.Type, rel.Detail, rel.Quote); err != nil {
			return err
		}
	}

	// Insert containment
	for _, c := range ext.Containment {
		if _, err := tx.Exec("INSERT INTO extracted_containment (chapter_idx, child, parent) VALUES (?, ?, ?)",
			ext.ChapterIndex, c.Child, c.Parent); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ReadExtraction loads a chapter's extraction results.
func (s *Store) ReadExtraction(chapterIdx int) (*model.ChapterExtraction, error) {
	ext := &model.ChapterExtraction{ChapterIndex: chapterIdx}

	// Meta
	err := s.DB.QueryRow("SELECT model, extracted_at FROM extraction_meta WHERE chapter_idx = ?", chapterIdx).
		Scan(&ext.Model, &ext.ExtractedAt)
	if err != nil {
		return nil, err
	}

	// Chapter title
	s.DB.QueryRow("SELECT web_title FROM chapters WHERE idx = ?", chapterIdx).Scan(&ext.ChapterTitle)

	// Locations
	rows, err := s.DB.Query("SELECT name, type, aliases, description, visual_description, context_quotes FROM extracted_locations WHERE chapter_idx = ?", chapterIdx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var loc model.ExtractedLocation
		var aliases, quotes, visualDesc sql.NullString
		if err := rows.Scan(&loc.Name, &loc.Type, &aliases, &loc.Description, &visualDesc, &quotes); err != nil {
			return nil, err
		}
		if aliases.Valid {
			json.Unmarshal([]byte(aliases.String), &loc.Aliases)
		}
		if visualDesc.Valid {
			loc.VisualDescription = visualDesc.String
		}
		if quotes.Valid {
			json.Unmarshal([]byte(quotes.String), &loc.ContextQuotes)
		}
		ext.Locations = append(ext.Locations, loc)
	}

	// Relationships
	relRows, err := s.DB.Query("SELECT from_loc, to_loc, type, detail, quote FROM extracted_relationships WHERE chapter_idx = ?", chapterIdx)
	if err != nil {
		return nil, err
	}
	defer relRows.Close()
	for relRows.Next() {
		var rel model.ExtractedRelationship
		if err := relRows.Scan(&rel.From, &rel.To, &rel.Type, &rel.Detail, &rel.Quote); err != nil {
			return nil, err
		}
		ext.Relationships = append(ext.Relationships, rel)
	}

	// Containment
	cRows, err := s.DB.Query("SELECT child, parent FROM extracted_containment WHERE chapter_idx = ?", chapterIdx)
	if err != nil {
		return nil, err
	}
	defer cRows.Close()
	for cRows.Next() {
		var c model.Containment
		if err := cRows.Scan(&c.Child, &c.Parent); err != nil {
			return nil, err
		}
		ext.Containment = append(ext.Containment, c)
	}

	return ext, nil
}

// ExtractionExists checks if a chapter has been extracted.
func (s *Store) ExtractionExists(chapterIdx int) bool {
	var n int
	s.DB.QueryRow("SELECT 1 FROM extraction_meta WHERE chapter_idx = ?", chapterIdx).Scan(&n)
	return n == 1
}

// WriteAggregated saves the aggregated location data.
func (s *Store) WriteAggregated(data *model.AggregatedData) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear previous aggregation
	for _, tbl := range []string{"locations", "relationships", "containment"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tbl)); err != nil {
			return fmt.Errorf("clearing %s: %w", tbl, err)
		}
	}

	// Dedup locations by ID before inserting
	seenLoc := make(map[string]bool)
	for _, loc := range data.Locations {
		if seenLoc[loc.ID] {
			continue
		}
		seenLoc[loc.ID] = true
		aliases, _ := json.Marshal(loc.Aliases)
		indices, _ := json.Marshal(loc.ChapterIndices)
		if _, err := tx.Exec("INSERT INTO locations (id, name, type, aliases, description, visual_description, first_chapter_idx, mention_count, chapter_indices) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO NOTHING",
			loc.ID, loc.Name, loc.Type, string(aliases), loc.Description, loc.VisualDescription, loc.FirstChapterIndex, loc.MentionCount, string(indices)); err != nil {
			return fmt.Errorf("inserting location %s: %w", loc.ID, err)
		}
	}

	for _, rel := range data.Relationships {
		if _, err := tx.Exec("INSERT INTO relationships (from_loc, to_loc, type, detail, first_chapter_idx) VALUES (?, ?, ?, ?, ?)",
			rel.From, rel.To, rel.Type, rel.Detail, rel.FirstChapterIndex); err != nil {
			return err
		}
	}

	for _, c := range data.Containment {
		if _, err := tx.Exec("INSERT INTO containment (child, parent) VALUES (?, ?)", c.Child, c.Parent); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES ('aggregated_at', ?)", data.AggregatedAt); err != nil {
		return err
	}

	return tx.Commit()
}

// ReadAggregated loads the aggregated location data.
func (s *Store) ReadAggregated() (*model.AggregatedData, error) {
	data := &model.AggregatedData{}

	// Locations
	rows, err := s.DB.Query("SELECT id, name, type, aliases, description, visual_description, first_chapter_idx, mention_count, chapter_indices FROM locations ORDER BY first_chapter_idx")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var loc model.AggregatedLocation
		var aliases, indices, visualDesc sql.NullString
		if err := rows.Scan(&loc.ID, &loc.Name, &loc.Type, &aliases, &loc.Description, &visualDesc, &loc.FirstChapterIndex, &loc.MentionCount, &indices); err != nil {
			return nil, err
		}
		if aliases.Valid {
			json.Unmarshal([]byte(aliases.String), &loc.Aliases)
		}
		if visualDesc.Valid {
			loc.VisualDescription = visualDesc.String
		}
		if indices.Valid {
			json.Unmarshal([]byte(indices.String), &loc.ChapterIndices)
		}
		data.Locations = append(data.Locations, loc)
	}

	// Relationships
	relRows, err := s.DB.Query("SELECT from_loc, to_loc, type, detail, first_chapter_idx FROM relationships ORDER BY first_chapter_idx")
	if err != nil {
		return nil, err
	}
	defer relRows.Close()
	for relRows.Next() {
		var rel model.AggregatedRelationship
		if err := relRows.Scan(&rel.From, &rel.To, &rel.Type, &rel.Detail, &rel.FirstChapterIndex); err != nil {
			return nil, err
		}
		data.Relationships = append(data.Relationships, rel)
	}

	// Containment
	cRows, err := s.DB.Query("SELECT child, parent FROM containment")
	if err != nil {
		return nil, err
	}
	defer cRows.Close()
	for cRows.Next() {
		var c model.Containment
		if err := cRows.Scan(&c.Child, &c.Parent); err != nil {
			return nil, err
		}
		data.Containment = append(data.Containment, c)
	}

	var aggAt sql.NullString
	s.DB.QueryRow("SELECT value FROM meta WHERE key = 'aggregated_at'").Scan(&aggAt)
	data.AggregatedAt = aggAt.String

	return data, nil
}

// WriteCoordinate inserts or updates a single location's coordinates.
func (s *Store) WriteCoordinate(c model.Coordinate) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO coordinates (location_id, x, y, confidence, manual) VALUES (?, ?, ?, ?, ?)",
		c.LocationID, c.X, c.Y, c.Confidence, c.Manual)
	return err
}

// ReadCoordinates loads all coordinates.
func (s *Store) ReadCoordinates() ([]model.Coordinate, error) {
	rows, err := s.DB.Query("SELECT location_id, x, y, confidence, manual FROM coordinates")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coords []model.Coordinate
	for rows.Next() {
		var c model.Coordinate
		if err := rows.Scan(&c.LocationID, &c.X, &c.Y, &c.Confidence, &c.Manual); err != nil {
			return nil, err
		}
		coords = append(coords, c)
	}
	return coords, rows.Err()
}

// ChapterCount returns the total number of chapters in the TOC.
func (s *Store) ChapterCount() int {
	var n int
	s.DB.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&n)
	return n
}

// ChapterTextCount returns how many chapters have been scraped.
func (s *Store) ChapterTextCount() int {
	var n int
	s.DB.QueryRow("SELECT COUNT(*) FROM chapter_text").Scan(&n)
	return n
}

// ExtractionCount returns how many chapters have been extracted.
func (s *Store) ExtractionCount() int {
	var n int
	s.DB.QueryRow("SELECT COUNT(*) FROM extraction_meta").Scan(&n)
	return n
}

// LocationCount returns the number of aggregated locations.
func (s *Store) LocationCount() int {
	var n int
	s.DB.QueryRow("SELECT COUNT(*) FROM locations").Scan(&n)
	return n
}

// ChapterCountByVolume returns chapter counts per volume.
func (s *Store) ChapterCountByVolume() map[string]int {
	m := make(map[string]int)
	rows, err := s.DB.Query("SELECT volume, COUNT(*) FROM chapters GROUP BY volume ORDER BY volume")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var vol string
		var cnt int
		rows.Scan(&vol, &cnt)
		m[vol] = cnt
	}
	return m
}

// ScrapedCountByVolume returns scraped chapter counts per volume.
func (s *Store) ScrapedCountByVolume() map[string]int {
	m := make(map[string]int)
	rows, err := s.DB.Query("SELECT c.volume, COUNT(*) FROM chapter_text ct JOIN chapters c ON ct.chapter_idx = c.idx GROUP BY c.volume ORDER BY c.volume")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var vol string
		var cnt int
		rows.Scan(&vol, &cnt)
		m[vol] = cnt
	}
	return m
}

// ExtractedCountByVolume returns extraction counts per volume.
func (s *Store) ExtractedCountByVolume() map[string]int {
	m := make(map[string]int)
	rows, err := s.DB.Query("SELECT c.volume, COUNT(*) FROM extraction_meta em JOIN chapters c ON em.chapter_idx = c.idx GROUP BY c.volume ORDER BY c.volume")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var vol string
		var cnt int
		rows.Scan(&vol, &cnt)
		m[vol] = cnt
	}
	return m
}
