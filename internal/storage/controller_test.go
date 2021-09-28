package storage

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
)

var ErrBypassTest error = errors.New("test bypass error")

type MockTasksStorekeeper struct {
	tasks        map[uint64]common.BotLeetCodeTask
	callsJournal []string
	IDToFail     uint64
}

func (k *MockTasksStorekeeper) getTask(dateID uint64) (common.BotLeetCodeTask, error) {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("getTask %d", dateID))
	if dateID == k.IDToFail {
		return common.BotLeetCodeTask{}, ErrBypassTest
	}
	if task, ok := k.tasks[dateID]; ok {
		return task, nil
	}
	return common.BotLeetCodeTask{}, ErrNoSuchTask
}
func (k *MockTasksStorekeeper) saveTask(task common.BotLeetCodeTask) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("saveTask %d", task.DateID))
	if task.DateID == k.IDToFail {
		return ErrBypassTest
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
		return common.User{}, ErrBypassTest
	}
	if user, ok := k.users[userID]; ok {
		return *user, nil
	}
	return common.User{}, ErrNoSuchUser
}

func (k *MockUsersStorekeeper) saveUser(user common.User) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("saveUser %d", user.ID))
	if user.ID == k.IDToFail {
		return ErrBypassTest
	}
	k.users[user.ID] = &user
	return nil
}

func (k *MockUsersStorekeeper) subscribeUser(userID uint64) error {
	k.callsJournal = append(k.callsJournal, fmt.Sprintf("subscribeUser %d", userID))
	if userID == k.IDToFail {
		return ErrBypassTest
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
		return ErrBypassTest
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
		return []common.User{}, ErrBypassTest
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
	if storageController.tasksCache == nil {
		t.Error("NewYDBandFileCacheController should set tasksCache")
	}
	if storageController.tasksDB == nil {
		t.Error("NewYDBandFileCacheController should set tasksDB")
	}
	if storageController.usersDB == nil {
		t.Error("NewYDBandFileCacheController should set usersDB")
	}
}

func TestNotConfiguredStorage(t *testing.T) {
	storageController := YDBandFileCacheController{}
	storageController.tasksCache = nil
	storageController.tasksDB = nil
	storageController.usersDB = nil
	if storageController.UnsubscribeUser(3435) != ErrNoActiveUsersStorage {
		t.Errorf("UnsubscribeUser should return ErrNoActiveUsersStorage when users storage isn't set")
	}
	if storageController.SubscribeUser(common.User{}) != ErrNoActiveUsersStorage {
		t.Errorf("SubscribeUser should return ErrNoActiveUsersStorage when users storage isn't set")
	}
	if _, err := storageController.GetSubscribedUsers(); err != ErrNoActiveUsersStorage {
		t.Errorf("GetSubscribedUsers should return ErrNoActiveUsersStorage when users storage isn't set")
	}
	err := storageController.SaveTask(common.BotLeetCodeTask{})
	if err != nil {
		t.Errorf("If task storage configured, SaveTask shouln't return error")
	}
	_, err = storageController.GetTask(12312)
	if err != ErrNoSuchTask {
		t.Errorf("If task storage configured, ErrNoSuchTask error should be returned")
	}
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
	if err != nil {
		t.Errorf("No error awaited on existing task from cache, but got %q", err)
	}
	if !reflect.DeepEqual(task, cacheStorage.tasks[12345]) {
		t.Errorf("Returned task not equal to what we are waited from cache:\n%q\ngot:\n%q\n", cacheStorage.tasks[12345], task)
	}
	if len(DBStorage.callsJournal) != 0 {
		t.Errorf("Error: database called %d times, when task persists in cache", len(DBStorage.callsJournal))
	}
	awaitedCallsJournal := []string{"getTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("Cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

}

func TestGetTaskFromDBMissedInCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	// Get task from DB and check that it will be saved in cache
	task, err := storageController.GetTask(12346)
	if err != nil {
		t.Errorf("No error awaited on existing task from db, but got %q", err)
	}
	if !reflect.DeepEqual(task, DBStorage.tasks[12346]) {
		t.Errorf("Returned task not equal to what we are waited from cache:\n%q\ngot:\n%q\n", cacheStorage.tasks[12346], task)
	}
	if cacheTask, ok := cacheStorage.tasks[12346]; ok {
		if !reflect.DeepEqual(task, cacheTask) {
			t.Errorf("Returned task not equal to what we saved to cache:\n%q\ngot in cache:\n%q\n", task, cacheTask)
		}
	} else {
		t.Errorf("Returned task is not saved to cache")
	}
	awaitedCallsJournal := []string{"getTask 12346"}
	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("Cache calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
	awaitedCallsJournal = []string{"getTask 12346", "saveTask 12346"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("Cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}
}

func TestGetTaskFromDBWithBrokenCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()

	// This ID persists in cache and in DB. Now we should get it from db and not save to the cache
	cacheStorage.IDToFail = 12345
	originalCacheTask := cacheStorage.tasks[12345]
	task, err := storageController.GetTask(12345)
	if err != nil {
		t.Errorf("No error awaited on existing task from db, but got %q", err)
	}
	if !reflect.DeepEqual(task, DBStorage.tasks[12345]) {
		t.Errorf("Returned task not equal to what we are waited from cache:\n%q\ngot:\n%q\n", cacheStorage.tasks[12346], task)
	}
	if !reflect.DeepEqual(originalCacheTask, cacheStorage.tasks[12345]) {
		t.Errorf("Task in cache were updated. Was:\n%q\nbecome:\n%q\n", originalCacheTask, cacheStorage.tasks[12346])
	}
	awaitedCallsJournal := []string{"getTask 12345"}
	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
	awaitedCallsJournal = []string{"getTask 12345", "saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("Cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}
}

func TestSaveTaskToDBAndCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	err := storageController.SaveTask(taskToSave)
	if err != nil {
		t.Errorf("No error awaited on saving task from db and cache, but got %q", err)
	}
	if task, ok := cacheStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in cache task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in cache after SaveTask")
	}

	if task, ok := DBStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in DB task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in DB after SaveTask")
	}

	awaitedCallsJournal := []string{"saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
}

func TestSaveTaskWithBrokenCache(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	cacheStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)
	if err != nil {
		t.Errorf("No error awaited on saving task to db and cache, but got %q", err)
	}
	if _, ok := cacheStorage.tasks[12345]; ok {
		t.Errorf("Task saved to broken cache?!")
	}

	if task, ok := DBStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in DB task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in DB after SaveTask")
	}

	awaitedCallsJournal := []string{"saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
}

func TestSaveTaskWithBrokenDB(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	DBStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)

	if err != ErrBypassTest {
		t.Errorf("ErrBypassTest awaited on saving task with broken DB, but got %q", err)
	}

	if _, ok := DBStorage.tasks[12345]; ok {
		t.Errorf("Task saved to broken DB?!")
	}

	if task, ok := cacheStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in cache task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in cache after SaveTask")
	}

	awaitedCallsJournal := []string{"saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
}

func TestSaveTaskWithBrokenAll(t *testing.T) {
	storageController, cacheStorage, DBStorage := getTestController()
	taskToSave := cacheStorage.tasks[12345]
	delete(cacheStorage.tasks, 12345)
	delete(DBStorage.tasks, 12345)
	DBStorage.IDToFail = 12345
	cacheStorage.IDToFail = 12345
	err := storageController.SaveTask(taskToSave)

	if err != ErrBypassTest {
		t.Errorf("ErrBypassTest awaited on saving task with broken all, but got %q", err)
	}

	if _, ok := DBStorage.tasks[12345]; ok {
		t.Errorf("Task saved to broken DB?!")
	}

	if _, ok := cacheStorage.tasks[12345]; ok {
		t.Errorf("Task saved to broken cache?!")
	}

	awaitedCallsJournal := []string{"saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
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
	if err != nil {
		t.Errorf("No error awaited on saving task from db and cache, but got %q", err)
	}
	if task, ok := cacheStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in cache task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in cache after SaveTask")
	}

	if task, ok := DBStorage.tasks[12345]; ok {
		if !reflect.DeepEqual(task, taskToSave) {
			t.Errorf("Saved in DB task is not equal with sent:\n%q\nsend:\n%q\n", task, taskToSave)
		}
	} else {
		t.Errorf("Task missed in DB after SaveTask")
	}

	awaitedCallsJournal := []string{"saveTask 12345"}
	if !reflect.DeepEqual(cacheStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("cache calls journal not equal with awaited. Got: %q, waited: %q", cacheStorage.callsJournal, awaitedCallsJournal)
	}

	if !reflect.DeepEqual(DBStorage.callsJournal, awaitedCallsJournal) {
		t.Errorf("DB calls journal not equal with awaited. Got: %q, waited: %q", DBStorage.callsJournal, awaitedCallsJournal)
	}
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
		if err != nil {
			t.Errorf("Error returned on GetSubscribedUsers: %q", err)
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].ID < list[j].ID
		})
		sort.Slice(awaitedList, func(i, j int) bool {
			return awaitedList[i].ID < awaitedList[j].ID
		})
		if !reflect.DeepEqual(list, awaitedList) {
			t.Errorf("Awaited following list of users\n%v\nbut\n%v\nreturned", awaitedList, list)
		}
		callList := []string{"getSubscribedUsers"}
		if !reflect.DeepEqual(usersStore.callsJournal, callList) {
			t.Errorf("Call list is wrong:\n%v\nawaited:\n%v\n", usersStore.callsJournal, callList)
		}
	}
}

func TestGetSubscribedUsersWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.getSubscribedUsersMustFail = true
	list, err := storageController.GetSubscribedUsers()
	if err != ErrBypassTest {
		t.Errorf("ErrBypassTest error wawited from GetSubscribedUsers but %q returned", err)
	}
	if !reflect.DeepEqual(list, []common.User{}) {
		t.Errorf("On error empty list should be returned but got following:\n%v\n", list)
	}
	callList := []string{"getSubscribedUsers"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Call list is wrong:\n%v\nawaited:\n%v\n", usersStore.callsJournal, callList)
	}
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
	if err != nil {
		t.Errorf("Error returned on SubscribeUser: %q", err)
	}
	newUser.Subscribed = true
	if !reflect.DeepEqual(*usersStore.users[1000], newUser) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1000], newUser)
	}
	callList := []string{"getUser 1000", "saveUser 1000"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestSubscribeUserOld(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1124]
	err := storageController.SubscribeUser(user)
	if err != nil {
		t.Errorf("Error returned on SubscribeUser: %q", err)
	}
	user.Subscribed = true
	if !reflect.DeepEqual(*usersStore.users[1124], user) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1124], user)
	}
	callList := []string{"getUser 1124", "subscribeUser 1124"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestSubscribeUserAlreadySubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.users[1124].Subscribed = true
	userToSend := *usersStore.users[1124]
	err := storageController.SubscribeUser(userToSend)
	if err != ErrUserAlreadySubscribed {
		t.Errorf("ErrUserAlreadySubscribed error awaited on SubscribeUser but returned: %q", err)
	}

	if !reflect.DeepEqual(*usersStore.users[1124], userToSend) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1124], userToSend)
	}
	callList := []string{"getUser 1124"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestSubscribeUserWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.IDToFail = 1124
	userToSend := *usersStore.users[1124]
	err := storageController.SubscribeUser(userToSend)
	if err != ErrBypassTest {
		t.Errorf("ErrBypassTest error awaited on SubscribeUser but returned: %q", err)
	}

	if !reflect.DeepEqual(*usersStore.users[1124], userToSend) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1124], userToSend)
	}

	callList := []string{"getUser 1124"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestUnsubscribeUserNew(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	err := storageController.UnsubscribeUser(1000)
	if err != ErrUserAlreadyUnsubscribed {
		t.Errorf("Error returned on UnsubscribeUser: %q", err)
	}
	callList := []string{"getUser 1000"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestUnsubscribeUserOldNotSubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1124]
	err := storageController.UnsubscribeUser(1124)
	if err != ErrUserAlreadyUnsubscribed {
		t.Errorf("Error returned on UnsubscribeUser: %q", err)
	}

	if !reflect.DeepEqual(*usersStore.users[1124], user) {
		t.Errorf("Stored user changed on unsubscring attemp:\n%v\nbut before was:\n%v\n", *usersStore.users[1124], user)
	}
	callList := []string{"getUser 1124"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestUnsubscribeUserAlreadySubscribed(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	user := *usersStore.users[1126]
	err := storageController.UnsubscribeUser(1126)
	if err != nil {
		t.Errorf("Error returned on SubscribeUser: %q", err)
	}
	user.Subscribed = false
	if !reflect.DeepEqual(*usersStore.users[1126], user) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1126], user)
	}
	callList := []string{"getUser 1126", "unsubscribeUser 1126"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}

func TestUnsubscribeUserWithError(t *testing.T) {
	usersStore := getTestUsersStorekeeper()
	storageController := YDBandFileCacheController{
		usersDB: usersStore,
	}
	usersStore.IDToFail = 1124
	userToSend := *usersStore.users[1124]
	err := storageController.UnsubscribeUser(1124)
	if err != ErrBypassTest {
		t.Errorf("ErrBypassTest error awaited on SubscribeUser but returned: %q", err)
	}

	if !reflect.DeepEqual(*usersStore.users[1124], userToSend) {
		t.Errorf("Stored user differ with sent one:\n%v\nsent one:\n%v\n", *usersStore.users[1124], userToSend)
	}

	callList := []string{"getUser 1124"}
	if !reflect.DeepEqual(usersStore.callsJournal, callList) {
		t.Errorf("Wrong call list:\n%v\nbut should be:\n%v\n", usersStore.callsJournal, callList)
	}
}
