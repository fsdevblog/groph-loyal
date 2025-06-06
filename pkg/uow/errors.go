package uow

import "errors"

var (
	ErrRepositoryNotRegistered     = errors.New("[uow] repository not registered")
	ErrRepositoryAlreadyRegistered = errors.New("[uow] repository already registered")
	ErrInvalidRepositoryType       = errors.New("[uow] invalid repository type")
)
