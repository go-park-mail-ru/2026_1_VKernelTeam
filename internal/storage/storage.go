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
// ErrUserNotFound возвращается, когда запрашиваемый пользователь не найден в хранилище.
var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
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

	nextUserID int64

	path string // путь к файлу дампа JSON
}

// dump - это структура, которая сохраняется на диск. Мы храним ее отдельно, чтобы
// политика блокировок внутри Storage не была видна снаружи.

type dump struct {
	Users      map[string]models.User `json:"users"`
	UsersByID  map[int64]models.User  `json:"users_by_id"`
	NextUserID int64                  `json:"next_user_id"`
}

// New создает экземпляр Storage и, если файл уже существует,
// предварительно заполняет его содержимым файла. Если файл отсутствует,
// хранилище создается пустым. Путь может быть пустым; в этом случае сохранение
// пропускается, и хранилище остается только в памяти.
func New(path string) (*Storage, error) {
	s := &Storage{
		usersByEmail: make(map[string]models.User),
		usersByID:    make(map[int64]models.User),
		path:         path,
	}

	for _, u := range generateMockUsers() {
		s.usersByEmail[u.Email] = u
		s.usersByID[u.ID] = u
		if u.ID > s.nextUserID {
			s.nextUserID = u.ID
		}
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

	if d.Users != nil {
		s.usersByEmail = d.Users
	}
	if d.UsersByID != nil {
		s.usersByID = d.UsersByID
	}
	s.nextUserID = d.NextUserID

	return nil
}

func (s *Storage) persist() error {
	if s.path == "" {
		return nil
	}

	d := dump{
		Users:      s.usersByEmail,
		UsersByID:  s.usersByID,
		NextUserID: s.nextUserID,
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

	// Если такой email уже есть, возвращаем ошибку
	if _, exists := s.usersByEmail[email]; exists {
		return 0, ErrUserExists
	}

	s.nextUserID++
	u := models.User{ID: s.nextUserID, Email: email, PassHash: passHash}
	s.usersByEmail[email] = u
	s.usersByID[u.ID] = u

	if err := s.persist(); err != nil {
		delete(s.usersByEmail, email)
		delete(s.usersByID, u.ID)
		s.nextUserID--
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

// generateMockUsers returns a slice of mock users (profiles) suitable for testing.
// It includes various roles, empty emails, and different data variations.
func generateMockUsers() []models.User {
	return []models.User{
		{
			ID:       1,
			Email:    "admin@vk.com",
			PassHash: []byte("hashed_password_1"),
			IsAdmin:  true,
		},
		{
			ID:       2,
			Email:    "ivan.ivanov@mail.ru",
			PassHash: []byte("hashed_password_2"),
			IsAdmin:  false,
		},
		{
			ID:       3,
			Email:    "petr.petrov@yandex.ru",
			PassHash: []byte("hashed_password_3"),
			IsAdmin:  false,
		},
		{
			ID:       4,
			Email:    "anna.smith@gmail.com",
			PassHash: []byte("hashed_password_4"),
			IsAdmin:  false,
		},
		{
			ID:       5,
			Email:    "weird.user+test@domain.co.uk",
			PassHash: []byte("hashed_password_5"),
			IsAdmin:  false,
		},
		{
			ID:       6,
			Email:    "errtytyu@gmail.com",
			PassHash: []byte("hashed_password_6"),
			IsAdmin:  false,
		},
		{
			ID:       7,
			Email:    "moderator@local.host",
			PassHash: []byte("hashed_password_7"),
			IsAdmin:  true,
		},
	}
}
