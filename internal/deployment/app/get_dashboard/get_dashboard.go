// Package get_dashboard holds the read models behind the Overview dashboard: a summary of
// deployments over a range, per-application statistics, and a cursor-paginated activity feed.
//
// Aggregates are computed in Go over a bounded set of rows (at most twice the range, 180 days),
// which keeps day buckets in the viewer's time zone, medians and recovery times simple and
// testable without relying on SQLite date functions.
package get_dashboard

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"sort"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_deployments"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

// Allowed values, enforced by the HTTP layer and again here.
const (
	DefaultDays     = 30
	MaxPerPage      = 25
	DefaultPerPage  = 6
	MaxActivity     = 50
	DefaultActivity = 8
	MaxOffset       = 14 * 60 // minutes, the widest time zone offsets are ±14h
	TopErrors       = 5
)

var (
	AllowedDays = []int{7, 30, 90}

	ErrInvalidCursor = apperr.New("invalid_cursor")
)

type (
	// Filters shared by the summary and the applications statistics.
	Filters struct {
		Days        int                 `json:"-"` // 7, 30 or 90
		Environment monad.Maybe[string] `json:"-"` // production or staging, none for both
		Offset      int                 `json:"-"` // viewer's offset from UTC, in minutes (east positive)
		Until       time.Time           `json:"-"` // now, set by the caller so results are testable
	}

	SummaryQuery struct {
		bus.Query[Summary]
		Filters
	}

	AppsQuery struct {
		bus.Query[storage.Paginated[AppStat]]
		Filters

		Sort    string `json:"-"` // deploys, failures or recent
		Page    int    `json:"-"`
		PerPage int    `json:"-"`
	}

	ActivityQuery struct {
		bus.Query[Activity]

		Environment monad.Maybe[string]    `json:"-"`
		Status      monad.Maybe[uint8]     `json:"-"`
		AppID       monad.Maybe[string]    `json:"-"`
		Since       monad.Maybe[time.Time] `json:"-"` // only deployments requested from then on
		After       monad.Maybe[Cursor]    `json:"-"`
		Limit       int                    `json:"-"`
	}

	// One deployment, as needed to compute the aggregates.
	Row struct {
		AppID       string
		Environment string
		Status      uint8
		ErrCode     monad.Maybe[string]
		StartedAt   monad.Maybe[time.Time]
		FinishedAt  monad.Maybe[time.Time]
		RequestedAt time.Time
	}

	AppRef struct {
		ID   string
		Name string
	}

	Summary struct {
		Days          int               `json:"days"`
		From          time.Time         `json:"from"`
		Until         time.Time         `json:"until"`
		Apps          int               `json:"apps"`
		Current       Period            `json:"current"`
		Previous      Period            `json:"previous"`
		Series        []Day             `json:"series"`
		ByEnvironment []EnvironmentStat `json:"by_environment"`
		Errors        []ErrorStat       `json:"errors"`
		Recovery      RecoveryOf        `json:"recovery"`
	}

	Period struct {
		Deployments    int                  `json:"deployments"`
		Succeeded      int                  `json:"succeeded"`
		Failed         int                  `json:"failed"`
		Other          int                  `json:"other"`           // pending or running
		SuccessRate    monad.Maybe[float64] `json:"success_rate"`    // 0..1 over finished deployments
		MedianDuration monad.Maybe[float64] `json:"median_duration"` // seconds
		MaxDuration    monad.Maybe[float64] `json:"max_duration"`    // seconds
		MedianRecovery monad.Maybe[float64] `json:"median_recovery"` // seconds
		Recoveries     int                  `json:"recoveries"`
	}

	Day struct {
		Date           string               `json:"date"` // YYYY-MM-DD in the viewer's time zone
		Succeeded      int                  `json:"succeeded"`
		Failed         int                  `json:"failed"`
		Other          int                  `json:"other"`
		MedianDuration monad.Maybe[float64] `json:"median_duration"`
		MaxDuration    monad.Maybe[float64] `json:"max_duration"`
	}

	EnvironmentStat struct {
		Environment string `json:"environment"`
		Deployments int    `json:"deployments"`
		Succeeded   int    `json:"succeeded"`
		Failed      int    `json:"failed"`
	}

	ErrorStat struct {
		Code  string `json:"code"`
		Count int    `json:"count"`
	}

	// Which environment the recovery time was measured on.
	RecoveryOf struct {
		Environment string `json:"environment"`
	}

	AppStat struct {
		ID             string                 `json:"id"`
		Name           string                 `json:"name"`
		Deployments    int                    `json:"deployments"`
		Succeeded      int                    `json:"succeeded"`
		Failed         int                    `json:"failed"`
		SuccessRate    monad.Maybe[float64]   `json:"success_rate"`
		LastDeployment monad.Maybe[time.Time] `json:"last_deployment"`
	}

	Activity struct {
		Data       []get_app_deployments.Deployment `json:"data"`
		NextCursor monad.Maybe[string]              `json:"next_cursor"`
	}

	// Position in the activity feed: the last row returned, in feed order
	// (requested_at DESC, app_id DESC, deployment_number DESC).
	Cursor struct {
		RequestedAt time.Time
		AppID       string
		Number      int
	}

	cursorJSON struct {
		T string `json:"t"`
		A string `json:"a"`
		N int    `json:"n"`
	}
)

func (SummaryQuery) Name_() string  { return "deployment.query.get_dashboard_summary" }
func (AppsQuery) Name_() string     { return "deployment.query.get_dashboard_apps" }
func (ActivityQuery) Name_() string { return "deployment.query.get_dashboard_activity" }

// Clamp the filters to the allowed values.
func (f Filters) Normalized() Filters {
	if !slices.Contains(AllowedDays, f.Days) {
		f.Days = DefaultDays
	}
	f.Offset = max(-MaxOffset, min(MaxOffset, f.Offset))
	if env, isSet := f.Environment.TryGet(); isSet && !isEnvironment(env) {
		f.Environment = monad.None[string]()
	}
	if f.Until.IsZero() {
		f.Until = time.Now().UTC()
	}
	return f
}

// Bounds of the current period and the previous one, in UTC. The current period covers
// today and the days-1 days before it in the viewer's time zone.
func (f Filters) Bounds() (previousFrom, from, until time.Time) {
	loc := time.FixedZone("viewer", f.Offset*60)
	local := f.Until.In(loc)
	startOfToday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	from = startOfToday.AddDate(0, 0, -(f.Days - 1))
	previousFrom = from.AddDate(0, 0, -f.Days)
	return previousFrom.UTC(), from.UTC(), f.Until.UTC()
}

func isEnvironment(env string) bool {
	return env == string(domain.Production) || env == string(domain.Staging)
}

// Compute the summary from the rows requested between the previous period's start and until.
func Summarize(f Filters, rows []Row, apps int) Summary {
	f = f.Normalized()
	previousFrom, from, until := f.Bounds()
	loc := time.FixedZone("viewer", f.Offset*60)

	s := Summary{
		Days:  f.Days,
		From:  from,
		Until: until,
		Apps:  apps,
	}

	var (
		current, previous   periodAcc
		days                = make([]dayAcc, f.Days)
		byEnv               = map[string]*EnvironmentStat{}
		errs                = map[string]int{}
		recoveryEnvironment = string(domain.Production)
	)

	if env, isSet := f.Environment.TryGet(); isSet {
		recoveryEnvironment = env
	}
	s.Recovery.Environment = recoveryEnvironment

	sorted := slices.Clone(rows)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].RequestedAt.Before(sorted[j].RequestedAt) })

	for _, r := range sorted {
		if r.RequestedAt.Before(previousFrom) || r.RequestedAt.After(until) {
			continue
		}

		if r.RequestedAt.Before(from) {
			previous.add(r)
			continue
		}

		current.add(r)

		idx := int(r.RequestedAt.In(loc).Sub(from.In(loc)).Hours() / 24)
		if idx >= 0 && idx < len(days) {
			days[idx].add(r)
		}

		env := byEnv[r.Environment]
		if env == nil {
			env = &EnvironmentStat{Environment: r.Environment}
			byEnv[r.Environment] = env
		}
		env.Deployments++
		switch domain.DeploymentStatus(r.Status) {
		case domain.DeploymentStatusSucceeded:
			env.Succeeded++
		case domain.DeploymentStatusFailed:
			env.Failed++
			errs[r.ErrCode.Get("unknown")]++
		}
	}

	// Recovery: from the first failure after a success to the next success, per app.
	failedSince := map[string]time.Time{}
	for _, r := range sorted {
		if r.Environment != recoveryEnvironment {
			continue
		}
		at := r.FinishedAt.Get(r.RequestedAt)
		switch domain.DeploymentStatus(r.Status) {
		case domain.DeploymentStatusFailed:
			if _, failing := failedSince[r.AppID]; !failing {
				failedSince[r.AppID] = at
			}
		case domain.DeploymentStatusSucceeded:
			since, failing := failedSince[r.AppID]
			if !failing {
				continue
			}
			delete(failedSince, r.AppID)
			gap := at.Sub(since).Seconds()
			if gap < 0 || r.RequestedAt.Before(previousFrom) {
				continue
			}
			if r.RequestedAt.Before(from) {
				previous.recoveries = append(previous.recoveries, gap)
			} else {
				current.recoveries = append(current.recoveries, gap)
			}
		}
	}

	s.Current = current.period()
	s.Previous = previous.period()

	s.Series = make([]Day, len(days))
	for i := range days {
		s.Series[i] = days[i].day(from.In(loc).AddDate(0, 0, i).Format(time.DateOnly))
	}

	s.ByEnvironment = make([]EnvironmentStat, 0, len(byEnv))
	for _, env := range []string{string(domain.Production), string(domain.Staging)} {
		if stat, ok := byEnv[env]; ok {
			s.ByEnvironment = append(s.ByEnvironment, *stat)
		}
	}

	s.Errors = make([]ErrorStat, 0, len(errs))
	for code, count := range errs {
		s.Errors = append(s.Errors, ErrorStat{code, count})
	}
	sort.Slice(s.Errors, func(i, j int) bool {
		if s.Errors[i].Count != s.Errors[j].Count {
			return s.Errors[i].Count > s.Errors[j].Count
		}
		return s.Errors[i].Code < s.Errors[j].Code
	})
	if len(s.Errors) > TopErrors {
		s.Errors = s.Errors[:TopErrors]
	}

	return s
}

// Per-application statistics over the current period, sorted and paginated. Every active
// application is listed, even without deployments in the range.
func RankApps(q AppsQuery, apps []AppRef, rows []Row) storage.Paginated[AppStat] {
	q.Filters = q.Filters.Normalized()
	_, from, until := q.Filters.Bounds()

	stats := make(map[string]*AppStat, len(apps))
	list := make([]*AppStat, 0, len(apps))
	for _, a := range apps {
		stat := &AppStat{ID: a.ID, Name: a.Name}
		stats[a.ID] = stat
		list = append(list, stat)
	}

	for _, r := range rows {
		stat := stats[r.AppID]
		if stat == nil || r.RequestedAt.Before(from) || r.RequestedAt.After(until) {
			continue
		}
		stat.Deployments++
		switch domain.DeploymentStatus(r.Status) {
		case domain.DeploymentStatusSucceeded:
			stat.Succeeded++
		case domain.DeploymentStatusFailed:
			stat.Failed++
		}
		if last, isSet := stat.LastDeployment.TryGet(); !isSet || r.RequestedAt.After(last) {
			stat.LastDeployment.Set(r.RequestedAt)
		}
	}

	for _, stat := range list {
		if finished := stat.Succeeded + stat.Failed; finished > 0 {
			stat.SuccessRate.Set(float64(stat.Succeeded) / float64(finished))
		}
	}

	byName := func(a, b *AppStat) bool { return a.Name < b.Name }
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch q.Sort {
		case "failures":
			if a.Failed != b.Failed {
				return a.Failed > b.Failed
			}
			if a.Deployments != b.Deployments {
				return a.Deployments > b.Deployments
			}
		case "recent":
			la, aSet := a.LastDeployment.TryGet()
			lb, bSet := b.LastDeployment.TryGet()
			if aSet != bSet {
				return aSet
			}
			if aSet && !la.Equal(lb) {
				return la.After(lb)
			}
		default:
			if a.Deployments != b.Deployments {
				return a.Deployments > b.Deployments
			}
		}
		return byName(a, b)
	})

	perPage := q.PerPage
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	perPage = min(perPage, MaxPerPage)
	page := max(1, q.Page)
	start := min((page-1)*perPage, len(list))
	end := min(start+perPage, len(list))

	data := make([]AppStat, 0, end-start)
	for _, stat := range list[start:end] {
		data = append(data, *stat)
	}

	return storage.Paginated[AppStat]{
		Data:        data,
		Page:        page,
		IsFirstPage: page == 1,
		IsLastPage:  end >= len(list),
		PerPage:     perPage,
		Total:       len(list),
	}
}

func (c Cursor) Encode() string {
	b, _ := json.Marshal(cursorJSON{T: c.RequestedAt.UTC().Format(time.RFC3339Nano), A: c.AppID, N: c.Number})
	return base64.RawURLEncoding.EncodeToString(b)
}

// Decode an opaque cursor given back by a client. Anything malformed is refused.
func DecodeCursor(value string) (Cursor, error) {
	if value == "" || len(value) > 256 {
		return Cursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	var c cursorJSON
	if err := json.Unmarshal(raw, &c); err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, c.T)
	if err != nil || c.A == "" || len(c.A) > 64 || c.N <= 0 {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{RequestedAt: at.UTC(), AppID: c.A, Number: c.N}, nil
}

type periodAcc struct {
	deployments, succeeded, failed, other int
	durations                             []float64
	recoveries                            []float64
}

func (p *periodAcc) add(r Row) {
	p.deployments++
	switch domain.DeploymentStatus(r.Status) {
	case domain.DeploymentStatusSucceeded:
		p.succeeded++
	case domain.DeploymentStatusFailed:
		p.failed++
	default:
		p.other++
	}
	if d, ok := duration(r); ok {
		p.durations = append(p.durations, d)
	}
}

func (p periodAcc) period() Period {
	out := Period{
		Deployments: p.deployments,
		Succeeded:   p.succeeded,
		Failed:      p.failed,
		Other:       p.other,
		Recoveries:  len(p.recoveries),
	}
	if finished := p.succeeded + p.failed; finished > 0 {
		out.SuccessRate.Set(float64(p.succeeded) / float64(finished))
	}
	if len(p.durations) > 0 {
		out.MedianDuration.Set(Median(p.durations))
		out.MaxDuration.Set(slices.Max(p.durations))
	}
	if len(p.recoveries) > 0 {
		out.MedianRecovery.Set(Median(p.recoveries))
	}
	return out
}

type dayAcc struct {
	succeeded, failed, other int
	durations                []float64
}

func (d *dayAcc) add(r Row) {
	switch domain.DeploymentStatus(r.Status) {
	case domain.DeploymentStatusSucceeded:
		d.succeeded++
	case domain.DeploymentStatusFailed:
		d.failed++
	default:
		d.other++
	}
	if v, ok := duration(r); ok {
		d.durations = append(d.durations, v)
	}
}

func (d dayAcc) day(date string) Day {
	out := Day{Date: date, Succeeded: d.succeeded, Failed: d.failed, Other: d.other}
	if len(d.durations) > 0 {
		out.MedianDuration.Set(Median(d.durations))
		out.MaxDuration.Set(slices.Max(d.durations))
	}
	return out
}

func duration(r Row) (float64, bool) {
	started, hasStarted := r.StartedAt.TryGet()
	finished, hasFinished := r.FinishedAt.TryGet()
	if !hasStarted || !hasFinished || finished.Before(started) {
		return 0, false
	}
	return finished.Sub(started).Seconds(), true
}

// Median of the given values (not modified).
func Median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}
