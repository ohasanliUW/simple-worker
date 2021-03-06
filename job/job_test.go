package job

import (
	"fmt"
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
	pid, err := good_job.Pid()
	c.Assert(err, check.IsNil)
	c.Assert(pid, check.Not(check.Equals), -1)
	good_status := good_job.Status()
	c.Assert(good_status.Status, check.Equals, EXITED)
	c.Assert(good_status.ExitCode, check.Equals, 0)
}

func (s *JobSuite) TestBadStart(c *check.C) {
	// Negative case
	bad_command := "maluba 123"
	bad_job := NewJob(bad_command)
	c.Assert(bad_job, check.NotNil)
	bad_job.Start()
	bad_job.Wait()
	c.Assert(bad_job.Status().Status, check.Equals, EXITED)
	c.Assert(bad_job.Status().ExitCode, check.Equals, -1)
}

func (s *JobSuite) TestStop(c *check.C) {
	job := NewJob("sleep 100")
	c.Assert(job, check.NotNil)

	job.Start()
	//wait for few milliseconds then stop the job.
	time.Sleep(time.Millisecond * 100)

	pid, err := job.Pid()
	c.Assert(err, check.IsNil)
	c.Assert(pid, check.Not(check.Equals), -1)
	c.Assert(job.Status().Status, check.Equals, RUNNING)

	err = job.Stop()
	job.Wait()
	c.Assert(err, check.IsNil)
	c.Assert(job.Status().Status, check.Equals, EXITED)
}

func (s *JobSuite) TestOutput(c *check.C) {
	N := 5
	job := NewJob(fmt.Sprintf("../testdata/echo.sh %v", N))
	c.Assert(job, check.NotNil)

	job.Start()

	expect := N * 2 // expecting N lines of stdout and N lines of stderr output
	actual1 := 0    // actual lines from first readout
	actual2 := 0    // actual lines from second readout
	ch1 := make(chan int)
	ch2 := make(chan int)

	// readHandle is a function that will read from channel
	// and count number of lines till channel is closed
	readHandle := func(s chan []byte, outC chan int) {
		counter := 0
		for {
			_, open := <-s

			// If stream is closed, then we are at the end of the output
			if !open {
				break
			}
			counter += 1
		}
		outC <- counter
		close(outC)
	}

	// get first stream
	s1 := job.Output()
	// start the first readout
	go readHandle(s1, ch1)
	// wait for couple seconds
	time.Sleep(2 * time.Second)

	// get second stream and start readout
	s2 := job.Output()
	go readHandle(s2, ch2)

	job.Wait()

	actual1, actual2 = <-ch1, <-ch2
	c.Assert(actual1, check.Equals, expect)
	c.Assert(actual2, check.Equals, expect)
}
