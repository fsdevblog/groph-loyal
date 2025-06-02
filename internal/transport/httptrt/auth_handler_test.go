package httptrt

import (
	"bytes"
	"encoding/json"
	"github.com/fsdevblog/groph-loyal/internal/config"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/logger"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/mocks"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/testutils"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"testing"
)

type AuthHandlerTestSuite struct {
	suite.Suite
	mockUserService *mocks.MockUserServicer
	router          *gin.Engine
	config          *config.Config
}

func (s *AuthHandlerTestSuite) SetupTest() {
	mockUser := gomock.NewController(s.T())
	defer mockUser.Finish()

	s.mockUserService = mocks.NewMockUserServicer(mockUser)
	s.config = &config.Config{
		RunAddress: "localhost:80",
	}

	s.router = New(RouterArgs{
		Logger:      logger.New(os.Stdout),
		UserService: s.mockUserService,
	})
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}

func (s *AuthHandlerTestSuite) TestRegister() {
	argsOk := service.RegisterUserArgs{Username: "test", Password: "password"}
	argsDup := service.RegisterUserArgs{Username: "duplicate", Password: "password"}
	argsIncorrectUsername := service.RegisterUserArgs{Username: "", Password: "password"}
	argsIncorrectPassword := service.RegisterUserArgs{Username: "test", Password: ""}

	s.mockUserService.EXPECT().Register(gomock.Any(), argsOk).Return(&domain.User{}, "valid_token", nil).Times(1)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsDup).Return(nil, "", domain.ErrDuplicateKey).Times(1)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsIncorrectUsername).Times(0)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsIncorrectPassword).Times(0)

	var cases = []struct {
		name       string
		args       *UserRegisterParams
		wantStatus int
	}{
		{
			name:       "user created",
			args:       &UserRegisterParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus: http.StatusOK,
		}, {
			name:       "duplicate username",
			args:       &UserRegisterParams{Username: argsDup.Username, Password: argsDup.Password},
			wantStatus: http.StatusConflict,
		}, {
			name:       "bad request",
			args:       nil,
			wantStatus: http.StatusBadRequest,
		}, {
			name: "empty username",
			args: &UserRegisterParams{
				Username: argsIncorrectUsername.Username,
				Password: argsIncorrectUsername.Password,
			},
			wantStatus: http.StatusUnprocessableEntity,
		}, {
			name: "empty password",
			args: &UserRegisterParams{
				Username: argsIncorrectPassword.Username,
				Password: argsIncorrectPassword.Password,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			var payload []byte
			if t.args != nil {
				payload, _ = json.Marshal(t.args)
			}

			res, err := testutils.MakeRequest(testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodPost,
				URL:    APIRouteGroup + APIRegisterRoute,
				Body:   bytes.NewReader(payload),
			})
			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}

func (s *AuthHandlerTestSuite) TestLogin() {
	argsOk := service.LoginUserArgs{Username: "test", Password: "password"}
	argsWrongUsername := service.LoginUserArgs{Username: "wrong", Password: "<PASSWORD>"}
	argsWrongPass := service.LoginUserArgs{Username: "test", Password: "<wrong>"}

	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsOk).
		Return(&domain.User{}, "token", nil).
		Times(1)
	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsWrongUsername).
		Return(nil, "", domain.ErrRecordNotFound).
		Times(1)
	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsWrongPass).
		Return(nil, "", domain.ErrPasswordMissMatch).
		Times(1)

	cases := []struct {
		name       string
		args       *UserLoginParams
		wantStatus int
	}{
		{
			name:       "ok",
			args:       &UserLoginParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus: http.StatusOK,
		}, {
			name:       "bad request",
			args:       nil,
			wantStatus: http.StatusBadRequest,
		}, {
			name:       "wrong username",
			args:       &UserLoginParams{Username: argsWrongUsername.Username, Password: argsWrongUsername.Password},
			wantStatus: http.StatusUnauthorized,
		}, {
			name:       "wrong password",
			args:       &UserLoginParams{Username: argsWrongPass.Username, Password: argsWrongPass.Password},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			var payload []byte
			if t.args != nil {
				payload, _ = json.Marshal(t.args)
			}

			res, err := testutils.MakeRequest(testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodPost,
				URL:    APIRouteGroup + APILoginRoute,
				Body:   bytes.NewReader(payload),
			})
			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}
