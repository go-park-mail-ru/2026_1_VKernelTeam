// Package storage предоставляет простое хранилище данных в памяти с
// периодическим дампом в JSON-файл. Он реализует интерфейсы, используемые
// сервисом аутентификации, такие как UserSaver, UserProvider и AppProvider.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
)

// ErrUserExists возвращается, когда пытаются создать пользователя с email, который уже существует в хранилище.
// ErrAppExists возвращается, когда пытаются создать приложение с именем, которое уже существует в хранилище.
// ErrUserNotFound возвращается, когда запрашиваемый пользователь не найден в хранилище.
// ErrAppNotFound возвращается, когда запрашиваемое приложение не найдено в хранилище.
var (
	ErrUserExists   = errors.New("user already exists")
	ErrAppExists    = errors.New("app already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrAppNotFound  = errors.New("app not found")
)

// Storage реализует все интерфейсы, необходимые сервису auth.
// Он хранит данные в памяти и периодически (при каждом изменении) записывает
// снимок в файл, указанный вызывающей стороной. Доступ защищен
// мьютексом чтения-записи (RWMutex), поэтому параллельные читатели могут работать одновременно, в то время как
// писатели получают эксклюзивный доступ.
type Storage struct {
	mu sync.RWMutex // защищает поля ниже

	usersByEmail map[string]models.User
	usersByID    map[int64]models.User
	apps         map[int64]models.App

	nextUserID int64
	nextAppID  int64

	path string // путь к файлу дампа JSON
}

// dump - это структура, которая сохраняется на диск. Мы храним ее отдельно, чтобы
// политика блокировок внутри Storage не была видна снаружи.

type dump struct {
	Users      map[string]models.User `json:"users"`
	UsersByID  map[int64]models.User  `json:"users_by_id"`
	Apps       map[int64]models.App   `json:"apps"`
	NextUserID int64                  `json:"next_user_id"`
	NextAppID  int64                  `json:"next_app_id"`
}

// New создает экземпляр Storage и, если файл уже существует,
// предварительно заполняет его содержимым файла. Если файл отсутствует,
// хранилище создается пустым. Путь может быть пустым; в этом случае сохранение
// пропускается, и хранилище остается только в памяти.
func New(path string) (*Storage, error) {
	s := &Storage{
		usersByEmail: make(map[string]models.User),
		usersByID:    make(map[int64]models.User),
		apps:         make(map[int64]models.App),
		path:         path,
	}

	if path == "" {
		return s, nil
	}

	// убеждаемся, что директория существует
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		if err := s.load(); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

func (s *Storage) load() error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	var d dump
	if err := json.NewDecoder(f).Decode(&d); err != nil {
		// пустой файл - это не ошибка, относимся к этому как к новому (пустому) хранилищу
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	s.usersByEmail = d.Users
	s.usersByID = d.UsersByID
	s.apps = d.Apps
	s.nextUserID = d.NextUserID
	s.nextAppID = d.NextAppID

	return nil
}

func (s *Storage) persist() error {
	if s.path == "" {
		return nil
	}

	d := dump{
		Users:      s.usersByEmail,
		UsersByID:  s.usersByID,
		Apps:       s.apps,
		NextUserID: s.nextUserID,
		NextAppID:  s.nextAppID,
	}

	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(&d)
}

// SaveUser удовлетворяет интерфейсу auth.UserSaver. Он возвращает ErrUserExists, если пользователь с
// таким же email уже существует.
func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.usersByEmail[email]; ok {
		return 0, ErrUserExists
	}

	s.nextUserID++
	u := models.User{ID: s.nextUserID, Email: email, PassHash: passHash}
	s.usersByEmail[email] = u
	s.usersByID[u.ID] = u

	if err := s.persist(); err != nil {
		return 0, err
	}

	return u.ID, nil
}

// User реализует интерфейс auth.UserProvider.User.
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByEmail[email]
	if !ok {
		return models.User{}, ErrUserNotFound
	}
	return u, nil
}

// IsAdmin реализует интерфейс auth.UserProvider.IsAdmin. Мы ищем пользователя по ID во
// второй мапе для доступа за константное время.
func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByID[userID]
	if !ok {
		return false, ErrUserNotFound
	}
	return u.IsAdmin, nil
}

// CreateApp добавляет новое приложение и возвращает его автоматически сгенерированный ID. Если
// приложение с таким же именем уже существует, возвращается ErrAppExists.
func (s *Storage) CreateApp(ctx context.Context, name, secret string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// убеждаемся, что имя уникально
	for _, a := range s.apps {
		if a.Name == name {
			return 0, ErrAppExists
		}
	}

	s.nextAppID++
	a := models.App{ID: s.nextAppID, Name: name, Secret: secret}
	s.apps[a.ID] = a

	if err := s.persist(); err != nil {
		return 0, err
	}

	return a.ID, nil
}

// App реализует интерфейс auth.AppProvider.App.
func (s *Storage) App(ctx context.Context, appID int64) (models.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.apps[appID]
	if !ok {
		return models.App{}, ErrAppNotFound
	}
	return a, nil
}
