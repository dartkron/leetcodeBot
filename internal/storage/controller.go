package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/dartkron/leetcodeBot/v3/internal/common"
)

// ErrNoSuchTask returns when storage works, but such task is not found in the storage and the cache
var ErrNoSuchTask = errors.New("no such task")

// ErrNoSuchUser returns when storage works, but such user is not found in the storage and the cache
var ErrNoSuchUser = errors.New("no such user")

// ErrUserAlreadySubscribed when user already subscribed
var ErrUserAlreadySubscribed = errors.New("the user is already subscribed for recieving daily tasks at same time, nothing to do")

// ErrUserAlreadyUnsubscribed when user already unsubscribed or were newer subscribed before
var ErrUserAlreadyUnsubscribed = errors.New("already unsubscribed, nothing to do")

// ErrNoActiveUsersStorage storage controller can't do any users related actions without users storage.
// This is different from tasks, because for tasks, this application works as an intermediate storage, i.e. a local cache,
// but the situation with users is completely different: there is no other storage in the universe with our users.
var ErrNoActiveUsersStorage = errors.New("users storage isn't configured")

// ErrNoActiveTasksStorage by some reasong storage could not work
var ErrNoActiveTasksStorage = errors.New("tasks storage isn't configured or not available")

type tasksStorekeeper interface {
	getTask(context.Context, uint64) (common.BotLeetCodeTask, error)
	saveTask(context.Context, common.BotLeetCodeTask) error
}

type usersStorekeeper interface {
	getUser(context.Context, uint64) (common.User, error)
	saveUser(context.Context, common.User) error
	subscribeUser(context.Context, uint64, uint8) error
	unsubscribeUser(context.Context, uint64) error
	getSubscribedUsers(context.Context, uint8) ([]common.User, error)
}

// Controller should hide logic of storage layers inside
type Controller interface {
	GetTask(context.Context, uint64) (common.BotLeetCodeTask, error)
	SaveTask(context.Context, common.BotLeetCodeTask) error
	SubscribeUser(context.Context, common.User, uint8) error
	UnsubscribeUser(context.Context, uint64) error
	GetSubscribedUsers(context.Context, uint8) ([]common.User, error)
}

// YDBandFileCacheController is an instance of Controller which store users in database and store tasks into cache AND database
type YDBandFileCacheController struct {
	tasksDB    tasksStorekeeper
	tasksCache tasksStorekeeper
	usersDB    usersStorekeeper
}

func (s *YDBandFileCacheController) getTaskFromStorage(ctx context.Context, storage tasksStorekeeper, dateID uint64) (common.BotLeetCodeTask, error) {
	if storage == nil {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}
	return storage.getTask(ctx, dateID)
}

func (s *YDBandFileCacheController) getTaskFromCache(ctx context.Context, dateID uint64) (common.BotLeetCodeTask, error) {
	return s.getTaskFromStorage(ctx, s.tasksCache, dateID)
}

func (s *YDBandFileCacheController) getTaskFromDB(ctx context.Context, dateID uint64) (common.BotLeetCodeTask, error) {
	return s.getTaskFromStorage(ctx, s.tasksDB, dateID)
}

func (s *YDBandFileCacheController) saveTaskToStorage(ctx context.Context, storage tasksStorekeeper, task common.BotLeetCodeTask) error {
	if storage == nil {
		return nil
	}
	return storage.saveTask(ctx, task)
}

func (s *YDBandFileCacheController) saveTaskToCache(ctx context.Context, task common.BotLeetCodeTask) error {
	return s.saveTaskToStorage(ctx, s.tasksCache, task)
}

func (s *YDBandFileCacheController) saveTaskToDB(ctx context.Context, task common.BotLeetCodeTask) error {
	return s.saveTaskToStorage(ctx, s.tasksDB, task)
}

// GetTask retrive task from all layers of storage in order and return ErrNoSuchTask if task isn't found
func (s *YDBandFileCacheController) GetTask(ctx context.Context, dateID uint64) (common.BotLeetCodeTask, error) {
	task, err := s.getTaskFromCache(ctx, dateID)
	if err != nil {
		if err != ErrNoSuchTask {
			fmt.Printf("Error on geting task from cache: %q. Fallback to database.\n", err)
		}
		task, err := s.getTaskFromDB(ctx, dateID)
		if err != nil {
			return task, err
		}

		// Since there were no such task in cache, let's add it
		err = s.saveTaskToCache(ctx, task)
		if err != nil {
			// Have cache is a good idea, but no reason to stop show if it's broken
			fmt.Printf("Error on saving task to cache:: %q\n", err)
		}
		return task, nil
	}
	return task, err
}

// SaveTask save task to all layers of storage
func (s *YDBandFileCacheController) SaveTask(ctx context.Context, task common.BotLeetCodeTask) error {
	err := s.saveTaskToCache(ctx, task)
	if err != nil {
		fmt.Printf("Error on saving task to cache:: %q\n", err)
	}
	return s.saveTaskToDB(ctx, task)
}

// SubscribeUser subscribe and create user in storage if necessary.
// Returns ErrNoSuchUser ErrUserAlreadySubscribed if user were already subscribed
func (s *YDBandFileCacheController) SubscribeUser(ctx context.Context, user common.User, sendingHour uint8) error {
	if s.usersDB == nil {
		return ErrNoActiveUsersStorage
	}
	storedUser, err := s.usersDB.getUser(ctx, user.ID)
	if err != nil {
		if err == ErrNoSuchUser {
			user.Subscribed = true
			err = s.usersDB.saveUser(ctx, user)
		}
		return err
	}
	if storedUser.Subscribed && storedUser.SendingHour == sendingHour {
		return ErrUserAlreadySubscribed
	}
	return s.usersDB.subscribeUser(ctx, user.ID, sendingHour)
}

// UnsubscribeUser unsubscribing user with userID.
// Returns ErrUserAlreadyUnsubscribed if user were already subscribed
func (s *YDBandFileCacheController) UnsubscribeUser(ctx context.Context, userID uint64) error {
	if s.usersDB == nil {
		return ErrNoActiveUsersStorage
	}
	user, err := s.usersDB.getUser(ctx, userID)
	if err != nil {
		if err == ErrNoSuchUser {
			err = ErrUserAlreadyUnsubscribed
		}
		return err
	}
	if !user.Subscribed {
		return ErrUserAlreadyUnsubscribed
	}
	return s.usersDB.unsubscribeUser(ctx, user.ID)
}

// GetSubscribedUsers necessary when we need to send notification to all subscribed users
func (s *YDBandFileCacheController) GetSubscribedUsers(ctx context.Context, sendingHour uint8) ([]common.User, error) {
	if s.usersDB == nil {
		return []common.User{}, ErrNoActiveUsersStorage
	}
	return s.usersDB.getSubscribedUsers(ctx, sendingHour)
}

// NewYDBandFileCacheController constructs default storage controller
func NewYDBandFileCacheController() *YDBandFileCacheController {
	databaseStorage := newYdbStorage()
	return &YDBandFileCacheController{
		tasksDB:    databaseStorage,
		tasksCache: newFileCache(),
		usersDB:    databaseStorage,
	}
}
