package domain

import (
	"context"

	"github.com/oklog/ulid/v2"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user in the database
	Create(ctx context.Context, user *User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id ulid.ULID) (*User, error)

	// FindByEmail finds a user by email
	FindByEmail(ctx context.Context, email string) (*User, error)

	// ExistsByEmail checks if a user exists with the given email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// Update updates a user
	Update(ctx context.Context, user *User) error

	// UpdatePassword updates a user's password
	UpdatePassword(ctx context.Context, userID ulid.ULID, hashedPassword string) error

	// Delete deletes a user
	Delete(ctx context.Context, id ulid.ULID) error

	// List lists all users with pagination
	List(ctx context.Context, limit, offset int) ([]*User, error)

	// AddRole adds a role to a user
	AddRole(ctx context.Context, userID ulid.ULID, role string) error

	// RemoveRole removes a role from a user
	RemoveRole(ctx context.Context, userID ulid.ULID, role string) error
}
