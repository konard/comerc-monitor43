//go:build smoke

package smoke

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	authv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMain(m *testing.M) {
	// chdir к корню монорепо: test/smoke/smoke_test.go → ../../../..
	if err := os.Chdir(filepath.Join("..", "..", "..", "..")); err != nil {
		panic("chdir to repo root: " + err.Error())
	}
	os.Exit(m.Run())
}

type SmokeSuite struct {
	suite.Suite
	stack    tc.ComposeStack
	httpPort string
	grpcPort string
}

func TestSmoke(t *testing.T) {
	suite.Run(t, new(SmokeSuite))
}

func (s *SmokeSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	stack, err := tc.NewDockerCompose("backend/auth-service/test/smoke/testdata/docker-compose.yml")
	s.Require().NoError(err)
	s.stack = stack

	err = stack.
		WaitForService("auth-service",
			wait.ForListeningPort("8080/tcp").WithStartupTimeout(3*time.Minute),
		).
		Up(ctx, tc.Wait(true))
	s.Require().NoError(err, "smoke stack failed to start")

	container, err := stack.ServiceContainer(ctx, "auth-service")
	s.Require().NoError(err)

	httpPort, err := container.MappedPort(ctx, "8080/tcp")
	s.Require().NoError(err)
	s.httpPort = httpPort.Port()

	grpcPort, err := container.MappedPort(ctx, "5001/tcp")
	s.Require().NoError(err)
	s.grpcPort = grpcPort.Port()
}

func (s *SmokeSuite) TearDownSuite() {
	if s.stack != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = s.stack.Down(ctx, tc.RemoveOrphans(true))
	}
}

func (s *SmokeSuite) TestHealth() {
	resp := s.get("/health")
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *SmokeSuite) TestReady() {
	resp := s.get("/ready")
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *SmokeSuite) TestGRPCValidateInvalidToken() {
	conn, err := grpc.NewClient(
		"localhost:"+s.grpcPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		AccessToken: "invalid.token",
	})
	s.Require().NoError(err)
	s.Assert().False(resp.Valid)
}

func (s *SmokeSuite) get(path string) *http.Response {
	s.T().Helper()
	url := fmt.Sprintf("http://localhost:%s%s", s.httpPort, path)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(url)
	s.Require().NoError(err, "GET %s", path)
	return resp
}
