package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
)

type fileCache struct {
	path string
}

func (c *fileCache) getTask(dateID uint64) (common.BotLeetCodeTask, error) {
	cachePath := c.getTaskCachePath(dateID)
	cacheFile, err := os.Open(cachePath)
	if os.IsNotExist(err) {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}
	defer cacheFile.Close()
	bytes, err := ioutil.ReadAll(cacheFile)
	if err != nil {
		return common.BotLeetCodeTask{}, err
	}
	task := common.BotLeetCodeTask{}
	err = json.Unmarshal(bytes, &task)
	return task, err
}

func (c *fileCache) saveTask(task common.BotLeetCodeTask) error {
	cachePath := c.getTaskCachePath(task.DateID)
	bytesTask, err := json.Marshal(task)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(cachePath, bytesTask, 0644)
	return err
}

func (c *fileCache) getTaskCachePath(dateID uint64) string {
	return path.Join(c.path, fmt.Sprintf("task_%d.cache", dateID))
}

func newFileCache() *fileCache {
	return &fileCache{path: "/tmp/"}
}
