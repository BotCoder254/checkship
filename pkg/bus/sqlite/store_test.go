package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
	bussqlite "github.com/checkship-deploy/checkship/pkg/bus/sqlite"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type (
	firstCommand struct {
		bus.Command[bus.UnitType]
		Resource string `json:"resource"`
	}

	secondCommand struct {
		bus.Command[bus.UnitType]
		Resource string `json:"resource"`
	}
)

func (firstCommand) Name_() string        { return "store_test.first" }
func (c firstCommand) ResourceID() string { return c.Resource }

func (secondCommand) Name_() string        { return "store_test.second" }
func (c secondCommand) ResourceID() string { return c.Resource }

func init() {
	bus.Marshallable.Register(firstCommand{}, func(s string) (bus.Request, error) {
		return storage.UnmarshalJSON[firstCommand](s)
	})
	bus.Marshallable.Register(secondCommand{}, func(s string) (bus.Request, error) {
		return storage.UnmarshalJSON[secondCommand](s)
	})
}

func Test_Store(t *testing.T) {
	ctx := context.Background()

	t.Run("should claim a job due in the current millisecond", func(t *testing.T) {
		store, db := openStoreAndDatabase(t)

		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "a"}, bus.CreateOptions{}))

		// SQLite's current millisecond plus 0.6 ms: julianday rounds it up to the next millisecond.
		_, err := db.ExecContext(ctx, `UPDATE scheduled_jobs SET not_before = STRFTIME('%Y-%m-%d %H:%M:%f', 'now') || '6+00:00'`)
		assert.Nil(t, err)

		jobs, err := store.GetNextPendingJobs(ctx, 10)

		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
	})

	t.Run("should claim a job queued a moment ago", func(t *testing.T) {
		store := openStore(t)

		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "a"}, bus.CreateOptions{}))

		jobs, err := store.GetNextPendingJobs(ctx, 10)

		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
	})

	t.Run("should claim no more jobs than the given limit", func(t *testing.T) {
		store := openStore(t)

		for _, resource := range []string{"a", "b", "c"} {
			assert.Nil(t, store.Create(ctx, firstCommand{Resource: resource}, bus.CreateOptions{}))
		}

		first, err := store.GetNextPendingJobs(ctx, 2)
		assert.Nil(t, err)
		assert.HasLength(t, 2, first)

		second, err := store.GetNextPendingJobs(ctx, 2)
		assert.Nil(t, err)
		assert.HasLength(t, 1, second)

		none, err := store.GetNextPendingJobs(ctx, 0)
		assert.Nil(t, err)
		assert.HasLength(t, 0, none)
	})

	t.Run("should only claim jobs with the given message names", func(t *testing.T) {
		store := openStore(t)

		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "a"}, bus.CreateOptions{}))
		assert.Nil(t, store.Create(ctx, secondCommand{Resource: "b"}, bus.CreateOptions{}))

		jobs, err := store.GetNextPendingJobs(ctx, 10, secondCommand{}.Name_())

		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
		assert.Equal(t, secondCommand{}.Name_(), jobs[0].Message().Name_())
	})

	t.Run("should not claim a job while an older job of its group waits for another runner", func(t *testing.T) {
		store := openStore(t)

		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "a"}, bus.CreateOptions{}))
		assert.Nil(t, store.Create(ctx, secondCommand{Resource: "a"}, groupOptions("group")))
		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "b"}, groupOptions("group")))

		// Only "first" is in the group, the job above was queued without one
		jobs, err := store.GetNextPendingJobs(ctx, 10, firstCommand{}.Name_())
		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
		assert.Equal(t, "a", jobs[0].Message().(firstCommand).Resource)

		// The head of "group" is the second command, the first command behind it must wait
		jobs, err = store.GetNextPendingJobs(ctx, 10, firstCommand{}.Name_())
		assert.Nil(t, err)
		assert.HasLength(t, 0, jobs)

		head, err := store.GetNextPendingJobs(ctx, 10, secondCommand{}.Name_())
		assert.Nil(t, err)
		assert.HasLength(t, 1, head)

		// Still waiting while the head runs
		jobs, err = store.GetNextPendingJobs(ctx, 10, firstCommand{}.Name_())
		assert.Nil(t, err)
		assert.HasLength(t, 0, jobs)

		assert.Nil(t, store.Done(ctx, head[0]))

		jobs, err = store.GetNextPendingJobs(ctx, 10, firstCommand{}.Name_())
		assert.Nil(t, err)
		assert.HasLength(t, 1, jobs)
		assert.Equal(t, "b", jobs[0].Message().(firstCommand).Resource)
	})

	t.Run("should give released jobs back to the queue", func(t *testing.T) {
		store := openStore(t)

		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "a"}, bus.CreateOptions{}))
		assert.Nil(t, store.Create(ctx, firstCommand{Resource: "b"}, bus.CreateOptions{}))

		claimed, err := store.GetNextPendingJobs(ctx, 10)
		assert.Nil(t, err)
		assert.HasLength(t, 2, claimed)

		assert.Nil(t, store.Release(ctx))
		assert.Nil(t, store.Release(ctx, claimed[0].ID()))

		again, err := store.GetNextPendingJobs(ctx, 10)
		assert.Nil(t, err)
		assert.HasLength(t, 1, again)
		assert.Equal(t, claimed[0].ID(), again[0].ID())
	})
}

func groupOptions(name string) bus.CreateOptions {
	var opts bus.CreateOptions
	bus.WithGroup(name)(&opts)
	return opts
}

func openStore(t *testing.T) bus.ScheduledJobsStore {
	store, _ := openStoreAndDatabase(t)
	return store
}

func openStoreAndDatabase(t *testing.T) (bus.ScheduledJobsStore, *sqlite.Database) {
	db, err := sqlite.Open("file:"+filepath.Join(t.TempDir(), "test.db")+"?_journal=WAL&_txlock=immediate",
		must.Panic(log.NewLogger()), spy.NewDispatcher())

	assert.Nil(t, err)
	t.Cleanup(func() { _ = db.Close() })

	store := bussqlite.NewScheduledJobsStore(db)
	assert.Nil(t, store.Setup())

	return store, db
}
