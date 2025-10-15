package services

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials is returned when username/password is incorrect
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserAlreadyExists is returned when trying to create a user that already exists
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidUsername is returned when username doesn't meet requirements
	ErrInvalidUsername = errors.New("username must be 3-32 characters, alphanumeric and underscores only")
	// ErrInvalidPassword is returned when password doesn't meet requirements
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
)

// UserService handles user operations
type UserService struct {
	db *gorm.DB
}

// NewUserService creates a new user service
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// ValidateUsername checks if username meets requirements
func (s *UserService) ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return ErrInvalidUsername
	}

	// Only allow alphanumeric and underscores
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
	if !matched {
		return ErrInvalidUsername
	}

	return nil
}

// ValidatePassword checks if password meets requirements
func (s *UserService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	return nil
}

// CreateUser creates a new user with the given username and password
func (s *UserService) CreateUser(username, password string, isAdmin bool) (*models.User, error) {
	// Validate username
	if err := s.ValidateUsername(username); err != nil {
		return nil, err
	}

	// Validate password
	if err := s.ValidatePassword(password); err != nil {
		return nil, err
	}

	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}

	// Create new user
	user := &models.User{
		Username: username,
		IsAdmin:  isAdmin,
	}

	// Hash password
	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Save to database
	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// AuthenticateUser verifies username and password, returns user if valid
func (s *UserService) AuthenticateUser(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check password
	if !user.CheckPassword(password) {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// UpdatePassword updates a user's password after verifying the old password
func (s *UserService) UpdatePassword(userID uuid.UUID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !user.CheckPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	// Validate new password
	if err := s.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Save to database
	if err := s.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ListUsers returns all users (admin only)
func (s *UserService) ListUsers() ([]models.User, error) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(userID uuid.UUID) error {
	result := s.db.Delete(&models.User{}, "id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CountUsers returns the total number of users
func (s *UserService) CountUsers() (int64, error) {
	var count int64
	if err := s.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}
