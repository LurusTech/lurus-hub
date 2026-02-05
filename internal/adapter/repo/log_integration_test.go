package repo

import (
	"testing"
)

func TestLog_RecordLog_CreatesEntry(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	RecordLog(normal.Id, LogTypeSystem, "test log entry")

	var count int64
	LOG_DB.Model(&Log{}).Where("user_id = ?", normal.Id).Count(&count)
	if count == 0 {
		t.Error("expected at least 1 log entry, got 0")
	}

	var log Log
	LOG_DB.Where("user_id = ? AND type = ?", normal.Id, LogTypeSystem).
		Order("id desc").First(&log)
	if log.Content != "test log entry" {
		t.Errorf("Content = %q, want %q", log.Content, "test log entry")
	}
	if log.Username != "testnormal" {
		t.Errorf("Username = %q, want %q", log.Username, "testnormal")
	}
}

func TestLog_QueryByUserId(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	root, normal, _ := SeedTestUsers(t)

	// Create logs for different users
	RecordLog(root.Id, LogTypeSystem, "root log 1")
	RecordLog(root.Id, LogTypeSystem, "root log 2")
	RecordLog(normal.Id, LogTypeSystem, "normal log 1")

	var rootCount int64
	LOG_DB.Model(&Log{}).Where("user_id = ?", root.Id).Count(&rootCount)

	var normalCount int64
	LOG_DB.Model(&Log{}).Where("user_id = ?", normal.Id).Count(&normalCount)

	// Root may have extra logs from SeedTestUsers, but should have at least 2
	if rootCount < 2 {
		t.Errorf("root log count = %d, want >= 2", rootCount)
	}
	if normalCount < 1 {
		t.Errorf("normal log count = %d, want >= 1", normalCount)
	}
}

func TestChineseCharacters_LogContent(t *testing.T) {
	cleanup := SetupTestDB(t)
	defer cleanup()

	_, normal, _ := SeedTestUsers(t)

	chineseContent := "\u7528\u6237\u5145\u503c\u6210\u529f\uff0c\u5145\u503c\u91d1\u989d: 100\u5143"
	RecordLog(normal.Id, LogTypeTopup, chineseContent)

	var log Log
	LOG_DB.Where("user_id = ? AND type = ?", normal.Id, LogTypeTopup).
		Order("id desc").First(&log)
	if log.Content != chineseContent {
		t.Errorf("Content = %q, want %q", log.Content, chineseContent)
	}
}
