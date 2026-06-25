package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Entitlement is a catalog row exposed to the UI via Wails bindings.
type Entitlement struct {
	EntitlementID string `json:"entitlement_id,omitempty"`
	Key           string `json:"key"`
	FriendlyName  string `json:"friendly_name"`
	Description   string `json:"description,omitempty"`
	Scope         string `json:"scope"`
	ValueType     string `json:"value_type"`
	Unit          string `json:"unit,omitempty"`
	DefaultValue  string `json:"default_value"`
	Category      string `json:"category,omitempty"`
	Deprecated    bool   `json:"deprecated"`
}

// PlanEntitlementRef binds an entitlement key to a plan value (JSON text).
type PlanEntitlementRef struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value"`
}

// Plan is a priced bundle from GET /admin/plans.
type Plan struct {
	PlanID           string               `json:"plan_id,omitempty"`
	Code             string               `json:"code"`
	FriendlyName     string               `json:"friendly_name"`
	Description      string               `json:"description,omitempty"`
	SubjectType      string               `json:"subject_type"`
	PriceAmountMinor int64                `json:"price_amount_minor"`
	PriceCurrency    string               `json:"price_currency"`
	BillingInterval  string               `json:"billing_interval"`
	TrialDays        int                  `json:"trial_days"`
	Status           string               `json:"status"`
	Entitlements     []PlanEntitlementRef `json:"entitlements,omitempty"`
}

type entitlementAPI struct {
	EntitlementID string          `json:"entitlement_id"`
	Key           string          `json:"key"`
	FriendlyName  string          `json:"friendly_name"`
	Description   string          `json:"description"`
	Scope         string          `json:"scope"`
	ValueType     string          `json:"value_type"`
	Unit          string          `json:"unit"`
	DefaultValue  json.RawMessage `json:"default_value"`
	Category      string          `json:"category"`
	Deprecated    bool            `json:"deprecated"`
}

type planAPI struct {
	PlanID           string                 `json:"plan_id"`
	Code             string                 `json:"code"`
	FriendlyName     string                 `json:"friendly_name"`
	Description      string                 `json:"description"`
	SubjectType      string                 `json:"subject_type"`
	PriceAmountMinor int64                  `json:"price_amount_minor"`
	PriceCurrency    string                 `json:"price_currency"`
	BillingInterval  string                 `json:"billing_interval"`
	TrialDays        int                  `json:"trial_days"`
	Status           string                 `json:"status"`
	Entitlements     []planEntitlementAPI   `json:"entitlements"`
}

type planEntitlementAPI struct {
	Key   string          `json:"key,omitempty"`
	Value json.RawMessage `json:"value"`
}

// createEntitlementRequest is POST /admin/entitlements body (no server-assigned or read-only fields).
type createEntitlementRequest struct {
	Key          string          `json:"key"`
	FriendlyName string          `json:"friendly_name"`
	Description  string          `json:"description,omitempty"`
	Scope        string          `json:"scope"`
	ValueType    string          `json:"value_type"`
	Unit         string          `json:"unit,omitempty"`
	DefaultValue json.RawMessage `json:"default_value"`
	Category     string          `json:"category,omitempty"`
}

// EntitlementPatch updates mutable entitlement fields (PATCH /admin/entitlements/{id}).
type EntitlementPatch struct {
	FriendlyName string `json:"friendly_name"`
	Description  string `json:"description,omitempty"`
	DefaultValue string `json:"default_value"`
	Category     string `json:"category,omitempty"`
	Deprecated   bool   `json:"deprecated"`
}

func rawJSONText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}

func parseDefaultValueJSON(raw string) (json.RawMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("default_value is required")
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("default_value must be valid JSON")
	}
	return json.RawMessage(raw), nil
}

func entitlementFromAPI(e entitlementAPI) Entitlement {
	return Entitlement{
		EntitlementID: e.EntitlementID,
		Key:           e.Key,
		FriendlyName:  e.FriendlyName,
		Description:   e.Description,
		Scope:         e.Scope,
		ValueType:     e.ValueType,
		Unit:          e.Unit,
		DefaultValue:  rawJSONText(e.DefaultValue),
		Category:      e.Category,
		Deprecated:    e.Deprecated,
	}
}

func planFromAPI(p planAPI) Plan {
	refs := make([]PlanEntitlementRef, 0, len(p.Entitlements))
	for _, ref := range p.Entitlements {
		refs = append(refs, PlanEntitlementRef{
			Key:   ref.Key,
			Value: rawJSONText(ref.Value),
		})
	}
	return Plan{
		PlanID:           p.PlanID,
		Code:             p.Code,
		FriendlyName:     p.FriendlyName,
		Description:      p.Description,
		SubjectType:      p.SubjectType,
		PriceAmountMinor: p.PriceAmountMinor,
		PriceCurrency:    p.PriceCurrency,
		BillingInterval:  p.BillingInterval,
		TrialDays:        p.TrialDays,
		Status:           p.Status,
		Entitlements:     refs,
	}
}

func (c *Client) ListEntitlements() ([]Entitlement, error) {
	data, status, err := c.do(http.MethodGet, "/admin/entitlements", nil, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp struct {
		Entitlements []entitlementAPI `json:"entitlements"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode entitlements: %w", err)
	}
	if resp.Entitlements == nil {
		return []Entitlement{}, nil
	}
	out := make([]Entitlement, len(resp.Entitlements))
	for i, e := range resp.Entitlements {
		out[i] = entitlementFromAPI(e)
	}
	return out, nil
}

// CreateEntitlement adds an entitlement to the catalog.
func (c *Client) CreateEntitlement(ent Entitlement) (*Entitlement, error) {
	defaultValue, err := parseDefaultValueJSON(ent.DefaultValue)
	if err != nil {
		return nil, err
	}
	body := createEntitlementRequest{
		Key:          ent.Key,
		FriendlyName: ent.FriendlyName,
		Description:  ent.Description,
		Scope:        ent.Scope,
		ValueType:    ent.ValueType,
		Unit:         ent.Unit,
		DefaultValue: defaultValue,
		Category:     ent.Category,
	}
	data, status, err := c.do(http.MethodPost, "/admin/entitlements", body, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, parseError(status, data)
	}
	var api entitlementAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("decode entitlement: %w", err)
	}
	out := entitlementFromAPI(api)
	return &out, nil
}

// PatchEntitlement updates mutable entitlement fields.
func (c *Client) PatchEntitlement(entitlementID string, patch EntitlementPatch) (*Entitlement, error) {
	defaultValue, err := parseDefaultValueJSON(patch.DefaultValue)
	if err != nil {
		return nil, err
	}
	body := struct {
		FriendlyName string          `json:"friendly_name"`
		Description  string          `json:"description,omitempty"`
		DefaultValue json.RawMessage `json:"default_value"`
		Category     string          `json:"category,omitempty"`
		Deprecated   bool            `json:"deprecated"`
	}{
		FriendlyName: patch.FriendlyName,
		Description:  patch.Description,
		DefaultValue: defaultValue,
		Category:     patch.Category,
		Deprecated:   patch.Deprecated,
	}
	path := "/admin/entitlements/" + entitlementID
	data, status, err := c.do(http.MethodPatch, path, body, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var api entitlementAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("decode entitlement: %w", err)
	}
	out := entitlementFromAPI(api)
	return &out, nil
}

// ListPlans returns plan definitions.
func (c *Client) ListPlans() ([]Plan, error) {
	data, status, err := c.do(http.MethodGet, "/admin/plans", nil, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp struct {
		Plans []planAPI `json:"plans"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode plans: %w", err)
	}
	if resp.Plans == nil {
		return []Plan{}, nil
	}
	out := make([]Plan, len(resp.Plans))
	for i, p := range resp.Plans {
		out[i] = planFromAPI(p)
	}
	return out, nil
}

type createPlanEntitlementRef struct {
	Key   string          `json:"key,omitempty"`
	Value json.RawMessage `json:"value"`
}

type createPlanRequest struct {
	Code             string                     `json:"code"`
	FriendlyName     string                     `json:"friendly_name"`
	Description      string                     `json:"description,omitempty"`
	SubjectType      string                     `json:"subject_type"`
	PriceAmountMinor int64                      `json:"price_amount_minor"`
	PriceCurrency    string                     `json:"price_currency"`
	BillingInterval  string                     `json:"billing_interval"`
	TrialDays        int                        `json:"trial_days"`
	Status           string                     `json:"status"`
	Entitlements     []createPlanEntitlementRef `json:"entitlements"`
}

// CreatePlan adds a plan definition.
func (c *Client) CreatePlan(plan Plan) (*Plan, error) {
	refs := make([]createPlanEntitlementRef, 0, len(plan.Entitlements))
	for _, ref := range plan.Entitlements {
		raw, err := parseDefaultValueJSON(ref.Value)
		if err != nil {
			return nil, fmt.Errorf("entitlement value for %s: %w", ref.Key, err)
		}
		refs = append(refs, createPlanEntitlementRef{Key: ref.Key, Value: raw})
	}
	body := createPlanRequest{
		Code:             plan.Code,
		FriendlyName:     plan.FriendlyName,
		Description:      plan.Description,
		SubjectType:      plan.SubjectType,
		PriceAmountMinor: plan.PriceAmountMinor,
		PriceCurrency:    plan.PriceCurrency,
		BillingInterval:  plan.BillingInterval,
		TrialDays:        plan.TrialDays,
		Status:           plan.Status,
		Entitlements:     refs,
	}
	data, status, err := c.do(http.MethodPost, "/admin/plans", body, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, parseError(status, data)
	}
	var api planAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("decode plan: %w", err)
	}
	out := planFromAPI(api)
	return &out, nil
}

// PlanPatch updates mutable plan fields (PATCH /admin/plans/{id}).
type PlanPatch struct {
	FriendlyName     string               `json:"friendly_name"`
	Description      string               `json:"description,omitempty"`
	PriceAmountMinor int64                `json:"price_amount_minor"`
	PriceCurrency    string               `json:"price_currency"`
	BillingInterval  string               `json:"billing_interval"`
	TrialDays        int                  `json:"trial_days"`
	Status           string               `json:"status"`
	Entitlements     []PlanEntitlementRef `json:"entitlements"`
}

type patchPlanBody struct {
	FriendlyName     string                     `json:"friendly_name"`
	Description      string                     `json:"description,omitempty"`
	PriceAmountMinor int64                      `json:"price_amount_minor"`
	PriceCurrency    string                     `json:"price_currency"`
	BillingInterval  string                     `json:"billing_interval"`
	TrialDays        int                      `json:"trial_days"`
	Status           string                     `json:"status"`
	Entitlements     []createPlanEntitlementRef `json:"entitlements"`
}

// PatchPlan updates mutable plan fields.
func (c *Client) PatchPlan(planID string, patch PlanPatch) (*Plan, error) {
	refs := make([]createPlanEntitlementRef, 0, len(patch.Entitlements))
	for _, ref := range patch.Entitlements {
		raw, err := parseDefaultValueJSON(ref.Value)
		if err != nil {
			return nil, fmt.Errorf("entitlement value for %s: %w", ref.Key, err)
		}
		refs = append(refs, createPlanEntitlementRef{Key: ref.Key, Value: raw})
	}
	body := patchPlanBody{
		FriendlyName:     patch.FriendlyName,
		Description:      patch.Description,
		PriceAmountMinor: patch.PriceAmountMinor,
		PriceCurrency:    patch.PriceCurrency,
		BillingInterval:  patch.BillingInterval,
		TrialDays:        patch.TrialDays,
		Status:           patch.Status,
		Entitlements:     refs,
	}
	path := "/admin/plans/" + planID
	data, status, err := c.do(http.MethodPatch, path, body, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var api planAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("decode plan: %w", err)
	}
	out := planFromAPI(api)
	return &out, nil
}
