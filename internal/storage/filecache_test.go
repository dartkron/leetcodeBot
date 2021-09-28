package storage

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
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
	if err != nil {
		t.Errorf("Error on creating new tmp dir: %q", err)
	}
	return tempDir
}

func TestGetTaskFileCache(t *testing.T) {
	fileStorage := getTestFileStorage()
	testCases := map[uint64]testCase{
		21240926: {nil, common.BotLeetCodeTask{
			DateID: 21240926,
			LeetCodeTask: leetcodeclient.LeetCodeTask{
				QuestionID: 432,
				ItemID:     3982,
				Title:      "Test question title",
				Content:    "You are given an <code>n x n</code> something, do something <code>0</code> or <code>1</code>.\n\n",
				Hints:      []string{"First hint is to be good", "Second hint is not to be evil"},
				Difficulty: "Hard",
			},
		}},
		21004095: {ErrNoSuchTask, common.BotLeetCodeTask{}},
	}
	for id, details := range testCases {
		task, err := fileStorage.getTask(id)
		if err != details.err {
			t.Errorf("Got an error %q, but expected %q", err, details.err)
		}

		if !reflect.DeepEqual(task, details.loadedTask) {
			t.Errorf("%q not equal %q", task.LeetCodeTask, details.loadedTask.LeetCodeTask)
		}
	}
	fileStorage.Mask = fileStorage.Mask + "_broken"
	_, err := fileStorage.getTask(21240926)
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Parse of broken JSON file should return error about invalid character")
	}

	fileStorage = getTestFileStorage()
	tempDir := createTempDir(t)
	fileStorage.Path = tempDir
	defer os.RemoveAll(tempDir)
	os.Mkdir(fileStorage.getTaskCachePath(21240926), 0644)
	_, err = fileStorage.getTask(21240926)
	if err == nil {
		t.Errorf("Trying of load directory as file should return error, but nil is returned")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("Error on load directory as file should contain \"is a directory\" substring, but %q is returned", err.Error())
	}

}

func TestSaveTaskFileCache(t *testing.T) {
	fileStorage := getTestFileStorage()
	task, err := fileStorage.getTask(21240926)
	if err != nil {
		t.Fatalf("Unexpected error on getting task: %q", err)
	}
	oldFileName := fileStorage.getTaskCachePath(task.DateID)

	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	fileStorage.Mask = "another%dmask"
	fileStorage.Path = tempDir
	newFileName := fileStorage.getTaskCachePath(task.DateID)
	err = fileStorage.saveTask(task)
	if err != nil {
		t.Fatalf("Unexpected error of saving file: %q", err)
	}
	defer os.Remove(newFileName)
	oldFile, err := os.ReadFile(oldFileName)
	if err != nil {
		t.Errorf("Error on reading file %q", err)
	}
	newFile, err := os.ReadFile(newFileName)
	if err != nil {
		t.Errorf("Error on reading file %q", err)
	}
	if !bytes.Equal(oldFile, newFile) {
		t.Errorf("File which were used for load and one which appeared on save are different!")
	}

	os.RemoveAll(tempDir)
	err = fileStorage.saveTask(task)
	if err == nil {
		t.Errorf("Error should be returned when trying SaveTask in unexistant directory")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("\"no such file or directory\" substring not found in the error on trying save to nonexisting path")
	}
}
