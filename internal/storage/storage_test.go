// Тесты для пакета storage проверяют поведение простого
// in-memory хранилища и его сериализации в файл.
package storage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// TestSaveAndLoad проверяет сохранение пользователя, затем перезагрузку
// хранилища с диска и корректное восстановление данных.
func TestSaveAndLoad(t *testing.T) {
	path := "test_dump.json"
	defer os.Remove(path)

	st, err := New(path)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	uid, err := st.SaveUser(context.Background(), "foo@example.com", []byte("hash"), "Foo User")
	if err != nil {
		t.Fatalf("save user: %v", err)
	}

	if uid == 0 {
		t.Fatalf("expected non-zero uid")
	}

	// reopen storage from disk
	st2, err := New(path)
	if err != nil {
		t.Fatalf("failed to reload storage: %v", err)
	}

	u, err := st2.User(context.Background(), "foo@example.com")
	if err != nil {
		t.Fatalf("retrieve user after reload: %v", err)
	}
	if u.ID != uid {
		t.Fatalf("id mismatch: got %d want %d", u.ID, uid)
	}
}

// TestConcurrency проверяет потокобезопасность SaveUser при параллельных
// вызовах и то, что все пользователи были успешно сохранены.
func TestConcurrency(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", i)
			name := fmt.Sprintf("User %d", i)
			if _, err := st.SaveUser(context.Background(), email, []byte("h"), name); err != nil {
				t.Errorf("save %s: %v", email, err)
			}
		}(i)
	}
	wg.Wait()

	// verify all users exist
	for i := 0; i < n; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		if _, err := st.User(context.Background(), email); err != nil {
			t.Errorf("missing user %s: %v", email, err)
		}
	}
}

func TestIsAdmin(t *testing.T) {
	st, _ := New("")
	st.usersByID[1] = models.User{ID: 1, Name: "Admin", Email: "admin@test.com", IsAdmin: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	st.usersByID[2] = models.User{ID: 2, Name: "User", Email: "user@test.com", IsAdmin: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	admin, _ := st.IsAdmin(context.Background(), 1)
	if !admin {
		t.Errorf("expected user 1 to be admin")
	}

	admin, _ = st.IsAdmin(context.Background(), 2)
	if admin {
		t.Errorf("expected user 2 not to be admin")
	}

	_, err := st.IsAdmin(context.Background(), 999)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound")
	}
}

func TestNew_EmptyFile(t *testing.T) {
	path := "test_empty_dump.json"
	os.WriteFile(path, []byte(""), 0644)
	defer os.Remove(path)

	_, err := New(path)
	if err != nil {
		t.Errorf("empty file should be handled as new storage")
	}
}

func TestNew_DirCreation(t *testing.T) {
	path := "testdir_new/dump.json"
	defer os.RemoveAll("testdir_new")

	_, err := New(path)
	if err != nil {
		t.Errorf("dir creation failed: %v", err)
	}
}

func TestSaveUser_PersistError(t *testing.T) {
	st, _ := New("")
	st.path = "/invalid_dir/invalid_1234/storage.json"
	_, err := st.SaveUser(context.Background(), "test@test.com", []byte("hash"), "Test User")
	if err == nil {
		t.Errorf("expected err due to invalid persist path")
	}
}
