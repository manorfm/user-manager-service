package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/manorfm/authM/internal/domain"
	"github.com/manorfm/authM/internal/infrastructure/config"
	"go.uber.org/zap"
)

// compositeStrategy implements JWTStrategy with fallback support
type compositeStrategy struct {
	vaultStrategy domain.JWTStrategy
	localStrategy domain.JWTStrategy
	logger        *zap.Logger
	useVault      bool
	mu            sync.RWMutex
}

// NewCompositeStrategy creates a new composite strategy with fallback support
func NewCompositeStrategy(cfg *config.Config, logger *zap.Logger) domain.JWTStrategy {
	// Create JWT configuration

	var vaultStrategy domain.JWTStrategy
	var err error

	if cfg.EnableVault {
		vaultStrategy, err = NewVaultStrategy(cfg, logger)
		if err != nil {
			logger.Warn("Failed to create Vault strategy, falling back to local strategy",
				zap.Error(err))
		}
	}

	// Create local strategy
	localStrategy, err := NewLocalStrategy(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create local strategy", zap.Error(err))
	}

	return &compositeStrategy{
		vaultStrategy: vaultStrategy,
		localStrategy: localStrategy,
		logger:        logger,
		useVault:      cfg.EnableVault,
	}
}

// Sign signs a JWT token using the current strategy with fallback
func (c *compositeStrategy) Sign(claims *domain.Claims) (string, error) {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		token, err := c.vaultStrategy.Sign(claims)
		if err == nil {
			c.logger.Debug("Signed token with Vault", zap.String("token", token))
			return token, nil
		}
		c.logger.Warn("Failed to sign token with Vault, falling back to local strategy",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)))

		// Only fallback for specific errors
		c.mu.Lock()
		c.useVault = false
		c.mu.Unlock()
	}
	return c.localStrategy.Sign(claims)
}

// GetPublicKey returns the public key from the current strategy
func (c *compositeStrategy) GetPublicKey() *rsa.PublicKey {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		publicKey := c.vaultStrategy.GetPublicKey()
		if publicKey != nil {
			return publicKey
		}
		c.logger.Warn("Failed to get public key from Vault, falling back to local strategy")
		c.mu.Lock()
		c.useVault = false
		c.mu.Unlock()
	}
	return c.localStrategy.GetPublicKey()
}

// GetKeyID returns the current key ID
func (c *compositeStrategy) GetKeyID() string {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		c.logger.Info("Key ID from Vault", zap.String("key_id", c.vaultStrategy.GetKeyID()))
		return c.vaultStrategy.GetKeyID()
	}
	return c.localStrategy.GetKeyID()
}

// RotateKey rotates the key in the current strategy
func (c *compositeStrategy) RotateKey() error {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		err := c.vaultStrategy.RotateKey()
		if err == nil {
			c.logger.Info("Rotated key in Vault")
			return nil
		}
		c.logger.Warn("Failed to rotate key in Vault, falling back to local strategy", zap.Error(err))
		c.mu.Lock()
		c.useVault = false
		c.mu.Unlock()
	}
	return c.localStrategy.RotateKey()
}

// GetLastRotation returns the last key rotation time
func (c *compositeStrategy) GetLastRotation() time.Time {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		return c.vaultStrategy.GetLastRotation()
	}
	return c.localStrategy.GetLastRotation()
}

// TryVault attempts to switch back to the Vault strategy
func (c *compositeStrategy) TryVault() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.vaultStrategy == nil {
		return domain.ErrInvalidClient
	}

	// Test Vault connection by getting public key
	publicKey := c.vaultStrategy.GetPublicKey()
	if publicKey == nil {
		return domain.ErrInvalidClient
	}

	c.useVault = true
	c.logger.Info("Successfully switched back to Vault strategy")
	return nil
}

// Verify verifies a JWT token using the current strategy with fallback
func (c *compositeStrategy) Verify(tokenString string) (*domain.Claims, error) {
	c.mu.RLock()
	useVault := c.useVault && c.vaultStrategy != nil
	c.mu.RUnlock()

	if useVault {
		claims, err := c.vaultStrategy.Verify(tokenString)
		if err == nil {
			c.logger.Info("Verified token with Vault")
			return claims, nil
		}
		c.logger.Warn("Failed to verify token with Vault, falling back to local strategy",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)))

		// Only fallback for specific errors
		if errors.Is(err, domain.ErrInvalidClient) || errors.Is(err, domain.ErrInvalidKeyConfig) {
			c.mu.Lock()
			c.useVault = false
			c.mu.Unlock()
		} else {
			return nil, err // Return other errors without fallback
		}
	}

	if c.localStrategy == nil {
		return nil, domain.ErrInvalidClient
	}

	return c.localStrategy.Verify(tokenString)
}
