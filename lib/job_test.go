package job

import (
	"testing"
	"time"

	"gopkg.in/check.v1"
)

type JobSuite struct{}

var _ = check.Suite(&JobSuite{})

func TestLib(t *testing.T) { check.TestingT(t) }

func (s *JobSuite) TestGoodStart(c *check.C) {
	// Positive case
	good_command := "echo Hello World; sleep 2; echo Bye!!!"

	good_job := NewJob(good_command)
	c.Assert(good_job, check.NotNil)

	good_job.Start()

	good_job.Wait()
	c.Assert(good_job.Pid, check.Not(check.Equals), 0)
	good_status := good_job.Status()
	INFO.Printf("TEST: %v\n", good_status)
	c.Assert(good_status.Exited, check.Equals, true)
	c.Assert(good_status.ExitCode, check.Equals, 0)

	// // Negative case
	// bad_command := "maluba 123"
	// bad_job := NewJob(bad_command)
	// c.Assert(bad_job, check.NotNil)
	// bad_job.Start()

	// bad_job.Wait()

	// TODO: Call Start()
	// TODO: check Status make sure it errored
	// TODO: check exit code

}

func (s *JobSuite) TestBadStart(c *check.C) {
	// Negative case
	bad_command := "maluba 123"
	bad_job := NewJob(bad_command)
	c.Assert(bad_job, check.NotNil)
	bad_job.Start()
	bad_job.Wait()
	c.Assert(bad_job.status.Exited, check.Equals, true)

	INFO.Printf("LIB-TEST: Exit reason: %v\n", bad_job.Status().String())

	// TODO: Call Start()
	// TODO: check Status make sure it errored
	// TODO: check exit code

}

func (s *JobSuite) TestStop(c *check.C) {
	job := NewJob("sleep 100")
	c.Assert(job, check.NotNil)

	job.Start()
	//wait for few milliseconds then stop the job.
	time.Sleep(time.Millisecond * 100)

	c.Assert(job.Pid, check.Not(check.Equals), 0)
	c.Assert(job.Status().Exited, check.Equals, false)

	err := job.Stop()
	job.Wait()
	c.Assert(err, check.IsNil)
	c.Assert(job.Status().Exited, check.Equals, true)
	INFO.Printf("LIB-TEST: Exit reason: %v\n", job.Status().String())
	INFO.Println(job.Status().ExitCode)
}
