package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/constant"
	"github.com/LurusTech/lurus-api/internal/pkg/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupContextTestDB sets up an in-memory SQLite database for context task tests.
func setupContextTestDB(t *testing.T) func() {
	t.Helper()

	dbName := fmt.Sprintf("file:contexttest%d?mode=memory&cache=shared", testDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	tables := []interface{}{
		&repo.User{},
		&repo.Channel{},
		&repo.Ability{},
		&repo.Task{},
		&repo.Midjourney{},
		&repo.Option{},
	}
	for _, tbl := range tables {
		if err := db.AutoMigrate(tbl); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			t.Fatalf("auto migrate failed for %T: %v", tbl, err)
		}
	}

	// Save previous state
	prevDB := repo.DB
	prevLogDB := repo.LOG_DB
	prevSQLite := common.UsingSQLite
	prevPG := common.UsingPostgreSQL
	prevRedis := common.RedisEnabled
	prevMemCache := common.MemoryCacheEnabled

	repo.DB = db
	repo.LOG_DB = db
	repo.InitCol()
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	// Initialize OptionMap
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMapRWMutex.Unlock()

	cleanup := func() {
		repo.DB = prevDB
		repo.LOG_DB = prevLogDB
		common.UsingSQLite = prevSQLite
		common.UsingPostgreSQL = prevPG
		common.RedisEnabled = prevRedis
		common.MemoryCacheEnabled = prevMemCache
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return cleanup
}

// TestUpdateTaskBulkWithContext_WithEmptyTasks tests task update with no tasks in database.
func TestUpdateTaskBulkWithContext_WithEmptyTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateTaskBulkWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestUpdateTaskBulkWithContext_WithNullTaskID tests handling of tasks with empty TaskID.
func TestUpdateTaskBulkWithContext_WithNullTaskID(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create tasks with empty TaskID
	tasks := []*repo.Task{
		{
			TaskID:    "",
			Platform:  constant.TaskPlatformSuno,
			Status:    "PENDING",
			Progress:  "0%",
			ChannelId: 1,
		},
		{
			TaskID:    "",
			Platform:  constant.TaskPlatformSuno,
			Status:    "PENDING",
			Progress:  "0%",
			ChannelId: 1,
		},
	}
	for _, task := range tasks {
		if err := repo.DB.Create(task).Error; err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// Use a very short timeout - the actual ticker won't fire, but context cancel will
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateTaskBulkWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK - context cancelled
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestUpdateMidjourneyTaskBulkWithContext_WithEmptyTasks tests midjourney task update with no tasks.
func TestUpdateMidjourneyTaskBulkWithContext_WithEmptyTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateMidjourneyTaskBulkWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestUpdateMidjourneyTaskBulkWithContext_WithPendingTasks tests midjourney task update with pending tasks.
func TestUpdateMidjourneyTaskBulkWithContext_WithPendingTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create a test channel first
	channel := &repo.Channel{
		Name:   "test-mj-channel",
		Type:   1,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
	}
	if err := repo.DB.Create(channel).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// Create pending midjourney tasks
	tasks := []*repo.Midjourney{
		{
			MjId:      "mj-task-1",
			ChannelId: channel.Id,
			Status:    "PENDING",
			Progress:  "0%",
		},
	}
	for _, task := range tasks {
		if err := repo.DB.Create(task).Error; err != nil {
			t.Fatalf("failed to create midjourney task: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		UpdateMidjourneyTaskBulkWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestAutomaticallyUpdateChannelsWithContext_WithChannel tests channel update with existing channels.
func TestAutomaticallyUpdateChannelsWithContext_WithChannel(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create a test channel
	channel := &repo.Channel{
		Name:    "test-channel",
		Type:    1,
		Key:     "test-key",
		Status:  common.ChannelStatusEnabled,
		Balance: 100.0,
	}
	if err := repo.DB.Create(channel).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 60) // Long interval, won't fire
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestAutomaticallyTestChannelsWithContext_MasterEnabled tests channel testing when master and enabled.
func TestAutomaticallyTestChannelsWithContext_MasterEnabled(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Set up master node
	prevMaster := common.IsMasterNode
	common.IsMasterNode = true
	defer func() { common.IsMasterNode = prevMaster }()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyTestChannelsWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestMultipleFunctionsWithSameDB tests multiple context functions sharing same database state.
func TestMultipleFunctionsWithSameDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create test data
	channel := &repo.Channel{
		Name:   "shared-channel",
		Type:   1,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
	}
	repo.DB.Create(channel)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	var completed atomic.Int32
	const numFuncs = 3

	go func() {
		UpdateTaskBulkWithContext(ctx)
		completed.Add(1)
	}()

	go func() {
		UpdateMidjourneyTaskBulkWithContext(ctx)
		completed.Add(1)
	}()

	go func() {
		AutomaticallyUpdateChannelsWithContext(ctx, 60)
		completed.Add(1)
	}()

	<-ctx.Done()

	// Wait for all to complete
	timeout := time.After(3 * time.Second)
	for completed.Load() < numFuncs {
		select {
		case <-timeout:
			t.Fatalf("only %d/%d functions completed", completed.Load(), numFuncs)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// TestContextCancelDuringDBOperation tests context cancellation during database operations.
func TestContextCancelDuringDBOperation(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create many tasks to make DB operation take longer
	for i := 0; i < 100; i++ {
		task := &repo.Task{
			TaskID:    fmt.Sprintf("task-%d", i),
			Platform:  constant.TaskPlatformSuno,
			Status:    "PENDING",
			Progress:  "0%",
			ChannelId: 1,
		}
		repo.DB.Create(task)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		UpdateTaskBulkWithContext(ctx)
		close(done)
	}()

	// Cancel almost immediately
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not respond to cancel")
	}
}

// TestOperationSettingIntegration tests integration with operation settings.
func TestOperationSettingIntegration(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	prevMaster := common.IsMasterNode
	common.IsMasterNode = true
	defer func() { common.IsMasterNode = prevMaster }()

	// Store a test monitor setting
	setting := operation_setting.MonitorSetting{
		AutoTestChannelEnabled: false,
		AutoTestChannelMinutes: 60,
	}
	settingJSON, _ := json.Marshal(setting)
	repo.DB.Create(&repo.Option{
		Key:   "MonitorSetting",
		Value: string(settingJSON),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		AutomaticallyTestChannelsWithContext(ctx)
		close(done)
	}()

	<-ctx.Done()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("function did not exit after context timeout")
	}
}

// TestUpdateMidjourneyTasks_DirectCall tests the updateMidjourneyTasks helper function directly.
func TestUpdateMidjourneyTasks_DirectCall_EmptyTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	ctx := context.Background()
	// Should not panic with empty tasks
	updateMidjourneyTasks(ctx)
}

// TestUpdateMidjourneyTasks_DirectCall_WithTasks tests updateMidjourneyTasks with actual tasks.
func TestUpdateMidjourneyTasks_DirectCall_WithTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create tasks with empty MjId (null task IDs)
	tasks := []*repo.Midjourney{
		{
			MjId:      "",
			ChannelId: 1,
			Status:    "PENDING",
			Progress:  "0%",
		},
		{
			MjId:      "",
			ChannelId: 1,
			Status:    "PENDING",
			Progress:  "0%",
		},
	}
	for _, task := range tasks {
		if err := repo.DB.Create(task).Error; err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	ctx := context.Background()
	updateMidjourneyTasks(ctx)

	// Verify tasks were updated to FAILURE
	var updatedTasks []repo.Midjourney
	repo.DB.Find(&updatedTasks)
	for _, task := range updatedTasks {
		if task.Status != "FAILURE" {
			t.Errorf("expected status FAILURE, got %s", task.Status)
		}
	}
}

// TestUpdateMidjourneyTasks_DirectCall_WithValidTasks tests updateMidjourneyTasks with valid MjId.
// This test verifies the code path when tasks have valid MjId but channel is not cached.
func TestUpdateMidjourneyTasks_DirectCall_WithValidTasks(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create tasks with valid MjId but for a non-existent channel
	// This will trigger the CacheGetChannel error path
	tasks := []*repo.Midjourney{
		{
			MjId:      "mj-valid-1",
			ChannelId: 999, // Non-existent channel
			Status:    "PENDING",
			Progress:  "0%",
		},
	}
	for _, task := range tasks {
		if err := repo.DB.Create(task).Error; err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	ctx := context.Background()
	// This will hit the CacheGetChannel error path and set tasks to FAILURE
	updateMidjourneyTasks(ctx)

	// Verify task was updated to FAILURE due to channel not found
	var updatedTask repo.Midjourney
	repo.DB.First(&updatedTask)
	if updatedTask.Status != "FAILURE" {
		t.Errorf("expected status FAILURE, got %s", updatedTask.Status)
	}
}

// TestUpdateMidjourneyTasks_ContextCancellation tests cancellation during updateMidjourneyTasks.
func TestUpdateMidjourneyTasks_ContextCancellation(t *testing.T) {
	cleanup := setupContextTestDB(t)
	defer cleanup()

	// Create many channels to increase chance of hitting cancellation check
	for i := 0; i < 10; i++ {
		channel := &repo.Channel{
			Name:    fmt.Sprintf("test-channel-%d", i),
			Type:    1,
			Key:     fmt.Sprintf("key-%d", i),
			Status:  common.ChannelStatusEnabled,
			BaseURL: stringPtr("http://localhost:8080"),
		}
		repo.DB.Create(channel)

		task := &repo.Midjourney{
			MjId:      fmt.Sprintf("mj-task-%d", i),
			ChannelId: channel.Id,
			Status:    "PENDING",
			Progress:  "0%",
		}
		repo.DB.Create(task)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	// Should exit early due to context cancellation
	updateMidjourneyTasks(ctx)
}

func stringPtr(s string) *string {
	return &s
}

// Benchmark integration tests
func BenchmarkUpdateTaskWithDB_Cancel(b *testing.B) {
	// This benchmark uses real DB
	dbName := "file:benchtest?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	db.AutoMigrate(&repo.Task{})

	prevDB := repo.DB
	repo.DB = db
	defer func() { repo.DB = prevDB }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		UpdateTaskBulkWithContext(ctx)
	}
}

func BenchmarkUpdateMidjourneyWithDB_Cancel(b *testing.B) {
	dbName := "file:benchtest2?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	db.AutoMigrate(&repo.Midjourney{}, &repo.Channel{})

	prevDB := repo.DB
	repo.DB = db
	defer func() { repo.DB = prevDB }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		UpdateMidjourneyTaskBulkWithContext(ctx)
	}
}
