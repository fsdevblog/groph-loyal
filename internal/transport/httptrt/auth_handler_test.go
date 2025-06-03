package httptrt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fsdevblog/groph-loyal/internal/config"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/logger"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/mocks"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/testutils"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/tokens"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"testing"
	"time"
)

type AuthHandlerTestSuite struct {
	suite.Suite
	mockUserService *mocks.MockUserServicer
	router          *gin.Engine
	config          *config.Config
	jwtSecret       []byte
}

func (s *AuthHandlerTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	s.mockUserService = mocks.NewMockUserServicer(mockCtrl)
	s.config = &config.Config{
		RunAddress: "localhost:80",
	}
	s.jwtSecret = []byte("super secret key")

	s.router = New(RouterArgs{
		Logger:       logger.New(os.Stdout),
		UserService:  s.mockUserService,
		JWTSecretKey: s.jwtSecret,
	})
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}

func (s *AuthHandlerTestSuite) TestRegister() {
	jwtTokenStr, jwtErr := tokens.GenerateUserJWT(1, time.Hour, s.jwtSecret)
	s.Require().NoError(jwtErr)

	argsOk := service.RegisterUserArgs{Username: "test", Password: "password"}
	argsDup := service.RegisterUserArgs{Username: "duplicate", Password: "password"}
	argsIncorrectUsername := service.RegisterUserArgs{Username: "", Password: "password"}
	argsIncorrectPassword := service.RegisterUserArgs{Username: "test"}

	s.mockUserService.EXPECT().Register(gomock.Any(), argsOk).Return(&domain.User{}, jwtTokenStr, nil)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsDup).Return(nil, "", domain.ErrDuplicateKey)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsIncorrectUsername).Times(0)
	s.mockUserService.EXPECT().Register(gomock.Any(), argsIncorrectPassword).Times(0)

	var cases = []struct {
		name        string
		args        *UserRegisterParams
		jwtTokenStr *string
		wantStatus  int
	}{
		{
			name:       "user created",
			args:       &UserRegisterParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus: http.StatusOK,
		}, {
			name:        "user already logged in",
			args:        &UserRegisterParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus:  http.StatusUnauthorized,
			jwtTokenStr: &jwtTokenStr,
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

			args := testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodPost,
				URL:    APIRouteGroup + APIRegisterRoute,
				Body:   bytes.NewReader(payload),
			}

			var reqOpts []func(*testutils.RequestOptions)
			if t.jwtTokenStr != nil {
				v := fmt.Sprintf("Bearer %s", *t.jwtTokenStr)
				reqOpts = append(reqOpts, testutils.WithHeader("Authorization", v))
			}

			res, err := testutils.MakeRequest(args, reqOpts...)

			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}

func (s *AuthHandlerTestSuite) TestLogin() {
	jwtTokenStr, jwtErr := tokens.GenerateUserJWT(1, time.Hour, s.jwtSecret)
	s.Require().NoError(jwtErr)

	argsOk := service.LoginUserArgs{Username: "test", Password: "password"}
	argsWrongUsername := service.LoginUserArgs{Username: "wrong", Password: "<PASSWORD>"}
	argsWrongPass := service.LoginUserArgs{Username: "test", Password: "<wrong>"}

	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsOk).
		Return(&domain.User{}, "token", nil)
	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsWrongUsername).
		Return(nil, "", domain.ErrRecordNotFound)
	s.mockUserService.EXPECT().
		Login(gomock.Any(), argsWrongPass).
		Return(nil, "", domain.ErrPasswordMissMatch)

	cases := []struct {
		name        string
		args        *UserLoginParams
		jwtTokenStr *string
		wantStatus  int
	}{
		{
			name:       "ok",
			args:       &UserLoginParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus: http.StatusOK,
		}, {
			name:        "already logged in",
			args:        &UserLoginParams{Username: argsOk.Username, Password: argsOk.Password},
			wantStatus:  http.StatusUnauthorized,
			jwtTokenStr: &jwtTokenStr,
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

			args := testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodPost,
				URL:    APIRouteGroup + APILoginRoute,
				Body:   bytes.NewReader(payload),
			}

			var reqOpts []func(*testutils.RequestOptions)
			if t.jwtTokenStr != nil {
				v := fmt.Sprintf("Bearer %s", *t.jwtTokenStr)
				reqOpts = append(reqOpts, testutils.WithHeader("Authorization", v))
			}

			res, err := testutils.MakeRequest(args, reqOpts...)
			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}
