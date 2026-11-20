package running_test

import (
	"context"
	"errors"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/running"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_Registry(t *testing.T) {
	id := domain.DeploymentIDFrom("app", 1)

	t.Run("should not cancel a deployment nobody is processing", func(t *testing.T) {
		r := running.New()

		assert.False(t, r.Cancel(id))
	})

	t.Run("should cancel a tracked deployment with the cancelled cause", func(t *testing.T) {
		r := running.New()
		ctx, release := r.Track(context.Background(), id)
		defer release()

		assert.True(t, r.Cancel(id))
		assert.NotNil(t, ctx.Err())
		assert.True(t, errors.Is(context.Cause(ctx), domain.ErrDeploymentCancelled))
	})

	t.Run("should forget a deployment once released", func(t *testing.T) {
		r := running.New()
		ctx, release := r.Track(context.Background(), id)
		release()

		assert.False(t, r.Cancel(id))
		assert.False(t, errors.Is(context.Cause(ctx), domain.ErrDeploymentCancelled))
	})

	t.Run("should only cancel the given deployment", func(t *testing.T) {
		r := running.New()
		other, releaseOther := r.Track(context.Background(), domain.DeploymentIDFrom("app", 2))
		defer releaseOther()
		_, release := r.Track(context.Background(), id)
		defer release()

		r.Cancel(id)

		assert.Nil(t, other.Err())
	})
}
