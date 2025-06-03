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

// CreateUser создает юзера в базе данных. В случае конфликта юзернейма возвращает ошибку domain.ErrDuplicateKey,
// во всех других случаях - domain.ErrUnknown.
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

// FindUserByUsername ищет юзера по его юзернейму. Возвращает ошибку domain.ErrRecordNotFound если запись не найдена,
// во всех других случаях - domain.ErrUnknown.
func (u *userRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	dbUser, err := u.q.Users_FindByUsername(ctx, username)
	if err != nil {
		return nil, convertErr(err, "finding user by username %s", username)
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
