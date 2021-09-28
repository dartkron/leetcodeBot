package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
)

// fileCache is a tasksStockpile with local filesystem backend
type fileCache struct {
	Path string
	Mask string
}

// getTask from local fs from path based on Path + mask
func (c *fileCache) getTask(dateID uint64) (common.BotLeetCodeTask, error) {
	cachePath := c.getTaskCachePath(dateID)
	cacheFile, err := os.Open(cachePath)
	if os.IsNotExist(err) {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}
	defer cacheFile.Close()
	bytes, err := io.ReadAll(cacheFile)
	if err != nil {
		return common.BotLeetCodeTask{}, err
	}
	task := common.BotLeetCodeTask{}
	err = json.Unmarshal(bytes, &task)
	return task, err
}

// saveTask to local fs cache storage
func (c *fileCache) saveTask(task common.BotLeetCodeTask) error {
	cachePath := c.getTaskCachePath(task.DateID)
	bytesTask, err := json.Marshal(task)
	if err != nil {
		return err
	}
	err = os.WriteFile(cachePath, bytesTask, 0644)
	return err
}

func (c *fileCache) getTaskCachePath(dateID uint64) string {
	return path.Join(c.Path, fmt.Sprintf(c.Mask, dateID))
}

// NewfileCache construct default fileCacher
func newFileCache() *fileCache {
	return &fileCache{
		Path: "/tmp/",
		Mask: "task_%d.cache",
	}
}
