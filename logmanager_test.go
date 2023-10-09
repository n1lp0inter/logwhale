package logwhale

import (
	"context"
	"fmt"
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

func (s *LogManagerTestSuite) TestCleanup() {
	of, err := os.CreateTemp(os.TempDir(), "test_log")
	defer of.Close()
	s.NoError(err)

	_, _, err = s.lm.AddLogFile(of.Name())
	s.NoError(err)

	_, err = of.WriteString("test line\n")
	s.NoError(err)

	s.NoError(s.lm.Close())
	s.True(s.lm.closed)

	s.Equal(0, len(s.lm.logsWatched)) // Not strictly necessary but we should clean up the logsWatched map
	s.Equal(0, len(s.lm.pathsWatched))
}

func (s *LogManagerTestSuite) TestRunAndClose() {
	of, err := os.CreateTemp(os.TempDir(), "test_log")
	defer of.Close()
	s.NoError(err)

	dataChan, errorChan, err := s.lm.AddLogFile(of.Name())
	s.NoError(err)

	_, err = of.WriteString("test line\n")
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	select {
	case data := <-dataChan:
		s.Equal("test line", string(data))
	case err := <-errorChan:
		s.Fail("unexpected error: %v", err)
	default:
		s.Fail("expected data, got none")
	}
}

func (s *LogManagerTestSuite) TestRemoveLogFile() {
	of, err := os.CreateTemp(os.TempDir(), "test_log")
	defer of.Close()
	s.NoError(err)

	dataChan, errorChan, err := s.lm.AddLogFile(of.Name())
	s.NoError(err)

	_, err = of.WriteString("test line\n")
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	select {
	case data := <-dataChan:
		s.Equal("test line", string(data))
	case err := <-errorChan:
		s.Fail("unexpected error: %v", err)
	default:
		s.Fail("expected data, got none")
	}

	err = of.Close()
	s.NoError(err)
	err = os.Remove(of.Name())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	select {
	case err := <-errorChan:
		s.Equal(fmt.Sprintf("state: File removed msg: file removed, waiting for creation: %s", of.Name()), err.Error())
	}
}

func TestLogManagerTestSuite(t *testing.T) {
	suite.Run(t, new(LogManagerTestSuite))
}
