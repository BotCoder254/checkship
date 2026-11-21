// Package cloudflare talks to Cloudflare's API (v4) for Connect Cloudflare: zones, DNS records
// and tunnels, with the token the person created. It only ever calls api.cloudflare.com, never
// follows redirects, and never logs or returns the token.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

const (
	defaultBaseURL = "https://api.cloudflare.com/client/v4"
	maxZonePages   = 20 // 50 per page: 1,000 zones is plenty for one token
	maxBody        = 4 << 20

	accountTokenPrefix = "cfat_" // Cloudflare's token formats: account API token
	userTokenPrefix    = "cfut_" // and user API token
	maxVerifyAccounts  = 5
)

type (
	Client struct {
		baseURL string
		http    *http.Client
	}

	envelope struct {
		Success bool            `json:"success"`
		Errors  []apiError      `json:"errors"`
		Result  json.RawMessage `json:"result"`
		Info    *resultInfo     `json:"result_info"`
	}

	apiError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	resultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	}

	zone struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Account struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"account"`
	}

	record struct {
		ID      string `json:"id,omitempty"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
		TTL     int    `json:"ttl,omitempty"`
		Comment string `json:"comment,omitempty"`
	}

	tunnel struct {
		ID          string            `json:"id"`
		Name        string            `json:"name"`
		Status      string            `json:"status"`
		Connections []json.RawMessage `json:"connections"`
	}
)

// Builds a client for Cloudflare's API. A base URL can be given for tests.
func New(baseURL ...string) *Client {
	base := defaultBaseURL
	if len(baseURL) > 0 && baseURL[0] != "" {
		base = baseURL[0]
	}

	return &Client{
		baseURL: base,
		http: &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Checks the token with Cloudflare. User tokens (cfut_…, and older ones without a prefix) are
// verified on /user/tokens/verify; account tokens (cfat_…) only on their account's
// /accounts/{id}/tokens/verify, and /user answers "Invalid API Token" for them even when valid.
// The account isn't known beforehand, so it comes from the accounts, then the zones, the token
// can see. Anything but not reaching Cloudflare is a refusal.
func (c *Client) VerifyToken(ctx context.Context, token string) error {
	if !strings.HasPrefix(token, accountTokenPrefix) {
		err := c.verify(ctx, token, "/user/tokens/verify")

		if err == nil || errors.Is(err, errUnreachable) || strings.HasPrefix(token, userTokenPrefix) {
			return err
		}
	}

	for _, account := range c.visibleAccounts(ctx, token) {
		err := c.verify(ctx, token, "/accounts/"+url.PathEscape(account)+"/tokens/verify")

		if err == nil || errors.Is(err, errUnreachable) {
			return err
		}
	}

	return domain.ErrCloudflareTokenRefused
}

func (c *Client) verify(ctx context.Context, token, path string) error {
	var result struct {
		Status string `json:"status"`
	}

	// An unknown token gets "400 Invalid API Token", an expired one 401: both are a refusal.
	if _, err := c.call(ctx, token, http.MethodGet, path, nil, nil, &result); err != nil {
		if errors.Is(err, errUnreachable) {
			return err
		}
		return domain.ErrCloudflareTokenRefused
	}

	if result.Status != "active" {
		return domain.ErrCloudflareTokenRefused
	}

	return nil
}

// Accounts the token can see: from /accounts, else from the zones it can read. At most a few.
func (c *Client) visibleAccounts(ctx context.Context, token string) []string {
	var (
		accounts []struct {
			ID string `json:"id"`
		}
		ids  []string
		seen = map[string]bool{}
	)

	add := func(id string) {
		if id != "" && !seen[id] && len(ids) < maxVerifyAccounts {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	if _, err := c.call(ctx, token, http.MethodGet, "/accounts", url.Values{"per_page": {"20"}}, nil, &accounts); err == nil {
		for _, a := range accounts {
			add(a.ID)
		}
	}

	if len(ids) == 0 {
		var zones []zone
		if _, err := c.call(ctx, token, http.MethodGet, "/zones", url.Values{"per_page": {"50"}}, nil, &zones); err == nil {
			for _, z := range zones {
				add(z.Account.ID)
			}
		}
	}

	return ids
}

func (c *Client) Zones(ctx context.Context, token string) ([]domain.CloudflareZone, error) {
	var zones []domain.CloudflareZone

	for page := 1; page <= maxZonePages; page++ {
		var result []zone
		info, err := c.call(ctx, token, http.MethodGet, "/zones", url.Values{
			"per_page": {"50"},
			"page":     {strconv.Itoa(page)},
			"status":   {"active"},
		}, nil, &result)

		if err != nil {
			return nil, permission(err, domain.CloudflarePermissionZoneRead)
		}

		for _, z := range result {
			zones = append(zones, domain.CloudflareZone{ID: z.ID, Name: z.Name, AccountID: z.Account.ID, AccountName: z.Account.Name})
		}

		if info == nil || page >= info.TotalPages {
			break
		}
	}

	return zones, nil
}

func (c *Client) Records(ctx context.Context, token, zoneID, name string) ([]domain.CloudflareRecord, error) {
	return c.records(ctx, token, zoneID, url.Values{"name": {name}, "per_page": {"100"}})
}

func (c *Client) RecordsWithComment(ctx context.Context, token, zoneID, comment string) ([]domain.CloudflareRecord, error) {
	return c.records(ctx, token, zoneID, url.Values{"comment.exact": {comment}, "per_page": {"500"}})
}

func (c *Client) records(ctx context.Context, token, zoneID string, query url.Values) ([]domain.CloudflareRecord, error) {
	var result []record

	if _, err := c.call(ctx, token, http.MethodGet, "/zones/"+url.PathEscape(zoneID)+"/dns_records", query, nil, &result); err != nil {
		return nil, permission(err, domain.CloudflarePermissionDnsEdit)
	}

	records := make([]domain.CloudflareRecord, len(result))

	for i, r := range result {
		records[i] = domain.CloudflareRecord{ID: r.ID, Type: r.Type, Name: r.Name, Content: r.Content, Proxied: r.Proxied, Comment: r.Comment}
	}

	return records, nil
}

func (c *Client) CreateRecord(ctx context.Context, token, zoneID string, r domain.CloudflareRecord) error {
	_, err := c.call(ctx, token, http.MethodPost, "/zones/"+url.PathEscape(zoneID)+"/dns_records", nil, toRecord(r), nil)
	return permission(err, domain.CloudflarePermissionDnsEdit)
}

func (c *Client) UpdateRecord(ctx context.Context, token, zoneID string, r domain.CloudflareRecord) error {
	_, err := c.call(ctx, token, http.MethodPut, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(r.ID), nil, toRecord(r), nil)
	return permission(err, domain.CloudflarePermissionDnsEdit)
}

func (c *Client) DeleteRecord(ctx context.Context, token, zoneID, id string) error {
	_, err := c.call(ctx, token, http.MethodDelete, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(id), nil, nil, nil)
	return permission(err, domain.CloudflarePermissionDnsEdit)
}

func (c *Client) FindTunnel(ctx context.Context, token, account, name string) (monad.Maybe[domain.CloudflareTunnel], error) {
	var result []tunnel

	if _, err := c.call(ctx, token, http.MethodGet, "/accounts/"+url.PathEscape(account)+"/cfd_tunnel", url.Values{
		"name":       {name},
		"is_deleted": {"false"},
	}, nil, &result); err != nil {
		return monad.None[domain.CloudflareTunnel](), permission(err, domain.CloudflarePermissionTunnelEdit)
	}

	for _, t := range result {
		if t.Name == name {
			return monad.Value(toTunnel(t)), nil
		}
	}

	return monad.None[domain.CloudflareTunnel](), nil
}

func (c *Client) CreateTunnel(ctx context.Context, token, account, name string) (domain.CloudflareTunnel, error) {
	var result tunnel

	_, err := c.call(ctx, token, http.MethodPost, "/accounts/"+url.PathEscape(account)+"/cfd_tunnel", nil, map[string]string{
		"name":       name,
		"config_src": "cloudflare", // Managed from Cloudflare: its routes are set through the API
	}, &result)

	return toTunnel(result), permission(err, domain.CloudflarePermissionTunnelEdit)
}

func (c *Client) Tunnel(ctx context.Context, token, account, id string) (domain.CloudflareTunnel, error) {
	var result tunnel

	_, err := c.call(ctx, token, http.MethodGet, "/accounts/"+url.PathEscape(account)+"/cfd_tunnel/"+url.PathEscape(id), nil, nil, &result)

	return toTunnel(result), permission(err, domain.CloudflarePermissionTunnelEdit)
}

func (c *Client) TunnelToken(ctx context.Context, token, account, id string) (string, error) {
	var result string

	_, err := c.call(ctx, token, http.MethodGet, "/accounts/"+url.PathEscape(account)+"/cfd_tunnel/"+url.PathEscape(id)+"/token", nil, nil, &result)

	return result, permission(err, domain.CloudflarePermissionTunnelEdit)
}

func (c *Client) ConfigureTunnel(ctx context.Context, token, account, id string, ingress []domain.CloudflareIngress) error {
	rules := make([]map[string]any, 0, len(ingress)+1)

	for _, rule := range ingress {
		entry := map[string]any{"hostname": rule.Hostname, "service": rule.Service}

		if rule.NoTLSVerify {
			entry["originRequest"] = map[string]any{"noTLSVerify": true, "originServerName": rule.Hostname}
		}

		rules = append(rules, entry)
	}

	// Cloudflare requires a last rule without hostname: anything else gets a 404
	rules = append(rules, map[string]any{"service": "http_status:404"})

	_, err := c.call(ctx, token, http.MethodPut, "/accounts/"+url.PathEscape(account)+"/cfd_tunnel/"+url.PathEscape(id)+"/configurations", nil,
		map[string]any{"config": map[string]any{"ingress": rules}}, nil)

	return permission(err, domain.CloudflarePermissionTunnelEdit)
}

func (c *Client) DeleteTunnel(ctx context.Context, token, account, id string) error {
	path := "/accounts/" + url.PathEscape(account) + "/cfd_tunnel/" + url.PathEscape(id)

	// A tunnel with live connections can't be deleted: drop them first (the connector is
	// stopped before, so they don't come back).
	if _, err := c.call(ctx, token, http.MethodDelete, path+"/connections", nil, nil, nil); err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return permission(err, domain.CloudflarePermissionTunnelEdit)
	}

	_, err := c.call(ctx, token, http.MethodDelete, path, nil, nil, nil)

	if errors.Is(err, apperr.ErrNotFound) {
		return nil
	}

	return permission(err, domain.CloudflarePermissionTunnelEdit)
}

// One API call. Returns the pagination info, decodes the result into out.
func (c *Client) call(ctx context.Context, token, method, path string, query url.Values, body any, out any) (*resultInfo, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var reader io.Reader

	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, apperr.NewWithDetail(domain.ErrCloudflareUnavailable.Error(), errUnreachable)
	}

	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, domain.ErrCloudflareUnavailable
	}

	var env envelope
	_ = json.Unmarshal(raw, &env)

	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return nil, domain.ErrCloudflareTokenRefused
	case res.StatusCode == http.StatusForbidden || hasCode(env.Errors, 9109, 10000):
		message := firstMessage(env.Errors, res.Status)
		if len(env.Errors) > 0 && env.Errors[0].Code != 0 {
			message = fmt.Sprintf("%d: %s", env.Errors[0].Code, message)
		}
		return nil, forbiddenError{message: message}
	case res.StatusCode == http.StatusNotFound:
		return nil, apperr.ErrNotFound
	case res.StatusCode >= 300 || !env.Success:
		return nil, apperr.NewWithDetail(domain.ErrCloudflareUnavailable.Error(), fmt.Errorf("%s", firstMessage(env.Errors, res.Status)))
	}

	if out != nil && len(env.Result) > 0 {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return nil, domain.ErrCloudflareUnavailable
		}
	}

	return env.Info, nil
}

var (
	errForbidden   = errors.New("forbidden") // Cloudflare answers 403 when the token lacks a permission
	errUnreachable = errors.New("could not reach Cloudflare")
)

type missingPermission struct {
	Permission string `json:"permission"`
	Cloudflare string `json:"cloudflare,omitempty"` // What Cloudflare said, to tell what is missing
}

// Cloudflare refused a call for lack of permission, with its own words.
type forbiddenError struct{ message string }

func (e forbiddenError) Error() string        { return e.message }
func (e forbiddenError) Is(target error) bool { return target == errForbidden }

func (m missingPermission) Error() string { return m.Permission }

func permission(err error, name string) error {
	if errors.Is(err, errForbidden) {
		detail := missingPermission{Permission: name}
		var forbidden forbiddenError
		if errors.As(err, &forbidden) {
			detail.Cloudflare = forbidden.message
		}
		return apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), detail)
	}

	return err
}

func hasCode(errs []apiError, codes ...int) bool {
	for _, e := range errs {
		for _, code := range codes {
			if e.Code == code {
				return true
			}
		}
	}

	return false
}

func firstMessage(errs []apiError, fallback string) string {
	if len(errs) > 0 && errs[0].Message != "" {
		return errs[0].Message
	}

	return fallback
}

func toRecord(r domain.CloudflareRecord) record {
	return record{Type: r.Type, Name: r.Name, Content: r.Content, Proxied: r.Proxied, TTL: 1, Comment: r.Comment}
}

func toTunnel(t tunnel) domain.CloudflareTunnel {
	return domain.CloudflareTunnel{ID: t.ID, Name: t.Name, Status: t.Status, Connections: len(t.Connections)}
}
