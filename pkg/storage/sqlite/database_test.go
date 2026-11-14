package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type (
	entity struct {
		event.Emitter
	}

	somethingHappened struct {
		bus.Notification
	}
)

func (somethingHappened) Name_() string { return "something_happened" }

func Test_AfterCommit(t *testing.T) {
	t.Run("should run the function right away without a transaction", func(t *testing.T) {
		var called bool

		sqlite.AfterCommit(context.Background(), func() { called = true })

		assert.True(t, called)
	})

	t.Run("should run the function once the transaction is committed", func(t *testing.T) {
		var (
			db           = openDatabase(t)
			called       bool
			calledInside bool
		)

		err := write(db, func(ctx context.Context) error {
			sqlite.AfterCommit(ctx, func() { called = true })
			calledInside = called
			return nil
		})

		assert.Nil(t, err)
		assert.False(t, calledInside)
		assert.True(t, called)
	})

	t.Run("should not run the function if the transaction is rolled back", func(t *testing.T) {
		var (
			db          = openDatabase(t)
			called      bool
			expectedErr = errors.New("rollback")
		)

		err := write(db, func(ctx context.Context) error {
			sqlite.AfterCommit(ctx, func() { called = true })
			return expectedErr
		})

		assert.ErrorIs(t, expectedErr, err)
		assert.False(t, called)
	})
}

func openDatabase(t *testing.T) *sqlite.Database {
	db, err := sqlite.Open("file:"+filepath.Join(t.TempDir(), "test.db")+"?_journal=WAL&_txlock=immediate",
		must.Panic(log.NewLogger()), spy.NewDispatcher())

	assert.Nil(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// Writes one entity with one event, calling fn from the event switcher, inside the
// transaction opened by WriteAndDispatch.
func write(db *sqlite.Database, fn func(context.Context) error) error {
	ent := &entity{}
	event.Store(ent, somethingHappened{})

	return sqlite.WriteAndDispatch(db, context.Background(), []*entity{ent}, func(ctx context.Context, _ event.Event) error {
		return fn(ctx)
	})
}
