package service

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(password string, hashedPassword string) bool
}
