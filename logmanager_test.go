package logwhale

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type LogManagerTestSuite struct {
	suite.Suite
	lm  *LogManager
	ctx context.Context
}

func (s *LogManagerTestSuite) SetupTest() {
	s.ctx = context.Background()
	var err error
	s.lm, err = NewLogManager(s.ctx)
	s.NoError(err)
}

func (s *LogManagerTestSuite) TearDownTest() {
	s.lm.Close()
}

func (s *LogManagerTestSuite) TestAddRemoveFile() {
	s.NoError(s.lm.ctx.Err())

	of, err := os.CreateTemp(os.TempDir(), "test_log")
	defer of.Close()
	s.NoError(err)

	_, _, err = s.lm.AddLogFile(of.Name())
	s.NoError(err)

	_, exists := s.lm.logsWatched[of.Name()]
	s.True(exists)

	err = s.lm.RemoveLogFile(of.Name())
	s.NoError(err)
}

func (s *LogManagerTestSuite) TestErrorOnCreateLogManager() {
	s.ctx = nil
	_, err := NewLogManager(s.ctx)
	s.Error(err)
}

func (s *LogManagerTestSuite) TestErrorOnAddLogFile() {
	dirname, err := os.MkdirTemp(os.TempDir(), "testdir")
	s.NoError(err)

	dataChan, errorChan, err := s.lm.AddLogFile(dirname)
	s.Nil(dataChan)
	s.Nil(errorChan)
	s.Error(err)

	err = os.Remove(dirname)
	s.NoError(err)
}

func (s *LogManagerTestSuite) TestRunAndClose() {
	of, err := os.CreateTemp(os.TempDir(), "test_log")
	defer of.Close()
	s.NoError(err)

	dataChan, errorChan, err := s.lm.AddLogFile(of.Name())
	s.NoError(err)

	_, err = of.WriteString("test line\n")
	s.NoError(err)

	time.Sleep(time.Second)

	select {
	case data := <-dataChan:
		s.Equal(string(data), "test line")
	case err := <-errorChan:
		s.Fail("unexpected error: %v", err)
	default:
		s.Fail("expected data, got none")
	}

	s.NoError(s.lm.Close())
	s.True(s.lm.closed)
}

func TestLogManagerTestSuite(t *testing.T) {
	suite.Run(t, new(LogManagerTestSuite))
}
