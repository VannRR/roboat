// newsboatdb, for use with https://github.com/newsboat/newsboat
package newsboatdb

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/* newsboat database schema
CREATE TABLE rss_feed (
	rssurl VARCHAR(1024) PRIMARY KEY NOT NULL,
	url VARCHAR(1024) NOT NULL,
	title VARCHAR(1024) NOT NULL ,
	lastmodified INTEGER(11) NOT NULL DEFAULT 0,
	is_rtl INTEGER(1) NOT NULL DEFAULT 0,
	etag VARCHAR(128) NOT NULL DEFAULT ""
);
CREATE TABLE rss_item (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	guid VARCHAR(64) NOT NULL,
	title VARCHAR(1024) NOT NULL,
	author VARCHAR(1024) NOT NULL,
	url VARCHAR(1024) NOT NULL,
	feedurl VARCHAR(1024) NOT NULL,
	pubDate INTEGER NOT NULL,
	content VARCHAR(65535) NOT NULL,
	unread INTEGER(1) NOT NULL ,
	enclosure_url VARCHAR(1024),
	enclosure_type VARCHAR(1024),
	enqueued INTEGER(1) NOT NULL DEFAULT 0,
	flags VARCHAR(52),
	deleted INTEGER(1) NOT NULL DEFAULT 0,
	base VARCHAR(128) NOT NULL DEFAULT "",
	content_mime_type VARCHAR(255) NOT NULL DEFAULT "",
	enclosure_description VARCHAR(1024) NOT NULL DEFAULT "",
	enclosure_description_mime_type VARCHAR(128) NOT NULL DEFAULT ""
);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE google_replay (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	guid VARCHAR(64) NOT NULL,
	state INTEGER NOT NULL,
	ts INTEGER NOT NULL
);
CREATE INDEX idx_rssurl ON rss_feed(rssurl);
CREATE INDEX idx_guid ON rss_item(guid);
CREATE INDEX idx_feedurl ON rss_item(feedurl);
CREATE TABLE sqlite_stat1(tbl,idx,stat);
CREATE TABLE sqlite_stat4(tbl,idx,neq,nlt,ndlt,sample);
CREATE INDEX idx_lastmodified ON rss_feed(lastmodified);
CREATE INDEX idx_deleted ON rss_item(deleted);
CREATE TABLE metadata (
	db_schema_version_major INTEGER NOT NULL,
	db_schema_version_minor INTEGER NOT NULL
);
*/

type RSSFeed struct {
	Title       string
	RssURL      string
	UnreadItems int
	TotalItems  int
}

type RSSItem struct {
	ID      int64
	Title   string
	FeedURL string
	URL     string
	PubDate time.Time
	Unread  bool
}

type DBInterface interface {
	Close() error
	GetFeeds() ([]RSSFeed, error)
	GetItems(feedUrl string) ([]RSSItem, error)
	ToggleUnread(id int) error
	SetUnread(id int, unread bool) error
}

type NewsBoatDB struct {
	conn *sql.DB
	len  int
}

func NewNewsBoatDB(dbPath string) (*NewsBoatDB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &NewsBoatDB{
		conn: conn,
	}, nil
}

// Close closes the database connection.
func (db *NewsBoatDB) Close() error {
	return db.conn.Close()
}

func (db *NewsBoatDB) GetFeeds() ([]RSSFeed, error) {
	rows, err := db.conn.Query("SELECT rssurl, title FROM rss_feed")
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var feeds []RSSFeed
	for rows.Next() {
		var feed RSSFeed
		if err := rows.Scan(&feed.RssURL, &feed.Title); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		feed.UnreadItems, feed.TotalItems, err = db.getItemCount(feed.RssURL)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

func (db *NewsBoatDB) GetItems(feedUrl string) ([]RSSItem, error) {
	rows, err := db.conn.Query(
		"SELECT id, title, url, pubDate, unread FROM rss_item WHERE feedurl = ? ORDER BY pubDate DESC", feedUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to query RSS items: %w", err)
	}
	defer rows.Close()

	var items []RSSItem
	for rows.Next() {
		var item RSSItem
		var unixDate int64
		if err := rows.Scan(&item.ID, &item.Title, &item.URL, &unixDate, &item.Unread); err != nil {
			return nil, fmt.Errorf("failed to scan RSS item: %w", err)
		}
		item.PubDate = time.Unix(unixDate, 0)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (db *NewsBoatDB) ToggleUnread(id int) error {
	return db.withTransaction(func(tx *sql.Tx) error {
		var unread bool
		if err := tx.QueryRow("SELECT unread FROM rss_item WHERE id = ?", id).Scan(&unread); err != nil {
			return fmt.Errorf("failed to retrieve unread value: %w", err)
		}

		newUnread := "1"
		if unread {
			newUnread = "0"
		}

		if _, err := tx.Exec("UPDATE rss_item SET unread = ? WHERE id = ?", newUnread, id); err != nil {
			return fmt.Errorf("failed to update unread value: %w", err)
		}

		return nil
	})
}

func (db *NewsBoatDB) SetUnread(id int, unread bool) error {
	newUnread := "0"
	if unread {
		newUnread = "1"
	}

	return db.withTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec("UPDATE rss_item SET unread = ? WHERE id = ?", newUnread, id); err != nil {
			return fmt.Errorf("failed to update unread value: %w", err)
		}
		return nil
	})
}

// Utility functions

func (db *NewsBoatDB) withTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *NewsBoatDB) getItemCount(feedURL string) (int, int, error) {
	row := db.conn.QueryRow(`
        SELECT
            COUNT(*) as total_items,
            COALESCE(SUM(CASE WHEN unread THEN 1 ELSE 0 END), 0) as unread_items
        FROM rss_item
        WHERE feedurl = ?`, feedURL)

	var totalItems, unreadItems int
	if err := row.Scan(&totalItems, &unreadItems); err != nil {
		return 0, 0, fmt.Errorf("failed to scan row: %w", err)
	}
	return unreadItems, totalItems, nil
}
