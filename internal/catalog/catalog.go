// Package catalog loads and validates the site definition catalog.
package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// CheckType identifies how a site probe decides whether a profile exists.
type CheckType string

const (
	CheckStatus   CheckType = "status"
	CheckBody     CheckType = "body"
	CheckRedirect CheckType = "redirect"
)

// Scan profile names (CLI -profile and per-site introducing wave).
const (
	ProfileDefault   = "default"
	ProfileDeveloper = "developer"
	ProfileCreative  = "creative"
	ProfileCommunity = "community"
	ProfileFull      = "full"
)

// Profiles lists allowed -profile flag values in help order.
var Profiles = []string{
	ProfileDefault,
	ProfileDeveloper,
	ProfileCreative,
	ProfileCommunity,
	ProfileFull,
}

// siteProfiles are the allowed per-site introducing profile values.
var siteProfiles = map[string]struct{}{
	ProfileDefault:   {},
	ProfileDeveloper: {},
	ProfileCreative:  {},
	ProfileCommunity: {},
}

// scanProfiles are the allowed -profile flag values.
var scanProfiles = map[string]struct{}{
	ProfileDefault:   {},
	ProfileDeveloper: {},
	ProfileCreative:  {},
	ProfileCommunity: {},
	ProfileFull:      {},
}

// Check describes the probe classification for a site.
type Check struct {
	Type           CheckType `json:"type"`
	NotFoundStatus []int     `json:"not_found_status,omitempty"`
	NotFoundText   []string  `json:"not_found_text,omitempty"`
}

// Site is one searchable network or website.
type Site struct {
	Name              string            `json:"name"`
	HomeURL           string            `json:"home_url"`
	ProfileURL        string            `json:"profile_url"`
	ProbeURL          string            `json:"probe_url,omitempty"`
	Method            string            `json:"method,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	Check             Check             `json:"check"`
	UsernamePattern   string            `json:"username_pattern,omitempty"`
	UsernameClaimed   string            `json:"username_claimed,omitempty"`
	UsernameUnclaimed string            `json:"username_unclaimed,omitempty"`
	Profile           string            `json:"profile"`
	NSFW              bool              `json:"nsfw,omitempty"`
}

type fileSchema struct {
	Sites []Site `json:"sites"`
}

// LoadFile reads and validates a catalog JSON file from path.
func LoadFile(path string) ([]Site, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open catalog: %w", err)
	}
	defer f.Close()
	return Load(f)
}

// Load reads and validates a catalog JSON document from r.
func Load(r io.Reader) ([]Site, error) {
	var doc fileSchema
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}
	if len(doc.Sites) == 0 {
		return nil, fmt.Errorf("catalog has no sites")
	}
	seen := make(map[string]struct{}, len(doc.Sites))
	for i := range doc.Sites {
		if err := validateSite(&doc.Sites[i]); err != nil {
			return nil, fmt.Errorf("site[%d]: %w", i, err)
		}
		key := strings.ToLower(doc.Sites[i].Name)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate site name %q", doc.Sites[i].Name)
		}
		seen[key] = struct{}{}
	}
	return doc.Sites, nil
}

// FilterByProfile returns sites included by the given scan profiles.
// Empty profiles defaults to default only. Any full value returns all sites.
// Otherwise the result is default unioned with each requested profile
// (non-cumulative; selecting creative does not include developer).
func FilterByProfile(sites []Site, profiles []string) ([]Site, error) {
	if len(profiles) == 0 {
		profiles = []string{ProfileDefault}
	}

	allowed := map[string]struct{}{ProfileDefault: {}}
	for _, p := range profiles {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		if _, ok := scanProfiles[p]; !ok {
			return nil, fmt.Errorf(
				"profile must be one of: %s",
				strings.Join(Profiles, ", "),
			)
		}
		if p == ProfileFull {
			out := make([]Site, len(sites))
			copy(out, sites)
			return out, nil
		}
		allowed[p] = struct{}{}
	}

	out := make([]Site, 0, len(sites))
	for _, s := range sites {
		if _, ok := allowed[s.Profile]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// Filter returns sites matching the given options.
// If names is non-empty, only sites whose names match
// (case-insensitive) are kept.
// NSFW sites are omitted unless includeNSFW is true.
func Filter(sites []Site, names []string, includeNSFW bool) ([]Site, error) {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[strings.ToLower(strings.TrimSpace(n))] = struct{}{}
	}

	out := make([]Site, 0, len(sites))
	for _, s := range sites {
		if !includeNSFW && s.NSFW {
			continue
		}
		if len(want) > 0 {
			if _, ok := want[strings.ToLower(s.Name)]; !ok {
				continue
			}
		}
		out = append(out, s)
	}
	if len(want) > 0 && len(out) == 0 {
		return nil, fmt.Errorf("no sites matched filters")
	}
	if len(want) > 0 {
		found := make(map[string]struct{}, len(out))
		for _, s := range out {
			found[strings.ToLower(s.Name)] = struct{}{}
		}
		for n := range want {
			if _, ok := found[n]; !ok {
				return nil, fmt.Errorf("unknown site %q", n)
			}
		}
	}
	return out, nil
}

func validateSite(s *Site) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(s.HomeURL) == "" {
		return fmt.Errorf("home_url is required")
	}
	if strings.TrimSpace(s.ProfileURL) == "" {
		return fmt.Errorf("profile_url is required")
	}
	if !strings.Contains(s.ProfileURL, "{username}") {
		return fmt.Errorf("profile_url must contain {username}")
	}
	if s.ProbeURL != "" && !strings.Contains(s.ProbeURL, "{username}") {
		return fmt.Errorf("probe_url must contain {username}")
	}
	s.Profile = strings.TrimSpace(strings.ToLower(s.Profile))
	if s.Profile == "" {
		return fmt.Errorf("profile is required")
	}
	if _, ok := siteProfiles[s.Profile]; !ok {
		return fmt.Errorf("unknown profile %q", s.Profile)
	}
	switch s.Check.Type {
	case CheckStatus:
		if len(s.Check.NotFoundStatus) == 0 {
			s.Check.NotFoundStatus = []int{404}
		}
	case CheckBody:
		if len(s.Check.NotFoundText) == 0 {
			return fmt.Errorf("check.not_found_text is required for type body")
		}
	case CheckRedirect:
		// no extra fields
	default:
		return fmt.Errorf("unknown check.type %q", s.Check.Type)
	}
	if s.Method != "" {
		m := strings.ToUpper(s.Method)
		switch m {
		case "GET", "HEAD", "POST":
			s.Method = m
		default:
			return fmt.Errorf("unsupported method %q", s.Method)
		}
	}
	return nil
}
