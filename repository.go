// +build !mock

package dbconnector

import (
	"context"

	"github.com/go-jwdk/db-connector/internal"
)

type repositoryFactory struct {
	tmpl SQLTemplate
}

func (rf *repositoryFactory) new(querier internal.Querier) *repository {
	return &repository{querier: querier, tmpl: rf.tmpl}
}

type repository struct {
	querier internal.Querier
	tmpl    SQLTemplate
}

func (r *repository) enqueueJob(ctx context.Context, queue, jobID, content string, deduplicationID, groupID *string, delaySeconds int64) error {
	stmt, stmtArgs := r.tmpl.NewEnqueueJobDML(queue,
		jobID, content, deduplicationID, groupID, delaySeconds)
	_, err := r.querier.ExecContext(ctx, stmt, stmtArgs...)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) enqueueJobWithTime(ctx context.Context, queue string, jobID, content string, deduplicationID, groupID *string, enqueueAt int64) error {
	stmt, stmtArgs := r.tmpl.NewEnqueueJobWithTimeDML(queue,
		jobID, content, deduplicationID, groupID, enqueueAt)
	_, err := r.querier.ExecContext(ctx, stmt, stmtArgs...)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) deleteJob(ctx context.Context, queue string, jobID string) error {
	stmt, args := r.tmpl.NewDeleteJobDML(queue, jobID)
	_, err := r.querier.ExecContext(ctx, stmt, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) getJob(ctx context.Context, queue string, jobID string) (*internal.Job, error) {
	stmt, args := r.tmpl.NewFindJobDML(queue, jobID)
	row := r.querier.QueryRowContext(ctx, stmt, args...)

	var job internal.Job
	if err := row.Scan(
		&job.SecID,
		&job.JobID,
		&job.Content,
		&job.DeduplicationID,
		&job.GroupID,
		&job.InvisibleUntil,
		&job.RetryCount,
		&job.EnqueueAt,
	); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *repository) getJobs(ctx context.Context, queue string, limit int64) ([]*internal.Job, error) {

	if limit == 0 {
		limit = 1
	}

	stmt, args := r.tmpl.NewFindJobsDML(queue, limit)
	rows, err := r.querier.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}

	var jobs []*internal.Job
	for rows.Next() {
		var job internal.Job
		if err := rows.Scan(
			&job.SecID,
			&job.JobID,
			&job.Content,
			&job.DeduplicationID,
			&job.GroupID,
			&job.InvisibleUntil,
			&job.RetryCount,
			&job.EnqueueAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func (r *repository) grabJob(ctx context.Context,
	queue string, jobID string, currentRetryCount, currentInvisibleUntil, invisibleTime int64) (grabbed bool, err error) {
	stmt, args := r.tmpl.NewHideJobDML(queue, jobID, currentRetryCount, currentInvisibleUntil, invisibleTime)
	result, err := r.querier.ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected == 1, nil
}

func (r *repository) updateJobVisibility(ctx context.Context, queueRawName, jobID string, visibilityTimeout int64) (updated bool, err error) {
	stmt, args := r.tmpl.NewUpdateJobByVisibilityTimeoutDML(queueRawName, jobID, visibilityTimeout)
	result, err := r.querier.ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected == 1, nil
}

func (r *repository) getQueueAttributes(ctx context.Context, queueName string) (*QueueAttributes, error) {
	stmt, args := r.tmpl.NewFindQueueAttributesDML(queueName)
	row := r.querier.QueryRowContext(ctx, stmt, args...)
	var q QueueAttributes
	err := row.Scan(
		&q.Name,
		&q.RawName,
		&q.VisibilityTimeout,
		&q.DelaySeconds,
		&q.MaxReceiveCount,
		&q.DeadLetterTarget,
	)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *repository) createQueueAttributes(ctx context.Context, queueName, queueRawName string, visibilityTimeout, delaySeconds, maxReceiveCount int64, deadLetterTarget *string) error {
	stmt, args := r.tmpl.NewAddQueueAttributesDML(queueName, queueRawName, delaySeconds, maxReceiveCount, visibilityTimeout, deadLetterTarget)
	_, err := r.querier.ExecContext(ctx, stmt, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) updateQueueAttributes(ctx context.Context, queueRawName string,
	visibilityTimeout, delaySeconds, maxReceiveCount *int64, deadLetterTarget *string) (updated bool, err error) {
	stmt, args := r.tmpl.NewUpdateQueueAttributesDML(
		queueRawName,
		visibilityTimeout,
		delaySeconds,
		maxReceiveCount,
		deadLetterTarget)
	result, err := r.querier.ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected == 1, nil
}

func (r *repository) createQueueTable(ctx context.Context, queueRawName string) error {
	stmt := r.tmpl.NewCreateQueueDDL(queueRawName)
	_, err := r.querier.ExecContext(ctx, stmt)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) createQueueAttributesTable(ctx context.Context) error {
	stmt := r.tmpl.NewCreateQueueAttributesDDL()
	_, err := r.querier.ExecContext(ctx, stmt)
	if err != nil {
		return err
	}
	return nil
}
