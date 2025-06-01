package service

import (
	"context"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/uow"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	uow      uow.UOW
	userRepo domain.UserRepository
}

func NewUserService(u uow.UOW) (*UserService, error) {
	userRepo, userRepoErr := uow.GetRepositoryAs[domain.UserRepository](u, uow.RepositoryName(domain.UserRepoName))
	if userRepoErr != nil {
		return nil, userRepoErr
	}
	return &UserService{
		uow:      u,
		userRepo: userRepo,
	}, nil
}

type RegisterUserArgs struct {
	Username string
	Password string
}

func (s *UserService) Register(ctx context.Context, args RegisterUserArgs) (*domain.User, error) {
	password, hashErr := s.hashPassword(args.Password)
	if hashErr != nil {
		return nil, fmt.Errorf("registering user: %s", hashErr.Error())
	}
	user, createErr := s.userRepo.CreateUser(ctx, domain.User{
		Username: args.Username,
		Password: password,
	})

	return user, fmt.Errorf("registering user: %w", createErr)
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
