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
var ErrUserAlreadyUnsubscribed = errors.New(" already unsubscribed, nothing to do")

type tasksStorageInt interface {
	getTask(uint64) (common.BotLeetCodeTask, error)
	saveTask(common.BotLeetCodeTask) error
}

type usersStorageInt interface {
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
	tasksStorage tasksStorageInt
	tasksCache   tasksStorageInt
	usersStorage usersStorageInt
}

// GetTask retrive task from all layers of storage in order and return ErrNoSuchTask if task isn't found
func (s *YDBandFileCacheController) GetTask(dateID uint64) (common.BotLeetCodeTask, error) {
	task, err := s.tasksCache.getTask(dateID)
	if err != nil {
		if err != ErrNoSuchTask {
			fmt.Println("Error on geting task from cache:", err)
		}
		task, err := s.tasksStorage.getTask(dateID)
		if err != nil {
			return task, err
		}

		// Since there were no such task in cache, let's add it
		err = s.tasksCache.saveTask(task)
		return task, err
	}
	return task, err
}

// SaveTask save task to all layers of storage
func (s *YDBandFileCacheController) SaveTask(task common.BotLeetCodeTask) error {
	err := s.tasksCache.saveTask(task)
	if err != nil {
		fmt.Println("Error on saving task from cache:", err)
	}
	return s.tasksStorage.saveTask(task)
}

// SubscribeUser subscribe and create user in storage if necessary.
// Returns ErrNoSuchUser ErrUserAlreadySubscribed if user were already subscribed
func (s *YDBandFileCacheController) SubscribeUser(user common.User) error {
	storedUser, err := s.usersStorage.getUser(user.ID)
	if err == ErrNoSuchUser {
		user.Subscribed = true
		return s.usersStorage.saveUser(user)
	} else if err != nil {
		return err
	}
	if storedUser.Subscribed {
		return ErrUserAlreadySubscribed
	}
	return s.usersStorage.subscribeUser(user.ID)
}

// UnsubscribeUser unsubscribing user with userID.
// Returns ErrUserAlreadyUnsubscribed if user were already subscribed
func (s *YDBandFileCacheController) UnsubscribeUser(userID uint64) error {
	user, err := s.usersStorage.getUser(userID)
	if err == ErrNoSuchUser {
		return ErrUserAlreadyUnsubscribed
	} else if err != nil {
		return err
	}
	if !user.Subscribed {
		return ErrUserAlreadyUnsubscribed
	}
	return s.usersStorage.unsubscribeUser(user.ID)
}

// GetSubscribedUsers necessary when we need to send notification to all subscribed users
func (s *YDBandFileCacheController) GetSubscribedUsers() ([]common.User, error) {
	return s.usersStorage.getSubscribedUsers()
}

// NewYDBandFileCacheController constructs default storage controller
func NewYDBandFileCacheController() *YDBandFileCacheController {
	databaseStorage := newYdbStorage()
	return &YDBandFileCacheController{
		tasksStorage: databaseStorage,
		tasksCache:   newFileCache(),
		usersStorage: databaseStorage,
	}
}
