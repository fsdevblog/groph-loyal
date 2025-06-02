package service

import (
	"context"
	"fmt"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/tokens"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/uow"
	"golang.org/x/crypto/bcrypt"
)

const JWTTokenExpire = 1 * time.Hour

type UserService struct {
	uow            uow.UOW
	userRepo       domain.UserRepository
	jwtTokenSecret []byte
}

func NewUserService(u uow.UOW, jwtTokenSecret []byte) (*UserService, error) {
	userRepo, userRepoErr := uow.GetRepositoryAs[domain.UserRepository](u, uow.RepositoryName(domain.UserRepoName))
	if userRepoErr != nil {
		return nil, userRepoErr
	}
	return &UserService{
		uow:            u,
		userRepo:       userRepo,
		jwtTokenSecret: jwtTokenSecret,
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
	password, hashErr := s.hashPassword(args.Password)
	if hashErr != nil {
		return nil, "", fmt.Errorf("registering user: %s", hashErr.Error())
	}
	var user *domain.User
	var token string
	txErr := s.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		var userErr, tokenErr error
		userRepo, userRepoErr := uow.GetAs[domain.UserRepository](tx, uow.RepositoryName(domain.UserRepoName))
		if userRepoErr != nil {
			return userRepoErr //nolint:wrapcheck
		}
		user, userErr = userRepo.CreateUser(c, domain.User{
			Username: args.Username,
			Password: password,
		})
		if userErr != nil {
			return userErr //nolint:wrapcheck
		}

		token, tokenErr = tokens.GenerateUserJWT(user.ID, JWTTokenExpire, s.jwtTokenSecret)
		if tokenErr != nil {
			return tokenErr //nolint:wrapcheck
		}
		return nil
	})

	if txErr != nil {
		return nil, "", fmt.Errorf("registering user: %w", txErr)
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
// domain.ErrUnknown.
func (s *UserService) Login(ctx context.Context, args LoginUserArgs) (*domain.User, string, error) {
	userRepo, userRepoErr := uow.GetRepositoryAs[domain.UserRepository](s.uow, uow.RepositoryName(domain.UserRepoName))
	if userRepoErr != nil {
		return nil, "", fmt.Errorf("logging in user: %s", userRepoErr.Error())
	}

	user, userErr := userRepo.FindUserByUsername(ctx, args.Username)
	if userErr != nil {
		return nil, "", fmt.Errorf("logging in user: %w", userErr)
	}

	if !s.comparePasswords(user.Password, args.Password) {
		return nil, "", domain.ErrPasswordMissMatch
	}
	token, tokenErr := tokens.GenerateUserJWT(user.ID, JWTTokenExpire, s.jwtTokenSecret)
	if tokenErr != nil {
		return nil, "", fmt.Errorf("logging in user: %s", tokenErr.Error())
	}

	return user, token, nil
}

func (s *UserService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %s", err.Error())
	}
	return string(bytes), nil
}

func (s *UserService) comparePasswords(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
