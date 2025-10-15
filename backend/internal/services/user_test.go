package services

import (
	"testing"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestCreateUser(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	user, err := service.CreateUser("testuser", "password123", false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.PasswordHash == "" {
		t.Error("Password hash should not be empty")
	}

	if user.PasswordHash == "password123" {
		t.Error("Password should be hashed, not stored in plain text")
	}

	if user.IsAdmin {
		t.Error("User should not be admin by default")
	}
}

func TestCreateUserValidation(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "username too short",
			username: "ab",
			password: "password123",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "username too long",
			username: "abcdefghijklmnopqrstuvwxyz1234567",
			password: "password123",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "username with special characters",
			username: "user@test",
			password: "password123",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "password too short",
			username: "testuser",
			password: "pass",
			wantErr:  ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateUser(tt.username, tt.password, false)
			if err != tt.wantErr {
				t.Errorf("Expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAuthenticateUser(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	// Create a test user
	_, err := service.CreateUser("testuser", "password123", false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test successful authentication
	user, err := service.AuthenticateUser("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to authenticate user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	// Test failed authentication with wrong password
	_, err = service.AuthenticateUser("testuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}

	// Test failed authentication with non-existent user
	_, err = service.AuthenticateUser("nonexistent", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	// Create a test user
	user, err := service.CreateUser("testuser", "oldpassword", false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update password
	err = service.UpdatePassword(user.ID, "oldpassword", "newpassword123")
	if err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	// Verify new password works
	_, err = service.AuthenticateUser("testuser", "newpassword123")
	if err != nil {
		t.Errorf("Failed to authenticate with new password: %v", err)
	}

	// Verify old password doesn't work
	_, err = service.AuthenticateUser("testuser", "oldpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Old password should not work, got error: %v", err)
	}

	// Test with wrong old password
	err = service.UpdatePassword(user.ID, "wrongoldpassword", "anotherpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials when old password is wrong, got %v", err)
	}
}

func TestUserAlreadyExists(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	// Create first user
	_, err := service.CreateUser("testuser", "password123", false)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create user with same username
	_, err = service.CreateUser("testuser", "password456", false)
	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestCountUsers(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	// Initially should be 0
	count, err := service.CountUsers()
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 users, got %d", count)
	}

	// Create users
	service.CreateUser("user1", "password123", false)
	service.CreateUser("user2", "password123", false)

	count, err = service.CountUsers()
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 users, got %d", count)
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupUserTestDB(t)
	service := NewUserService(db)

	// Create a user
	user, err := service.CreateUser("testuser", "password123", false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete the user
	err = service.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = service.GetUserByUsername("testuser")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
