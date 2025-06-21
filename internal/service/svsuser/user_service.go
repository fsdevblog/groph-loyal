package svsuser

import (
	"context"
	"fmt"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/service/svsuser/psswd"

	"github.com/fsdevblog/groph-loyal/internal/service/tokens"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"

	"github.com/fsdevblog/groph-loyal/pkg/uow"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

const JWTTokenExpire = 1 * time.Hour

type UserService struct {
	uow            uow.UOW
	userRepo       UserRepository
	psswdHasher    PasswordHasher
	jwtTokenSecret []byte
}

func New(u uow.UOW, jwtTokenSecret []byte) (*UserService, error) {
	userRepo, err := uow.GetRepositoryAs[UserRepository](u, uow.RepositoryName(repoargs.UserRepoName))
	if err != nil {
		return nil, err
	}
	return &UserService{
		uow:            u,
		userRepo:       userRepo,
		jwtTokenSecret: jwtTokenSecret,
		psswdHasher:    new(psswd.PasswordHash),
	}, nil
}

type RegisterUserArgs struct {
	Username string
	Password string
}

// Register создает юзера в базе данных. После успешного создания генерирует jwt token. Возвращает 3 значения:
// созданный юзер, токен и ошибку. В случае если username занят, возвращается ошибка domain.ErrDuplicateKey, во
// всех других кейсах - domain.ErrUnknown.
func (s *UserService) Register(ctx context.Context, args RegisterUserArgs) (*domain.User, string, error) {
	password, errPsswdHash := s.psswdHasher.HashPassword(args.Password)
	if errPsswdHash != nil {
		return nil, "", fmt.Errorf("registering user: %s", errPsswdHash.Error())
	}
	var user *domain.User
	var token string
	errTx := s.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		var errUser, errToken error
		userRepo, errUserRepo := uow.GetAs[UserRepository](tx, uow.RepositoryName(repoargs.UserRepoName))
		if errUserRepo != nil {
			return errUserRepo //nolint:wrapcheck
		}
		user, errUser = userRepo.CreateUser(c, repoargs.CreateUser{
			Username: args.Username,
			Password: password,
		})
		if errUser != nil {
			return errUser //nolint:wrapcheck
		}

		token, errToken = tokens.GenerateUserJWT(user.ID, JWTTokenExpire, s.jwtTokenSecret)
		if errToken != nil {
			return errToken //nolint:wrapcheck
		}
		return nil
	})

	if errTx != nil {
		return nil, "", fmt.Errorf("registering user: %w", errTx)
	}
	return user, token, nil
}

type LoginUserArgs struct {
	Username string
	Password string
}

// Login Ищет юзера в БД по LoginUserArgs.Username и сверяет пароль из БД с паролем LoginUserArgs.Password. Возвращает
// 3 значения: юзера, jwt токен и ошибку. В случае, если переданный username не найден, вернется ошибка
// domain.RecordNotFound, если пароль неверный - domain.ErrPasswordMissMatch, во всех других случаях -
// domain.ErrUnknown или не типизированная ошибка.
func (s *UserService) Login(ctx context.Context, args LoginUserArgs) (*domain.User, string, error) {
	user, errUser := s.userRepo.FindUserByUsername(ctx, args.Username)
	if errUser != nil {
		return nil, "", fmt.Errorf("logging in user: %w", errUser)
	}

	if !s.psswdHasher.ComparePassword(args.Password, user.EncryptedPassword) {
		return nil, "", domain.ErrPasswordMissMatch
	}
	token, errToken := tokens.GenerateUserJWT(user.ID, JWTTokenExpire, s.jwtTokenSecret)
	if errToken != nil {
		return nil, "", fmt.Errorf("logging in user: %s", errToken.Error())
	}

	return user, token, nil
}
