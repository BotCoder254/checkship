package create_app_test

import (
	"context"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

func Test_CustomDomains(t *testing.T) {
	type world struct {
		ctx       context.Context
		db        *fixture.Context
		automatic domain.Target
		manual    domain.Target
		create    func(create_app.Command) (string, error)
		update    func(update_app.Command) (string, error)
	}

	arrange := func(t *testing.T, reserved monad.Maybe[domain.Url]) world {
		user := authfixture.User()
		automatic := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, automatic.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://apps.example.com")), true)))
		manual := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))

		db := fixture.PrepareDatabase(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&automatic, &manual),
		)
		rules := create_app.DomainsRules{Targets: db.TargetsStore, Apps: db.AppsStore, Reserved: reserved}
		create := create_app.Handler(db.AppsStore, db.AppsStore, rules)
		update := update_app.Handler(db.AppsStore, db.AppsStore, rules)

		return world{
			ctx:       db.Context,
			db:        db,
			automatic: automatic,
			manual:    manual,
			create:    func(c create_app.Command) (string, error) { return create(db.Context, c) },
			update:    func(c update_app.Command) (string, error) { return update(db.Context, c) },
		}
	}

	env := func(target domain.Target, domains map[string][]string) create_app.EnvironmentConfig {
		conf := create_app.EnvironmentConfig{Target: string(target.ID())}
		if domains != nil {
			conf.Domains.Set(domains)
		}
		return conf
	}

	t.Run("should create an application with custom domains", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())

		id, err := w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.automatic, map[string][]string{"web": {"Shop.Example.COM", "www.shop.example.com"}}),
			Staging:    env(w.automatic, nil),
		})

		assert.Nil(t, err)
		app := must.Panic(w.db.AppsStore.GetByID(w.ctx, domain.AppID(id)))
		assert.DeepEqual(t, domain.ServicesDomains{"web": {"shop.example.com", "www.shop.example.com"}}, app.Config(domain.Production).Domains().MustGet())
		assert.False(t, app.Config(domain.Staging).Domains().HasValue())
	})

	t.Run("should refuse invalid, duplicated and reserved domains", func(t *testing.T) {
		w := arrange(t, monad.Value(must.Panic(domain.UrlFrom("https://dashboard.example.com"))))

		_, err := w.create(create_app.Command{
			Name: "shop",
			Production: env(w.automatic, map[string][]string{
				"web": {"https://shop.example.com", "shop.apps.example.com", "dashboard.example.com", "ok.example.com"},
			}),
			Staging: env(w.automatic, map[string][]string{"web": {"ok.example.com"}}),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"production.domains.web.0": domain.ErrInvalidHostname,
			"staging.domains.web.0":    domain.ErrDomainDuplicated,
		}, err)

		_, err = w.create(create_app.Command{
			Name: "shop",
			Production: env(w.automatic, map[string][]string{
				"web": {"shop.apps.example.com", "dashboard.example.com", "ok.example.com"},
			}),
			Staging: env(w.automatic, nil),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"production.domains.web.0": domain.ErrDomainReservedByTarget,
			"production.domains.web.1": domain.ErrDomainReserved,
		}, err)
	})

	t.Run("should refuse custom domains on a target with a manual proxy", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())

		_, err := w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.manual, map[string][]string{"web": {"shop.example.com"}}),
			Staging:    env(w.manual, nil),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"production.domains": domain.ErrCustomDomainsNeedAutomaticProxy,
		}, err)
	})

	t.Run("should limit the number of domains per service", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())
		many := make([]string, domain.MaxDomainsPerService+1)
		for i := range many {
			many[i] = "d" + string(rune('a'+i)) + ".example.com"
		}

		_, err := w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.automatic, map[string][]string{"web": many}),
			Staging:    env(w.automatic, nil),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"production.domains.web": domain.ErrTooManyDomains,
		}, err)
	})

	t.Run("should refuse a domain another application serves, naming it", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())
		_, err := w.create(create_app.Command{
			Name:       "blog",
			Production: env(w.automatic, map[string][]string{"web": {"blog.example.com"}}),
			Staging:    env(w.automatic, nil),
		})
		assert.Nil(t, err)

		_, err = w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.automatic, map[string][]string{"web": {"blog.example.com", "x.apps.example.com"}}),
			Staging:    env(w.automatic, nil),
		})

		// Taken and reserved are reported together
		fields, _ := apperr.As[validate.FieldErrors](err)
		assert.ErrorIs(t, domain.ErrDomainReservedByTarget, fields["production.domains.web.1"])
		taken := fields["production.domains.web.0"]
		assert.ErrorIs(t, domain.ErrDomainAlreadyUsed, taken)
		owner, _ := apperr.As[create_app.DomainOwner](taken)
		assert.Equal(t, "blog", owner.App)
	})

	t.Run("should keep domains when an update leaves them out, and free them on removal", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())
		id, err := w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.automatic, map[string][]string{"web": {"shop.example.com"}}),
			Staging:    env(w.automatic, nil),
		})
		assert.Nil(t, err)

		// Saving variables only keeps the domains
		withVars := update_app.EnvironmentConfig{Target: string(w.automatic.ID())}
		withVars.Vars.Set(map[string]map[string]string{"web": {"DEBUG": "1"}})
		_, err = w.update(update_app.Command{ID: id, Production: monad.Value(withVars)})
		assert.Nil(t, err)
		app := must.Panic(w.db.AppsStore.GetByID(w.ctx, domain.AppID(id)))
		assert.DeepEqual(t, domain.ServicesDomains{"web": {"shop.example.com"}}, app.Config(domain.Production).Domains().MustGet())

		// The application can move its own domain to another environment
		moved := update_app.EnvironmentConfig{Target: string(w.automatic.ID())}
		moved.Domains.Set(map[string][]string{})
		staging := update_app.EnvironmentConfig{Target: string(w.automatic.ID())}
		staging.Domains.Set(map[string][]string{"web": {"shop.example.com"}})
		_, err = w.update(update_app.Command{ID: id, Production: monad.Value(moved), Staging: monad.Value(staging)})
		assert.Nil(t, err)

		// Removing it frees it for another application
		empty := update_app.EnvironmentConfig{Target: string(w.automatic.ID())}
		empty.Domains.Set(map[string][]string{})
		_, err = w.update(update_app.Command{ID: id, Staging: monad.Value(empty)})
		assert.Nil(t, err)

		_, err = w.create(create_app.Command{
			Name:       "other",
			Production: env(w.automatic, map[string][]string{"web": {"shop.example.com"}}),
			Staging:    env(w.automatic, nil),
		})
		assert.Nil(t, err)
	})

	t.Run("should re-check kept domains when the target changes to a manual proxy", func(t *testing.T) {
		w := arrange(t, monad.None[domain.Url]())
		id, err := w.create(create_app.Command{
			Name:       "shop",
			Production: env(w.automatic, map[string][]string{"web": {"shop.example.com"}}),
			Staging:    env(w.automatic, nil),
		})
		assert.Nil(t, err)

		_, err = w.update(update_app.Command{ID: id, Production: monad.Value(update_app.EnvironmentConfig{Target: string(w.manual.ID())})})

		assert.ValidationError(t, validate.FieldErrors{
			"production.domains": domain.ErrCustomDomainsNeedAutomaticProxy,
		}, err)
	})
}
