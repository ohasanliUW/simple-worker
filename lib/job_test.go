package job

import (
	"testing"

	"gopkg.in/check.v1"
)

type JobSuite struct{}

var _ = check.Suite(&JobSuite{})

func TestLib(t *testing.T) { check.TestingT(t) }

func (s *JobSuite) TestStart(c *check.C) {
	// Positive case
	good_command := "echo Hello World; sleep 2; echo Bye!!!"

	good_job := NewJob(good_command)
	c.Assert(good_job, check.NotNil)

	// TODO: Call Start()
	// TODO: check Status to make sure no errors
	// TODO: check PID > 0

	// Negative case
	bad_command := "maluba 123"
	bad_job := NewJob(bad_command)
	c.Assert(bad_job, check.NotNil)

	// TODO: Call Start()
	// TODO: check Status make sure it errored
	// TODO: check exit code

}

// TODO: Add some more test cases
