package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/dartkron/leetcodeBot/v3/internal/common"
)

// fileCache is a tasksStockpile with local filesystem backend
type fileCache struct {
	Path string
	Mask string
}

// getTask from local fs from path based on Path + mask
func (c *fileCache) getTask(ctx context.Context, dateID uint64) (common.BotLeetCodeTask, error) {
	respChan := make(chan common.BotLeetCodeTask)
	errChan := make(chan error)
	go func() {
		cachePath := c.getTaskCachePath(dateID)
		cacheFile, err := os.Open(cachePath)
		if os.IsNotExist(err) {
			errChan <- ErrNoSuchTask
			return
		}
		defer cacheFile.Close()
		bytes, err := io.ReadAll(cacheFile)
		if err != nil {
			errChan <- err
			return
		}
		task := common.BotLeetCodeTask{}
		err = json.Unmarshal(bytes, &task)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- task
	}()
	task := common.BotLeetCodeTask{}
	var err error
	select {
	case <-ctx.Done():
		err = common.ErrClosedContext
	case err = <-errChan:
	case task = <-respChan:
	}
	return task, err
}

// saveTask to local fs cache storage
func (c *fileCache) saveTask(ctx context.Context, task common.BotLeetCodeTask) error {
	errChan := make(chan error)
	go func() {
		cachePath := c.getTaskCachePath(task.DateID)
		bytesTask, err := json.Marshal(task)
		if err != nil {
			errChan <- err
			return
		}
		err = os.WriteFile(cachePath, bytesTask, 0644)
		errChan <- err
	}()
	var err error
	select {
	case <-ctx.Done():
		err = common.ErrClosedContext
	case err = <-errChan:
	}
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
