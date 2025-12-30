package domain

import (
	"context"
	"time"
)

// User represents a user entity in the system.
// A user can register, login, and perform actions like writing articles.
type User struct {
	ID        int64     // Unique identifier
	Name      string    // Display name
	Username  string    // Login username (unique)
	Password  string    // Bcrypt hashed password
	CreatedAt time.Time // Account creation timestamp
	UpdatedAt time.Time // Last profile update timestamp
}

// UserRepository defines the contract for user data persistence.
type UserRepository interface {
	// GetByID retrieves a user by their ID.
	// Returns ErrNotFound if the user doesn't exist.
	GetByID(ctx context.Context, id int64) (User, error)

	// Insert creates a new user account.
	// Backfills the ID in the provided User object upon success.
	Insert(ctx context.Context, u *User) error

	// Update modifies an existing user's information.
	Update(ctx context.Context, u *User) error

	// GetByUsername retrieves a user by their username.
	// Used during login to verify credentials.
	GetByUsername(ctx context.Context, username string) (User, error)

	GetByIDs(ctx context.Context, userIDs []int64) ([]User, error)
}

// UserUsecase defines the business logic contract for user operations.
// Handles authentication, registration, and user management.
type UserUsecase interface {
	// Register creates a new user account.
	// Returns ErrConflict if the username already exists.
	Register(ctx context.Context, name, username, password string) error

	// Login verifies user credentials and returns a JWT token.
	// Returns ErrNotFound if the user doesn't exist.
	// Returns ErrBadParamInput if the password is incorrect.
	Login(ctx context.Context, username, password string) (string, error)

	// EditPassword verifies user credentials and change the password by given new password
	EditPassword(ctx context.Context, id int64, oldPassword, newPassword string) error
}
