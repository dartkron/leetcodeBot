package storage

import (
	"context"
	"os"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	err        error
	loadedTask common.BotLeetCodeTask
}

func getTestFileStorage() fileCache {
	fileStorage := newFileCache()
	fileStorage.Path = "../../tests/data"
	return *fileStorage
}

func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "leetcodebotTest")
	assert.Nil(t, err, "os.MkdirTemp error")
	return tempDir
}

func TestGetTaskFileCache(t *testing.T) {
	fileStorage := getTestFileStorage()
	testCases := map[uint64]testCase{
		21240926: {nil, common.BotLeetCodeTask{
			DateID: 21240926,
			LeetCodeTask: leetcodeclient.LeetCodeTask{
				QuestionID: 432,
				TitleSlug:  "test-task1",
				Title:      "Test question title",
				Content:    "You are given an <code>n x n</code> something, do something <code>0</code> or <code>1</code>.\n\n",
				Hints:      []string{"First hint is to be good", "Second hint is not to be evil"},
				Difficulty: "Hard",
			},
		}},
		21004095: {ErrNoSuchTask, common.BotLeetCodeTask{}},
	}
	for id, details := range testCases {
		task, err := fileStorage.getTask(context.Background(), id)
		assert.Equal(t, err, details.err, "Unexpected error")
		assert.Equal(t, task, details.loadedTask, "Loaded task mismatch")
	}
	fileStorage.Mask = fileStorage.Mask + "_broken"
	_, err := fileStorage.getTask(context.Background(), 21240926)
	assert.Contains(t, err.Error(), "invalid character", "arse of broken JSON file should return error about invalid character")
	fileStorage = getTestFileStorage()
	tempDir := createTempDir(t)
	fileStorage.Path = tempDir
	defer os.RemoveAll(tempDir)
	os.Mkdir(fileStorage.getTaskCachePath(21240926), 0644)
	_, err = fileStorage.getTask(context.Background(), 21240926)
	if assert.NotNil(t, err, "Load directory as file should return an error") {
		assert.Contains(t, err.Error(), "is a directory", "Error on load directory as file should contain \"is a directory\"")
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	task, err := fileStorage.getTask(ctx, 21240926)
	assert.Equal(t, common.ErrClosedContext, err, "Unexpected error returned")
	assert.Equal(t, common.BotLeetCodeTask{}, task, "Empty task expected in error case")
}

func TestSaveTaskFileCache(t *testing.T) {
	fileStorage := getTestFileStorage()
	task, err := fileStorage.getTask(context.Background(), 21240926)
	assert.Nil(t, err, "Unexpected getTask error")
	oldFileName := fileStorage.getTaskCachePath(task.DateID)

	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	fileStorage.Mask = "another%dmask"
	fileStorage.Path = tempDir
	newFileName := fileStorage.getTaskCachePath(task.DateID)
	err = fileStorage.saveTask(context.Background(), task)
	assert.Nil(t, err, "Unexpected saveTask error")
	defer os.Remove(newFileName)
	oldFile, err := os.ReadFile(oldFileName)
	assert.Nil(t, err, "Unexpected os.ReadFile error")
	newFile, err := os.ReadFile(newFileName)
	assert.Nil(t, err, "Unexpected os.ReadFile error")
	assert.Equal(t, oldFile, newFile, "File which were used for load and one which appeared on save are different!")

	os.RemoveAll(tempDir)
	err = fileStorage.saveTask(context.Background(), task)
	if assert.NotNil(t, err, "Unexpected saveTask error") {
		assert.Contains(t, err.Error(), "no such file or directory", "\"no such file or directory\" substring not found in the error on trying save to nonexisting path")
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	err = fileStorage.saveTask(ctx, task)
	assert.Equal(t, common.ErrClosedContext, err, "Unexpected error returned")
}

func TestNewFileCache(t *testing.T) {
	fileCache := newFileCache()
	assert.NotEmpty(t, fileCache.Mask, "Mask should be set in constructor")
	assert.NotEmpty(t, fileCache.Path, "Path should be set in constructor")
}
