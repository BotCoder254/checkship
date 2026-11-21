package infra_test

import (
	"context"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
)

func Test_ReconfigureOutdatedProxies(t *testing.T) {
	user := authfixture.User()
	automatic := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	assert.Nil(t, automatic.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://apps.example.com")), true)))
	automatic.Configured(automatic.CurrentVersion(), nil, nil)
	manual := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	db := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&automatic, &manual))
	ctx := context.Background()
	logger := must.Panic(log.NewLogger())

	// Targets created now are already at the current revision
	outdated, err := db.TargetsStore.GetOutdatedProxies(ctx, sqlite.ProxyRevision)
	assert.Nil(t, err)
	assert.HasLength(t, 0, outdated)

	// As if configured by a previous version
	assert.Nil(t, db.TargetsStore.MarkProxyRevision(ctx, automatic.ID(), 0))
	assert.Nil(t, db.TargetsStore.MarkProxyRevision(ctx, manual.ID(), 0))

	assert.Nil(t, infra.ReconfigureOutdatedProxies(ctx, db.TargetsStore, logger))

	// Only the target with checkship's proxy is reconfigured, once
	changed := assert.Is[domain.TargetStateChanged](t, db.Dispatcher.Signals()[0])
	assert.Equal(t, automatic.ID(), changed.ID)
	assert.Equal(t, domain.TargetStatusConfiguring, changed.State.Status())
	assert.HasLength(t, 1, db.Dispatcher.Signals())

	outdated, err = db.TargetsStore.GetOutdatedProxies(ctx, sqlite.ProxyRevision)
	assert.Nil(t, err)
	assert.HasLength(t, 0, outdated)

	assert.Nil(t, infra.ReconfigureOutdatedProxies(ctx, db.TargetsStore, logger))
	assert.HasLength(t, 1, db.Dispatcher.Signals())
}
