package sqlc

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
)

type userRepository struct {
	q *sqlcgen.Queries
}

func NewUserRepository(conn sqlcgen.DBTX) domain.UserRepository {
	return &userRepository{q: sqlcgen.New(conn)}
}

func (u *userRepository) CreateUser(ctx context.Context, user domain.User) (*domain.User, error) {
	dbUser, err := u.q.Users_Create(ctx, sqlcgen.Users_CreateParams{
		Username: user.Username,
		Password: user.Password,
	})
	if err != nil {
		return nil, convertErr(err, "creating user")
	}

	return convertUserModel(dbUser), nil
}

func convertUserModel(dbModel sqlcgen.User) *domain.User {
	return &domain.User{
		ID:        dbModel.ID,
		CreatedAt: dbModel.CreatedAt.Time,
		UpdatedAt: dbModel.UpdatedAt.Time,
		Username:  dbModel.Username,
		Password:  dbModel.Password,
	}
}
