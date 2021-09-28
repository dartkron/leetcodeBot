package storage

import (
	"errors"
	"fmt"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
)

// ErrNoSuchTask returns when storage works, but such task is not found in the storage and the cache
var ErrNoSuchTask = errors.New("no such task")

// ErrNoSuchUser returns when storage works, but such user is not found in the storage and the cache
var ErrNoSuchUser = errors.New("no such user")

// ErrUserAlreadySubscribed when user already subscribed
var ErrUserAlreadySubscribed = errors.New("the user is already subscribed, nothing to do")

// ErrUserAlreadyUnsubscribed when user already unsubscribed or were newer subscribed before
var ErrUserAlreadyUnsubscribed = errors.New("already unsubscribed, nothing to do")

// ErrNoActiveUsersStorage storage controller can't do any users related actions without users storage.
// This is different from tasks, because for tasks, this application works as an intermediate storage, i.e. a local cache,
// but the situation with users is completely different: there is no other storage in the universe with our users.
var ErrNoActiveUsersStorage = errors.New("users storage isn't configured")

type tasksStorekeeper interface {
	getTask(uint64) (common.BotLeetCodeTask, error)
	saveTask(common.BotLeetCodeTask) error
}

type usersStorekeeper interface {
	getUser(uint64) (common.User, error)
	saveUser(common.User) error
	subscribeUser(uint64) error
	unsubscribeUser(uint64) error
	getSubscribedUsers() ([]common.User, error)
}

// Controller should hide logic of storage layers inside
type Controller interface {
	GetTask(uint64) (common.BotLeetCodeTask, error)
	SaveTask(common.BotLeetCodeTask) error
	SubscribeUser(common.User) error
	UnsubscribeUser(uint64) error
	GetSubscribedUsers() ([]common.User, error)
}

// YDBandFileCacheController is an instance of Controller which store users in database and store tasks into cache AND database
type YDBandFileCacheController struct {
	tasksDB    tasksStorekeeper
	tasksCache tasksStorekeeper
	usersDB    usersStorekeeper
}

func (s *YDBandFileCacheController) getTaskFromStorage(storage tasksStorekeeper, dateID uint64) (common.BotLeetCodeTask, error) {
	if storage == nil {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}
	return storage.getTask(dateID)
}

func (s *YDBandFileCacheController) getTaskFromCache(dateID uint64) (common.BotLeetCodeTask, error) {
	return s.getTaskFromStorage(s.tasksCache, dateID)
}

func (s *YDBandFileCacheController) getTaskFromDB(dateID uint64) (common.BotLeetCodeTask, error) {
	return s.getTaskFromStorage(s.tasksDB, dateID)
}

func (s *YDBandFileCacheController) saveTaskToStorage(storage tasksStorekeeper, task common.BotLeetCodeTask) error {
	if storage == nil {
		return nil
	}
	return storage.saveTask(task)
}

func (s *YDBandFileCacheController) saveTaskToCache(task common.BotLeetCodeTask) error {
	return s.saveTaskToStorage(s.tasksCache, task)
}

func (s *YDBandFileCacheController) saveTaskToDB(task common.BotLeetCodeTask) error {
	return s.saveTaskToStorage(s.tasksDB, task)
}

// GetTask retrive task from all layers of storage in order and return ErrNoSuchTask if task isn't found
func (s *YDBandFileCacheController) GetTask(dateID uint64) (common.BotLeetCodeTask, error) {
	task, err := s.getTaskFromCache(dateID)
	if err != nil {
		if err != ErrNoSuchTask {
			fmt.Printf("Error on geting task from cache: %q. Fallback to database.\n", err)
		}
		task, err := s.getTaskFromDB(dateID)
		if err != nil {
			return task, err
		}

		// Since there were no such task in cache, let's add it
		err = s.saveTaskToCache(task)
		if err != nil {
			// Have cache is a good idea, but no reason to stop show if it's broken
			fmt.Printf("Error on saving task to cache:: %q\n", err)
		}
		return task, nil
	}
	return task, err
}

// SaveTask save task to all layers of storage
func (s *YDBandFileCacheController) SaveTask(task common.BotLeetCodeTask) error {
	err := s.saveTaskToCache(task)
	if err != nil {
		fmt.Printf("Error on saving task to cache:: %q\n", err)
	}
	return s.saveTaskToDB(task)
}

// SubscribeUser subscribe and create user in storage if necessary.
// Returns ErrNoSuchUser ErrUserAlreadySubscribed if user were already subscribed
func (s *YDBandFileCacheController) SubscribeUser(user common.User) error {
	if s.usersDB == nil {
		return ErrNoActiveUsersStorage
	}
	storedUser, err := s.usersDB.getUser(user.ID)
	if err != nil {
		if err == ErrNoSuchUser {
			user.Subscribed = true
			err = s.usersDB.saveUser(user)
		}
		return err
	}
	if storedUser.Subscribed {
		return ErrUserAlreadySubscribed
	}
	return s.usersDB.subscribeUser(user.ID)
}

// UnsubscribeUser unsubscribing user with userID.
// Returns ErrUserAlreadyUnsubscribed if user were already subscribed
func (s *YDBandFileCacheController) UnsubscribeUser(userID uint64) error {
	if s.usersDB == nil {
		return ErrNoActiveUsersStorage
	}
	user, err := s.usersDB.getUser(userID)
	if err != nil {
		if err == ErrNoSuchUser {
			err = ErrUserAlreadyUnsubscribed
		}
		return err
	}
	if !user.Subscribed {
		return ErrUserAlreadyUnsubscribed
	}
	return s.usersDB.unsubscribeUser(user.ID)
}

// GetSubscribedUsers necessary when we need to send notification to all subscribed users
func (s *YDBandFileCacheController) GetSubscribedUsers() ([]common.User, error) {
	if s.usersDB == nil {
		return []common.User{}, ErrNoActiveUsersStorage
	}
	return s.usersDB.getSubscribedUsers()
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
