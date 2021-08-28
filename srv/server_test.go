package main

import (
	"reflect"
	lib "simple-worker/job"
	"testing"

	"gopkg.in/check.v1"
)

type ServerSuite struct{}

var _ = check.Suite(&ServerSuite{})

func TestServer(t *testing.T) { check.TestingT(t) }

func (s *ServerSuite) TestCreateJob(c *check.C) {
	srv := &server{
		jobs: make(map[string]map[int64]*lib.Job),
	}

	username := "alice"
	command := "echo Hello There"

	job, err := srv.startJob(username, command)
	c.Assert(err, check.IsNil)
	c.Assert(job, check.NotNil)
}

func (s *ServerSuite) TestDenyCommand(c *check.C) {
	srv := &server{
		jobs: make(map[string]map[int64]*lib.Job),
	}

	alice := "alice"
	command := "echo Hello There"

	job, err := srv.startJob(alice, command)
	c.Assert(err, check.IsNil)
	c.Assert(job, check.NotNil)

	bob := "bob"
	err = srv.stopJob(bob, job.Id())

	errType := reflect.TypeOf(err)
	c.Assert(errType, check.Equals, reflect.TypeOf(&AuthError{}))
}

func (s *ServerSuite) TestPermitCommand(c *check.C) {
	srv := &server{
		jobs: make(map[string]map[int64]*lib.Job),
	}

	alice := "alice"
	command := "echo Hello There"

	job, err := srv.startJob(alice, command)
	c.Assert(err, check.IsNil)
	c.Assert(job, check.NotNil)

	err = srv.stopJob(alice, job.Id())

	errType := reflect.TypeOf(err)
	c.Assert(errType, check.Not(check.Equals), reflect.TypeOf(&AuthError{}))
}
