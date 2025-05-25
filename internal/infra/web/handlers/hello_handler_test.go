package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/responsehandler"
	"github.com/stretchr/testify/suite"
)

type HelloWebHandlerTestSuite struct {
	suite.Suite
	ResponseHandlerMock *responsehandler.WebResponseHandler
	HelloWebHandler     *HelloWebHandler
}

func TestHelloWebHandler(t *testing.T) {
	suite.Run(t, new(HelloWebHandlerTestSuite))
}

func (suite *HelloWebHandlerTestSuite) SetupTest() {
	suite.ResponseHandlerMock = &responsehandler.WebResponseHandler{}
	suite.HelloWebHandler = NewHelloWebHandler(suite.ResponseHandlerMock)
}

func (s *HelloWebHandlerTestSuite) TestSayHello() {
	s.Run("should say hello", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		s.HelloWebHandler.SayHello(w, r)

		res := w.Result()
		defer res.Body.Close()

		expected := `{"message":"Hello World!"}`

		s.Equal(http.StatusOK, res.StatusCode)
		s.Equal(expected, strings.TrimSuffix(w.Body.String(), "\n"))
	})
}
