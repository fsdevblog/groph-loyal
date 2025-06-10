package service

import (
	"context"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"

	"github.com/fsdevblog/groph-loyal/pkg/uow"
	uowmocks "github.com/fsdevblog/groph-loyal/pkg/uow/mocks"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service/mocks"
	"github.com/fsdevblog/groph-loyal/internal/transport/api/tokens"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type UserServiceTestSuite struct {
	suite.Suite
	mockUOW      *uowmocks.MockUOW
	mockTX       *uowmocks.MockTX
	mockUserRepo *mocks.MockUserRepository
	mockPsswd    *mocks.MockPasswordHasher
	jwtSecret    []byte
	userService  *UserService
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

func (s *UserServiceTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.mockUOW = uowmocks.NewMockUOW(mockCtrl)
	s.mockUserRepo = mocks.NewMockUserRepository(mockCtrl)
	s.mockPsswd = mocks.NewMockPasswordHasher(mockCtrl)
	s.mockTX = uowmocks.NewMockTX(mockCtrl)

	s.jwtSecret = []byte("secret")

	// Мок получения репозитория из uow. Выполняется в инициализации сервиса.
	s.mockUOW.EXPECT().GetRepository(uow.RepositoryName(repoargs.UserRepoName)).
		Return(s.mockUserRepo, nil).AnyTimes()

	// Инициализация сервиса.
	userService, servErr := NewUserService(s.mockUOW, s.jwtSecret)

	// заменяем хешер пароля на мок
	userService.psswdHasher = s.mockPsswd

	s.Require().NoError(servErr)
	s.userService = userService
}

func (s *UserServiceTestSuite) TestLogin() {
	savedUserUsername := "test"
	// аргументы вызовов для кейсов ниже.
	argsOk := LoginUserArgs{
		Username: savedUserUsername,
		Password: "<PASSWORD>",
	}
	argsWrongUsername := LoginUserArgs{
		Username: "wrong",
		Password: "<PASSWORD>",
	}
	argsWrongPass := LoginUserArgs{
		Username: savedUserUsername,
		Password: "wrong pass",
	}

	validHashPassword := "hash ok"

	savedUser := domain.User{
		ID:                1,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Username:          savedUserUsername,
		EncryptedPassword: validHashPassword,
	}

	// Мок для сравнения пароля.
	s.mockPsswd.EXPECT().ComparePassword(argsOk.Password, validHashPassword).Return(true)
	s.mockPsswd.EXPECT().ComparePassword(argsWrongUsername.Password, validHashPassword).Times(0)
	s.mockPsswd.EXPECT().ComparePassword(argsWrongPass.Password, validHashPassword).Return(false)

	// Мок репозитория.
	s.mockUserRepo.EXPECT().
		FindUserByUsername(gomock.Any(), savedUserUsername).
		Return(&savedUser, nil).Times(2)

	s.mockUserRepo.EXPECT().
		FindUserByUsername(gomock.Any(), argsWrongUsername.Username).
		Return(nil, domain.ErrRecordNotFound)

	cases := []struct {
		name               string
		args               LoginUserArgs
		wantErr            error
		wantHashedPassword string
	}{
		{name: "ok", args: argsOk, wantErr: nil, wantHashedPassword: validHashPassword},
		{name: "wrong username", args: argsWrongUsername, wantErr: domain.ErrRecordNotFound},
		{name: "wrong password", args: argsWrongPass, wantErr: domain.ErrPasswordMissMatch},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			user, tokenStr, err := s.userService.Login(s.T().Context(), t.args)
			s.Require().ErrorIs(err, t.wantErr)

			if t.wantErr == nil {
				s.Equal(t.wantHashedPassword, user.EncryptedPassword)
				s.NotEmpty(tokenStr)

				token, tokenErr := tokens.ValidateUserJWT(tokenStr, s.jwtSecret)
				s.Require().NoError(tokenErr)
				s.Equal(token.Claims.(*tokens.UserClaims).ID, savedUser.ID) //nolint:errcheck
				s.NotNil(user)
			}
		})
	}
}

func (s *UserServiceTestSuite) TestRegister() {
	argsOk := RegisterUserArgs{
		Username: "validUser",
		Password: "<PASSWORD>",
	}
	argsDuplicateUsername := RegisterUserArgs{
		Username: "duplicateUser",
		Password: "<PASSWORD>",
	}

	encryptedPassword := "hashedPassword"
	dupEncryptedPassword := "duphashedpass"

	createdUser := domain.User{
		ID:                1,
		Username:          argsOk.Username,
		EncryptedPassword: encryptedPassword,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Мок транзакции uow.
	s.mockTX.EXPECT().Get(uow.RepositoryName(repoargs.UserRepoName)).
		Return(s.mockUserRepo, nil).MinTimes(1)

	// Мок хеширования пароля.
	s.mockPsswd.EXPECT().HashPassword(argsOk.Password).Return(encryptedPassword, nil)
	s.mockPsswd.EXPECT().HashPassword(argsDuplicateUsername.Password).Return(dupEncryptedPassword, nil)

	// Мок репозитория.
	s.mockUserRepo.EXPECT().
		CreateUser(gomock.Any(), repoargs.CreateUser{
			Username: argsOk.Username,
			Password: encryptedPassword,
		}).
		Return(&createdUser, nil)

	s.mockUserRepo.EXPECT().
		CreateUser(gomock.Any(), repoargs.CreateUser{
			Username: argsDuplicateUsername.Username,
			Password: dupEncryptedPassword,
		}).
		Return(nil, domain.ErrDuplicateKey)

	// Мок uow.
	s.mockUOW.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context, uow.TX) error) error {
			return fn(ctx, s.mockTX)
		}).AnyTimes()

	cases := []struct {
		name      string
		args      RegisterUserArgs
		wantErr   error
		wantUser  *domain.User
		wantToken bool
	}{
		{
			name:      "ok",
			args:      argsOk,
			wantUser:  &createdUser,
			wantToken: true,
		},
		{
			name:    "duplicate username",
			args:    argsDuplicateUsername,
			wantErr: domain.ErrDuplicateKey,
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			user, tokenStr, err := s.userService.Register(s.T().Context(), t.args)

			s.Require().ErrorIs(err, t.wantErr)
			s.Equal(t.wantUser, user)

			if t.wantToken {
				s.Require().NotEmpty(tokenStr)

				token, tokenErr := tokens.ValidateUserJWT(tokenStr, s.jwtSecret)
				s.Require().NoError(tokenErr)
				s.Equal(token.Claims.(*tokens.UserClaims).ID, user.ID) //nolint:errcheck
			} else {
				s.Empty(tokenStr)
			}
		})
	}
}
