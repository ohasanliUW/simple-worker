package main

import (
	"errors"
	"fmt"
	lib "simple-worker/job"
)

type server struct {
	jobs map[string]map[int64]*lib.Job // all jobs created by server
}

// a custom error that will be returned when a request is not authorized
type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}

func main() {
	// srv := &server{
	// 	jobs: make(map[string]map[int64]*lib.Job),
	// }
}

// createJob creates a job with command for user with username
func (s *server) createJob(username string, command string) (*lib.Job, error) {
	job := lib.NewJob(command)

	if job == nil {
		return nil, errors.New("failed to create a job")
	}
	userJobs, exists := s.jobs[username]
	if !exists {
		s.jobs[username] = make(map[int64]*lib.Job)
		userJobs = s.jobs[username]
	}
	userJobs[job.Id()] = job
	return job, nil
}

func (s *server) stopJob(username string, job_id int64) error {
	// find job
	authErr := s.isAuthorized(username, job_id)
	if authErr != nil {
		return authErr
	}

	job := s.jobs[username][job_id]
	err := job.Stop()
	return err
}

// returns AuthError if user with username is not authorized to take action
// on a job with job_id
func (s *server) isAuthorized(username string, job_id int64) error {
	// find job
	userJobs, exists := s.jobs[username]
	if !exists {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	_, exists = userJobs[job_id]
	if !exists {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	return nil
}
