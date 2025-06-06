package application

import (
	"context"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/ipede/user-manager-service/internal/domain"
	"github.com/ipede/user-manager-service/internal/infrastructure/password"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id ulid.ULID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) AddRole(ctx context.Context, userID ulid.ULID, role string) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserRepository) RemoveRole(ctx context.Context, userID ulid.ULID, role string) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id ulid.ULID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, userID ulid.ULID, hashedPassword string) error {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Error(0)
}

type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, email, code string) error {
	args := m.Called(ctx, email, code)
	return args.Error(0)
}

func (m *mockEmailService) SendPasswordResetEmail(ctx context.Context, email, code string) error {
	args := m.Called(ctx, email, code)
	return args.Error(0)
}

type mockJWTService struct {
	mock.Mock
}

func (m *mockJWTService) GenerateTokenPair(userID ulid.ULID, roles []string) (*domain.TokenPair, error) {
	args := m.Called(userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *mockJWTService) ValidateToken(token string) (*domain.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Claims), args.Error(1)
}

func (m *mockJWTService) GetJWKS(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockJWTService) GetPublicKey() *rsa.PublicKey {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*rsa.PublicKey)
}

func (m *mockJWTService) RotateKeys() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockJWTService) BlacklistToken(tokenID string, expiresAt time.Time) error {
	args := m.Called(tokenID, expiresAt)
	return args.Error(0)
}

func (m *mockJWTService) IsTokenBlacklisted(tokenID string) bool {
	args := m.Called(tokenID)
	return args.Bool(0)
}

func (m *mockJWTService) TryVault() error {
	args := m.Called()
	return args.Error(0)
}

type mockVerificationCodeRepository struct {
	mock.Mock
}

func (m *mockVerificationCodeRepository) Create(ctx context.Context, code *domain.VerificationCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *mockVerificationCodeRepository) FindByCode(ctx context.Context, code string) (*domain.VerificationCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VerificationCode), args.Error(1)
}

func (m *mockVerificationCodeRepository) FindByUserIDAndType(ctx context.Context, userID ulid.ULID, codeType domain.VerificationCodeType) (*domain.VerificationCode, error) {
	args := m.Called(ctx, userID, codeType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VerificationCode), args.Error(1)
}

func (m *mockVerificationCodeRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *mockVerificationCodeRepository) DeleteByUserIDAndType(ctx context.Context, userID ulid.ULID, codeType domain.VerificationCodeType) error {
	args := m.Called(ctx, userID, codeType)
	return args.Error(0)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		setupMocks    func(*MockUserRepository, *mockVerificationCodeRepository, *mockEmailService)
		expectedError error
	}{
		{
			name:     "successful registration",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				m.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				m.On("Create", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
					return user.Email == "test@example.com" && !user.EmailVerified
				})).Return(nil)
				v.On("Create", mock.Anything, mock.AnythingOfType("*domain.VerificationCode")).Return(nil)
				e.On("SendVerificationEmail", mock.Anything, "test@example.com", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "user already exists",
			email:    "existing@example.com",
			password: "password123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				m.On("ExistsByEmail", mock.Anything, "existing@example.com").Return(true, nil)
			},
			expectedError: domain.ErrUserAlreadyExists,
		},
		{
			name:     "email send failed",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				m.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				m.On("Create", mock.Anything, mock.Anything).Return(nil)
				v.On("Create", mock.Anything, mock.AnythingOfType("*domain.VerificationCode")).Return(nil)
				e.On("SendVerificationEmail", mock.Anything, "test@example.com", mock.Anything).Return(domain.ErrEmailSendFailed)
			},
			expectedError: domain.ErrEmailSendFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockVerificationRepo := new(mockVerificationCodeRepository)
			mockEmailSvc := new(mockEmailService)
			tt.setupMocks(mockUserRepo, mockVerificationRepo, mockEmailSvc)

			service := NewAuthService(mockUserRepo, mockVerificationRepo, nil, mockEmailSvc, zap.NewNop())
			_, err := service.Register(context.Background(), "Test User", tt.email, tt.password, "1234567890")

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
			mockVerificationRepo.AssertExpectations(t)
			mockEmailSvc.AssertExpectations(t)
		})
	}
}

func TestAuthService_VerifyEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		code          string
		setupMocks    func(*MockUserRepository, *mockVerificationCodeRepository, *mockEmailService)
		expectedError error
	}{
		{
			name:  "successful verification",
			email: "test@example.com",
			code:  "123456",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					EmailVerified: false,
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				verificationCode := domain.NewVerificationCode(user.ID, "123456", domain.EmailVerification, 24*time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.EmailVerification).Return(verificationCode, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.EmailVerification).Return(nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
					return user.EmailVerified
				})).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			code:  "123456",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				m.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, domain.ErrUserNotFound)
			},
			expectedError: domain.ErrUserNotFound,
		},
		{
			name:  "invalid code",
			email: "test@example.com",
			code:  "wrong",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					EmailVerified: false,
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				verificationCode := domain.NewVerificationCode(user.ID, "123456", domain.EmailVerification, 24*time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.EmailVerification).Return(verificationCode, nil)
			},
			expectedError: domain.ErrInvalidVerificationCode,
		},
		{
			name:  "expired code",
			email: "test@example.com",
			code:  "123456",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					EmailVerified: false,
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				verificationCode := domain.NewVerificationCode(user.ID, "123456", domain.EmailVerification, -24*time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.EmailVerification).Return(verificationCode, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.EmailVerification).Return(nil)
				v.On("Create", mock.Anything, mock.AnythingOfType("*domain.VerificationCode")).Return(nil)
				e.On("SendVerificationEmail", mock.Anything, "test@example.com", mock.Anything).Return(nil)
			},
			expectedError: domain.ErrVerificationCodeExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockVerificationRepo := new(mockVerificationCodeRepository)
			mockEmailSvc := new(mockEmailService)
			tt.setupMocks(mockUserRepo, mockVerificationRepo, mockEmailSvc)

			service := NewAuthService(mockUserRepo, mockVerificationRepo, nil, mockEmailSvc, zap.NewNop())
			err := service.VerifyEmail(context.Background(), tt.email, tt.code)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
			mockVerificationRepo.AssertExpectations(t)
			mockEmailSvc.AssertExpectations(t)
		})
	}
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		setupMocks    func(*MockUserRepository, *mockVerificationCodeRepository, *mockEmailService)
		expectedError error
	}{
		{
			name:  "successful request",
			email: "test@example.com",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(nil)
				v.On("Create", mock.Anything, mock.AnythingOfType("*domain.VerificationCode")).Return(nil)
				e.On("SendPasswordResetEmail", mock.Anything, "test@example.com", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				m.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, domain.ErrUserNotFound)
			},
			expectedError: domain.ErrUserNotFound,
		},
		{
			name:  "delete existing codes failed",
			email: "test@example.com",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(domain.ErrInternal)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name:  "email send failed",
			email: "test@example.com",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository, e *mockEmailService) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(nil)
				v.On("Create", mock.Anything, mock.AnythingOfType("*domain.VerificationCode")).Return(nil)
				e.On("SendPasswordResetEmail", mock.Anything, "test@example.com", mock.Anything).Return(domain.ErrEmailSendFailed)
			},
			expectedError: domain.ErrEmailSendFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockVerificationRepo := new(mockVerificationCodeRepository)
			mockEmailSvc := new(mockEmailService)
			tt.setupMocks(mockUserRepo, mockVerificationRepo, mockEmailSvc)

			service := NewAuthService(mockUserRepo, mockVerificationRepo, nil, mockEmailSvc, zap.NewNop())
			err := service.RequestPasswordReset(context.Background(), tt.email)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
			mockVerificationRepo.AssertExpectations(t)
			mockEmailSvc.AssertExpectations(t)
		})
	}
}

func TestAuthService_ResetPassword(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		code          string
		newPassword   string
		setupMocks    func(*MockUserRepository, *mockVerificationCodeRepository)
		expectedError error
	}{
		{
			name:        "successful reset",
			email:       "test@example.com",
			code:        "123456",
			newPassword: "newpassword123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				resetCode := domain.NewVerificationCode(user.ID, "123456", domain.PasswordReset, time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(resetCode, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(nil)
				m.On("UpdatePassword", mock.Anything, user.ID, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:        "user not found",
			email:       "nonexistent@example.com",
			code:        "123456",
			newPassword: "newpassword123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository) {
				m.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, domain.ErrUserNotFound)
			},
			expectedError: domain.ErrUserNotFound,
		},
		{
			name:        "invalid reset code",
			email:       "test@example.com",
			code:        "wrongcode",
			newPassword: "newpassword123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				resetCode := domain.NewVerificationCode(user.ID, "123456", domain.PasswordReset, time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(resetCode, nil)
			},
			expectedError: domain.ErrInvalidPasswordChangeCode,
		},
		{
			name:        "expired reset code",
			email:       "test@example.com",
			code:        "123456",
			newPassword: "newpassword123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				resetCode := &domain.VerificationCode{
					ID:        ulid.Make(),
					UserID:    user.ID,
					Code:      "123456",
					Type:      domain.PasswordReset,
					ExpiresAt: time.Now().Add(-time.Hour), // Expired
					CreatedAt: time.Now().Add(-2 * time.Hour),
				}
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(resetCode, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(nil)
			},
			expectedError: domain.ErrPasswordChangeCodeExpired,
		},
		{
			name:        "password update fails",
			email:       "test@example.com",
			code:        "123456",
			newPassword: "newpassword123",
			setupMocks: func(m *MockUserRepository, v *mockVerificationCodeRepository) {
				user := &domain.User{
					ID:    ulid.Make(),
					Email: "test@example.com",
				}
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				resetCode := domain.NewVerificationCode(user.ID, "123456", domain.PasswordReset, time.Hour)
				v.On("FindByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(resetCode, nil)
				v.On("DeleteByUserIDAndType", mock.Anything, user.ID, domain.PasswordReset).Return(nil)
				m.On("UpdatePassword", mock.Anything, user.ID, mock.Anything).Return(domain.ErrDatabaseQuery)
			},
			expectedError: domain.ErrDatabaseQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockVerificationRepo := new(mockVerificationCodeRepository)
			mockJWTService := new(mockJWTService)
			mockEmailService := new(mockEmailService)
			logger := zap.NewNop()

			tt.setupMocks(mockUserRepo, mockVerificationRepo)

			service := NewAuthService(mockUserRepo, mockVerificationRepo, mockJWTService, mockEmailService, logger)

			err := service.ResetPassword(context.Background(), tt.email, tt.code, tt.newPassword)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
			mockVerificationRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*MockUserRepository)
		email         string
		password      string
		expectedError error
		expectedToken *domain.TokenPair
	}{
		{
			name: "user not found",
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, domain.ErrUserNotFound)
			},
			email:         "nonexistent@example.com",
			password:      "password123",
			expectedError: domain.ErrInvalidCredentials,
			expectedToken: nil,
		},
		{
			name: "invalid password",
			mockSetup: func(mockRepo *MockUserRepository) {
				hashedPassword, _ := password.HashPassword("correctpassword")
				mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					Password:      hashedPassword,
					Roles:         []string{"user"},
					EmailVerified: true,
				}, nil)
			},
			email:         "test@example.com",
			password:      "wrongpassword",
			expectedError: domain.ErrInvalidCredentials,
			expectedToken: nil,
		},
		{
			name: "email not verified",
			mockSetup: func(mockRepo *MockUserRepository) {
				hashedPassword, _ := password.HashPassword("correctpassword")
				mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					Password:      hashedPassword,
					Roles:         []string{"user"},
					EmailVerified: false,
				}, nil)
			},
			email:         "test@example.com",
			password:      "correctpassword",
			expectedError: domain.ErrEmailNotVerified,
			expectedToken: nil,
		},
		{
			name: "successful login",
			mockSetup: func(mockRepo *MockUserRepository) {
				hashedPassword, _ := password.HashPassword("correctpassword")
				mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{
					ID:            ulid.Make(),
					Email:         "test@example.com",
					Password:      hashedPassword,
					Roles:         []string{"user"},
					EmailVerified: true,
				}, nil)
			},
			email:         "test@example.com",
			password:      "correctpassword",
			expectedError: nil,
			expectedToken: &domain.TokenPair{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			mockJWTService := new(mockJWTService)
			mockEmailSvc := new(mockEmailService)
			service := NewAuthService(repo, nil, mockJWTService, mockEmailSvc, zap.NewNop())

			tt.mockSetup(repo)
			if tt.expectedToken != nil {
				mockJWTService.On("GenerateTokenPair", mock.Anything, mock.Anything).Return(&domain.TokenPair{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token",
				}, nil)
			}

			token, err := service.Login(context.Background(), tt.email, tt.password)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.NotEmpty(t, token.AccessToken)
				assert.NotEmpty(t, token.RefreshToken)
			}
		})
	}
}
