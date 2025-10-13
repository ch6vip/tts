package jobs

import (
	"sync"
	"time"
	"tts/internal/models"

	"github.com/google/uuid"
)

// JobStore is a thread-safe in-memory store for TTS jobs.
type JobStore struct {
	jobs    map[string]*models.Job
	mu      sync.RWMutex
	ttl     time.Duration // Time-to-live for completed jobs
	cleanup time.Duration // Cleanup interval
}

// NewJobStore creates a new JobStore and starts the cleanup goroutine.
func NewJobStore(ttl, cleanupInterval time.Duration) *JobStore {
	store := &JobStore{
		jobs:    make(map[string]*models.Job),
		ttl:     ttl,
		cleanup: cleanupInterval,
	}
	go store.startCleanupRoutine()
	return store
}

// CreateJob creates a new job and returns it.
func (s *JobStore) CreateJob() *models.Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &models.Job{
		ID:        uuid.New().String(),
		Status:    models.JobStatusProcessing,
		CreatedAt: time.Now().UTC(),
	}
	s.jobs[job.ID] = job
	return job
}

// GetJob retrieves a job by its ID.
func (s *JobStore) GetJob(id string) (*models.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, found := s.jobs[id]
	return job, found
}

// UpdateProgress updates the progress of a job.
func (s *JobStore) UpdateProgress(id, progress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, found := s.jobs[id]; found {
		job.Progress = progress
	}
}

// SetJobComplete marks a job as complete and stores the audio data.
func (s *JobStore) SetJobComplete(id string, audioData []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, found := s.jobs[id]; found {
		now := time.Now().UTC()
		job.Status = models.JobStatusComplete
		job.AudioData = audioData
		job.Progress = "" // Clear progress
		job.CompletedAt = &now
	}
}

// SetJobError marks a job as failed with an error message.
func (s *JobStore) SetJobError(id, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, found := s.jobs[id]; found {
		now := time.Now().UTC()
		job.Status = models.JobStatusError
		job.Error = errorMsg
		job.CompletedAt = &now
	}
}

// startCleanupRoutine periodically removes old completed jobs.
func (s *JobStore) startCleanupRoutine() {
	ticker := time.NewTicker(s.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupJobs()
	}
}

// cleanupJobs removes jobs that have exceeded their TTL.
func (s *JobStore) cleanupJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for id, job := range s.jobs {
		// Clean up jobs that are completed and have expired
		if job.CompletedAt != nil && now.Sub(*job.CompletedAt) > s.ttl {
			delete(s.jobs, id)
		}
		// Clean up jobs that are still processing for a long time (e.g., > 1 hour)
		if job.Status == models.JobStatusProcessing && now.Sub(job.CreatedAt) > time.Hour {
			delete(s.jobs, id)
		}
	}
}