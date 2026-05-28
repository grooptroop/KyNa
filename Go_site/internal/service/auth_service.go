package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"

	"golang.org/x/crypto/bcrypt"

	"github.com/grooptroop/KyNa/Go_site/internal/model"

	"github.com/grooptroop/KyNa/Go_site/internal/repository"
)

type SessionStore struct {
	data map[string]string // sessionID -> username
}

func NewSessionStore() *SessionStore {
	return &SessionStore{data: make(map[string]string)}
}

func (s *SessionStore) Set(sessionID, username string) {
	s.data[sessionID] = username
}

func (s *SessionStore) Get(sessionID string) (string, bool) {
	u, ok := s.data[sessionID]
	return u, ok
}

func (s *SessionStore) Delete(sessionID string) {
	delete(s.data, sessionID)
}

type AuthService struct {
	accounts *repository.AccountRepository
	sessions *SessionStore
	users    *UserService
}

func NewAuthService(
	accounts *repository.AccountRepository,
	sessions *SessionStore,
	users *UserService,
) *AuthService {
	return &AuthService{
		accounts: accounts,
		sessions: sessions,
		users:    users,
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.Account, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	acc := &model.Account{
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: string(hashed),
		Role:         model.RoleUser,
	}

	if err := s.accounts.Create(ctx, acc); err != nil {
		return nil, err
	}

	// Автоматически создаём user_provisions для этого аккаунта,
	// чтобы FK user_machines.username → user_provisions.username не падал.
	if s.users != nil {
		_, err := s.users.CreateUser(ctx, CreateUserInput{
			Username: acc.Username,
			Domain:   fmt.Sprintf("%s.example.local", acc.Username),
			Mode:     "app",
		})
		if err != nil {
			log.Printf("failed to create user_provisions for %s: %v", acc.Username, err)
			// регистрацию не валим
		}
	}

	// Создаём namespace для пользователя (один раз при регистрации).
	if err := createUserNamespace(acc.Username); err != nil {
		log.Printf("failed to create namespace for %s: %v", acc.Username, err)
		// регистрацию тоже не валим
	}

	return acc, nil
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*model.Account, string, error) {
	acc, err := s.accounts.FindByUsername(ctx, in.Username)
	if err != nil {
		return nil, "", fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(in.Password)); err != nil {
		return nil, "", fmt.Errorf("invalid username or password")
	}

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, "", err
	}

	s.sessions.Set(sessionID, acc.Username)
	return acc, sessionID, nil
}

func (s *AuthService) Logout(sessionID string) {
	s.sessions.Delete(sessionID)
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// вспомогательная функция создания namespace
func createUserNamespace(username string) error {
	cmd := exec.Command(
		"kubectl",
		"create",
		"namespace",
		username,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Если namespace уже существует, это не критично
		log.Printf("kubectl create namespace %s failed: %v, output: %s", username, err, string(out))
		return nil
	}

	return nil
}
