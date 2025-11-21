package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"helios-auth-service/internal/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type mockUserRepository struct {
	users map[string]*models.User // email -> user
	mu    sync.RWMutex
}

func NewMockUserRepository() UserRepository {
	return &mockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (r *mockUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[user.Email]; exists {
		return errors.New("user with this email already exists")
	}
	r.users[user.Email] = user
	return nil
}

func (r *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *mockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}
