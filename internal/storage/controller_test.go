package storage

import (
	"fmt"
	"sort"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
	"github.com/dartkron/leetcodeBot/v2/tests"
	"github.com/stretchr/testify/assert"
)

type MockTasksStorekeeper struct {
	tasks        map[uint64]common.BotLeetCodeTask
	callsJournal []string
	IDToFail     uint64
}

func (k *MockTasksStorekeeper) getTask(dateID uint64) (common.BotLeetCodeTask, error) {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("getTask %d", dateID))
	if dateID == k.IDToFail {
		return common.BotLeetCodeTask{}, tests.ErrBypassTest
	}
	if task, ok := k.tasks[dateID]; ok {
		return task, nil
	}
	return common.BotLeetCodeTask{}, ErrNoSuchTask
}
func (k *MockTasksStorekeeper) saveTask(task common.BotLeetCodeTask) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("saveTask %d", task.DateID))
	if task.DateID == k.IDToFail {
		return tests.ErrBypassTest
	}
	k.tasks[task.DateID] = task
	return nil
}

type MockUsersStorekeeper struct {
	users                      map[uint64]*common.User
	callsJournal               []string
	IDToFail                   uint64
	getSubscribedUsersMustFail bool
}

func (k *MockUsersStorekeeper) getUser(userID uint64) (common.User, error) {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("getUser %d", userID))
	if userID == k.IDToFail {
		return common.User{}, tests.ErrBypassTest
	}
	if user, ok := k.users[userID]; ok {
		return *user, nil
	}
	return common.User{}, ErrNoSuchUser
}

func (k *MockUsersStorekeeper) saveUser(user common.User) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("saveUser %d", user.ID))
	if user.ID == k.IDToFail {
		return tests.ErrBypassTest
	}
	k.users[user.ID] = &user
	return nil
}

func (k *MockUsersStorekeeper) subscribeUser(userID uint64) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("subscribeUser %d", userID))
	if userID == k.IDToFail {
		return tests.ErrBypassTest
	}
	if user, ok := k.users[userID]; ok {
		user.Subscribed = true
	} else {
		return ErrNoSuchUser
	}
	return nil
}

func (k *MockUsersStorekeeper) unsubscribeUser(userID uint64) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("unsubscribeUser %d", userID))
	if userID == k.IDToFail {
		return tests.ErrBypassTest
	}
	if user, ok := k.users[userID]; ok {
		user.Subscribed = false
	} else {
		return ErrNoSuchUser
	}
	return nil
}

func (k *MockUsersStorekeeper) getSubscribedUsers() ([]common.User, error) {
	k.callsJournal = append(k.callsJournal, "getSubscribedUsers")
	if k.getSubscribedUsersMustFail {
		return []common.User{}, tests.ErrBypassTest
	}
	resp := []common.User{}
	for _, user := range k.users {
		if user.Subscribed {
			resp = append(resp, *user)
		}
	}
	return resp, nil
}

func TestNewYDBandFileCacheController(t *testing.T) {
	storageController := NewYDBandFileCacheController()
	assert.NotNil(t, storageController.tasksCache, "NewYDBandFileCacheController should set tasksCache")
	assert.NotNil(t, storageController.tasksDB, "NewYDBandFileCacheController should set tasksDB")
	assert.NotNil(t, storageController.usersDB, "NewYDBandFileCacheController should set usersDB")
}

func TestNotConfiguredStorage(t *testing.T) {
	storageController := YDBandFileCacheController{}
	storageController.tasksCache = nil
	storageController.tasksDB = nil
	storageController.usersDB = nil
	assert.Equal(t, storageController.UnsubscribeUser(3435), ErrNoActiveUsersStorage, "UnsubscribeUser should return ErrNoActiveUsersStorage when users storage isn't set")
	assert.Equal(t, storageController.SubscribeUser(common.User{}), ErrNoActiveUsersStorage, "SubscribeUser should return ErrNoActiveUsersStorage when users storage isn't set")
	_, err := storageController.GetSubscribedUsers()
	assert.Equal(t, err, ErrNoActiveUsersStorage, "GetSubscribedUsers should return ErrNoActiveUsersStorage when users storage isn't set")
	assert.Nil(t, storageController.SaveTask(common.BotLeetCodeTask{}), "Unexpected error from SaveTask with unconfigured storage")
	_, err = storageController.GetTask(12312)
	assert.Equal(t, err, ErrNoSuchTask, "Unexpected error from GetTask with unconfigured storage")
}

func getTestController() (*YDBandFileCacheController, *MockTasksStorekeeper, *MockTasksStorekeeper) {
	cacheStorage := MockTasksStorekeeper{
		tasks: map[uint64]common.BotLeetCodeTask{
			12345: {
				LeetCodeTask: leetcodeclient.LeetCodeTask{
					QuestionID: 456,
					ItemID:     566,
					Title:      "Test title",
					Content:    "Test content",
					Hints:      []string{"First one", "Second one"},
					Difficulty: "Medium",
				},
				DateID: 12345,
			},
		},
	}
	DBStorage := MockTasksStorekeeper{
		tasks: map[uint64]common.BotLeetCodeTask{
			12345: {
				LeetCodeTask: leetcodeclient.LeetCodeTask{
					QuestionID: 4534,
					ItemID:     6655,
					Title:      "Very another test title",
					Content:    "Absolutely different with cache content",
					Hints:      []string{},
					Difficulty: "Easy",
				},
				DateID: 12345,
			},
			12346: {
				LeetCodeTask: leetcodeclient.LeetCodeTask{
					QuestionID: 457,
					ItemID:     567,
					Title:      "Test title1",
					Content:    "Test content1",
					Hints:      []string{"First one!", "Second one?"},
					Difficulty: "Hard",
				},
				DateID: 12346,
			},
		},
	}

	return &YDBandFileCacheController{
			tasksDB:    &DBStorage,
			tasksCache: &cacheStorage,
		},
		&cacheStorage,
		&DBStorage
}

func TestGetTaskFromCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	// Get task from cache
	task, err := storageController.GetTask(12345)
	assert.Nil(t, err, "Unexpected GetTask error")
	assert.Equal(t, task, cacheStorage.tasks[12345], "Recieved task differs with task in storage")
	assert.Empty(t, DBStorage.callsJournal, "Datase shoudn't be called when task persists in the cache")
	assert.Equal(t, cacheStorage.callsJournal, []string{"getTask 12345"}, "Cache calls amount differ with expectations")
}

func TestGetTaskFromDBMissedInCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	// Get task from DB and check that it will be saved in cache
	task, err := storageController.GetTask(12346)
	assert.Nil(t, err, "Unexpected GetTask error")
	assert.Equal(t, task, DBStorage.tasks[12346], "Recieved task differs with task in storage")
	cacheTask, ok := cacheStorage.tasks[12346]
	if assert.True(t, ok, "Returned task not saved to cache") {
		assert.Equal(t, task, cacheTask, "Recieved task differs with task in cache")
	}
	assert.Equal(t, DBStorage.callsJournal, []string{"getTask 12346"}, "DB calls differ with expectations")
	assert.Equal(t, cacheStorage.callsJournal, []string{"getTask 12346", "saveTask 12346"}, "Cache calls differ with expectations")
}

func TestGetTaskFromDBWithBrokenCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	// This ID persists in cache and in DB. Now we should get it from db and not save to the cache
	cacheStorage.IDToFail = 12345
	originalCacheTask := cacheStorage.tasks[12345]
	task, err := storageController.GetTask(12345)
	assert.Nil(t, err, "Unexpected GetTask error")
	assert.Equal(t, task, DBStorage.tasks[12345], "Recieved task differs with task in storage")
	assert.Equal(t, originalCacheTask, cacheStorage.tasks[12345], "Task was updated in cache, but shouldn't")
	assert.Equal(t, DBStorage.callsJournal, []string{"getTask 12345"}, "DB calls differ with expectations")
	assert.Equal(t, cacheStorage.callsJournal, []string{"getTask 12345", "saveTask 12345"}, "Cache calls differ with expectations")
}

func TestSaveTaskToDBAndCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	err := storageController.SaveTask(taskToSave)
	assert.Nil(t, err, "Unexpected SaveTask error")
	task, ok := cacheStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in cache after SaveTask") {
		assert.Equal(t, task, taskToSave, "Task saved in cache differ with sent one")
	}
	task, ok = DBStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in database after SaveTask") {
		assert.Equal(t, task, taskToSave, "Task saved in storage differ with sent one")
	}
	assert.Equal(t, cacheStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected cache calls journal")
	assert.Equal(t, DBStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected DB calls journal")
}

func TestSaveTaskWithBrokenCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	cacheStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)
	assert.Nil(t, err, "Unexpected SaveTask error")
	_, ok := cacheStorage.tasks[12345]
	assert.False(t, ok, "The task has been saved to the broken cache?!")
	task, ok := DBStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in DB after SaveTask") {
		assert.Equal(t, task, taskToSave, "Saved in DB task is not equal with the sent one")
	}
	assert.Equal(t, cacheStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected cache calls journal")
	assert.Equal(t, DBStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected DB calls journal")
}

func TestSaveTaskWithBrokenDB(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	DBStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected SaveTask error")
	_, ok := DBStorage.tasks[12345]
	assert.False(t, ok, "Task saved to the broken DB?!")
	task, ok := cacheStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in cache after SaveTask") {
		assert.Equal(t, task, taskToSave, "Saved in cache task is not equal with the sent one")
	}
	assert.Equal(t, cacheStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected cache calls journal")
	assert.Equal(t, DBStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected DB calls journal")
}

func TestSaveTaskWithBrokenAll(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	DBStorage.IDToFail = 12345
	cacheStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)
	assert.Equal(t, err, tests.ErrBypassTest, "Unxpected SaveTask error")
	_, ok := DBStorage.tasks[12345]
	assert.False(t, ok, "Task saved to the broken DB?!")
	_, ok = cacheStorage.tasks[12345]
	assert.False(t, ok, "Task saved to the broken cache?!")
	assert.Equal(t, cacheStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected cache call journal")
	assert.Equal(t, DBStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected DB call journal")
}

func TestSaveTaskToDBAndReplace(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	taskToSave.Content += "change"
	taskToSave.Title += "change"
	taskToSave.Difficulty += "change"
	taskToSave.ItemID = 88888
	taskToSave.Hints = []string{"New one"}

	err := storageController.SaveTask(taskToSave)
	assert.Nil(t, err, "Unxpected SaveTask error")
	task, ok := cacheStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in cache after SaveTask") {
		assert.Equal(t, task, taskToSave, "Saved in cache task is not equal with the sent one")
	}
	task, ok = DBStorage.tasks[12345]
	if assert.True(t, ok, "Task missed in DB after SaveTask") {
		assert.Equal(t, task, taskToSave, "Saved in DB task is not equal with the sent one")
	}
	assert.Equal(t, cacheStorage.callsJournal, []string{"saveTask 12345"}, "Unexpected cache call journal")
}

func getTestUsersStorekeeper() *MockUsersStorekeeper {
	return &MockUsersStorekeeper{
		getSubscribedUsersMustFail: false,
		users: map[uint64]*common.User{
			1124: {
				ID:         1124,
				ChatID:     1124,
				Username:   "testuser1124",
				FirstName:  "1124firstname",
				LastName:   "1124lastname",
				Subscribed: false,
			},
			1126: {
				ID:         1126,
				ChatID:     1126,
				Username:   "testuser1126",
				FirstName:  "1126firstname",
				LastName:   "1126lastname",
				Subscribed: true,
			},
			1128: {
				ID:         1128,
				ChatID:     1128,
				Username:   "testuser1128",
				FirstName:  "1128firstname",
				LastName:   "1128lastname",
				Subscribed: false,
			},
			1120: {
				ID:         1120,
				ChatID:     1120,
				Username:   "testuser1120",
				FirstName:  "1120firstname",
				LastName:   "1120lastname",
				Subscribed: true,
			},
		},
	}
}

func TestGetSubscribedUsers(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	testCases := []map[uint64]bool{
		{1126: true, 1120: true},
		{1126: true, 1120: true, 1124: true},
		{1120: true},
		{},
	}

	for _, testCase := range testCases {
		usersStore.callsJournal = []string{}
		awaitedList := []common.User{}
		for _, user := range usersStore.users {
			if _, ok := testCase[user.ID]; ok {
				user.Subscribed = true
				awaitedList = append(awaitedList, *user)
			} else {
				user.Subscribed = false
			}
		}
		list, err := storageController.GetSubscribedUsers()
		assert.Nil(t, err, "Unexpected GetSubscribedUsers error")
		sort.Slice(list, func(i, j int) bool {
			return list[i].ID < list[j].ID
		})
		sort.Slice(awaitedList, func(i, j int) bool {
			return awaitedList[i].ID < awaitedList[j].ID
		})
		assert.Equal(t, list, awaitedList, "Unxpected users list from GetSubscribedUsers")
		assert.Equal(t, usersStore.callsJournal, []string{"getSubscribedUsers"}, "Unexpected users store call list")
	}
}

func TestGetSubscribedUsersWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.getSubscribedUsersMustFail = true
	list, err := storageController.GetSubscribedUsers()
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected GetSubscribedUsers error")
	assert.Equal(t, list, []common.User{}, "Empty list should be returned from GetSubscribedUsers on error")
	assert.Equal(t, usersStore.callsJournal, []string{"getSubscribedUsers"}, "Unexpected users store call list")
}

func TestSubscribeUserNew(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	newUser := common.User{
		ID:         1000,
		ChatID:     1000,
		Username:   "newUser1000",
		FirstName:  "1000firstname",
		LastName:   "1000lastname",
		Subscribed: false,
	}
	err := storageController.SubscribeUser(newUser)
	assert.Nil(t, err, "Unexpected SubscribeUser error")
	newUser.Subscribed = true
	assert.Equal(t, *usersStore.users[1000], newUser, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1000", "saveUser 1000"}, "Unexpected users store call list")
}

func TestSubscribeUserOld(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1124]
	err := storageController.SubscribeUser(user)
	assert.Nil(t, err, "Unexpected SubscribeUser error")
	user.Subscribed = true
	assert.Equal(t, *usersStore.users[1124], user, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1124", "subscribeUser 1124"}, "Unexpected users store call list")
}

func TestSubscribeUserAlreadySubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.users[1124].Subscribed = true
	userToSend := *usersStore.users[1124]
	err := storageController.SubscribeUser(userToSend)
	assert.Equal(t, err, ErrUserAlreadySubscribed, "Unexpected SubscribeUser error")
	assert.Equal(t, *usersStore.users[1124], userToSend, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1124"}, "Unexpected users store call list")
}

func TestSubscribeUserWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.IDToFail = 1124
	userToSend := *usersStore.users[1124]
	err := storageController.SubscribeUser(userToSend)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected SubscribeUser error")
	assert.Equal(t, *usersStore.users[1124], userToSend, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1124"}, "Unexpected users store call list")
}

func TestUnsubscribeUserNew(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	err := storageController.UnsubscribeUser(1000)
	assert.Equal(t, err, ErrUserAlreadyUnsubscribed, "Unexpected SubscribeUser error")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1000"}, "Unexpected users store call list")
}

func TestUnsubscribeUserOldNotSubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1124]
	err := storageController.UnsubscribeUser(1124)
	assert.Equal(t, err, ErrUserAlreadyUnsubscribed, "Unexpected SubscribeUser error")
	assert.Equal(t, *usersStore.users[1124], user, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1124"}, "Unexpected users store call list")
}

func TestUnsubscribeUserAlreadySubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1126]
	err := storageController.UnsubscribeUser(1126)
	assert.Nil(t, err, "Unexpected SubscribeUser error")
	user.Subscribed = false
	assert.Equal(t, *usersStore.users[1126], user, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1126", "unsubscribeUser 1126"}, "Unexpected users store call list")
}

func TestUnsubscribeUserWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.IDToFail = 1124
	userToSend := *usersStore.users[1124]
	err := storageController.UnsubscribeUser(1124)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected SubscribeUser error")
	assert.Equal(t, *usersStore.users[1124], userToSend, "Stored user differ with the sent one")
	assert.Equal(t, usersStore.callsJournal, []string{"getUser 1124"}, "Unexpected users store call list")
}
