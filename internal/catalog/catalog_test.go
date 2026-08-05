package catalog_test

import (
	"strings"
	"testing"

	"github.com/deangrant/social-gopher/internal/catalog"
)

func TestLoadValid(t *testing.T) {
	const src = `{
		"sites": [
			{
				"name": "Example",
				"home_url": "https://example.com",
				"profile_url": "https://example.com/{username}",
				"check": { "type": "status" },
				"profile": "default"
			},
			{
				"name": "BodySite",
				"home_url": "https://body.example",
				"profile_url": "https://body.example/{username}",
				"check": { "type": "body", "not_found_text": ["missing"] },
				"profile": "developer"
			},
			{
				"name": "RedirectSite",
				"home_url": "https://redir.example",
				"profile_url": "https://redir.example/{username}",
				"check": { "type": "redirect" },
				"profile": "creative"
			}
		]
	}`
	sites, err := catalog.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(sites) != 3 {
		t.Fatalf("len(sites) = %d, want 3", len(sites))
	}
	if got := sites[0].Check.NotFoundStatus; len(got) != 1 || got[0] != 404 {
		t.Fatalf("default not_found_status = %v, want [404]", got)
	}
	if sites[0].Profile != catalog.ProfileDefault {
		t.Fatalf(
			"profile = %q, want %q",
			sites[0].Profile,
			catalog.ProfileDefault,
		)
	}
}

func TestUsernameMatchesAfterLoad(t *testing.T) {
	const src = `{
		"sites": [{
			"name": "Pat",
			"home_url": "https://example.com",
			"profile_url": "https://example.com/{username}",
			"username_pattern": "^[a-z]+$",
			"check": { "type": "status" },
			"profile": "default"
		}]
	}`
	sites, err := catalog.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	ok, err := sites[0].UsernameMatches("alice")
	if err != nil {
		t.Fatalf("UsernameMatches(alice) error = %v", err)
	}
	if !ok {
		t.Fatal("UsernameMatches(alice) = false, want true")
	}
	ok, err = sites[0].UsernameMatches("Bad_User")
	if err != nil {
		t.Fatalf("UsernameMatches(Bad_User) error = %v", err)
	}
	if ok {
		t.Fatal("UsernameMatches(Bad_User) = true, want false")
	}
}

func TestUsernameMatchesNoPattern(t *testing.T) {
	site := catalog.Site{Name: "X"}
	ok, err := site.UsernameMatches("any")
	if err != nil {
		t.Fatalf("UsernameMatches() error = %v", err)
	}
	if !ok {
		t.Fatal("UsernameMatches() = false, want true")
	}
}

func TestUsernameMatchesHandBuiltInvalid(t *testing.T) {
	site := catalog.Site{UsernamePattern: "["}
	_, err := site.UsernameMatches("x")
	if err == nil {
		t.Fatal("UsernameMatches() error = nil, want error")
	}
}

func TestLoadRejectsInvalid(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "empty",
			src:  `{"sites":[]}`,
		},
		{
			name: "missing profile placeholder",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/u",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "body without text",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"check":{"type":"body"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "duplicate names",
			src: `{
				"sites":[
					{
						"name":"X",
						"home_url":"https://x.test",
						"profile_url":"https://x.test/{username}",
						"check":{"type":"status"},
						"profile":"default"
					},
					{
						"name":"x",
						"home_url":"https://y.test",
						"profile_url":"https://y.test/{username}",
						"check":{"type":"status"},
						"profile":"default"
					}
				]
			}`,
		},
		{
			name: "missing profile",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"check":{"type":"status"}
				}]
			}`,
		},
		{
			name: "invalid profile",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"check":{"type":"status"},
					"profile":"wave-a"
				}]
			}`,
		},
		{
			name: "ftp scheme",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"ftp://x.test",
					"profile_url":"ftp://x.test/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "loopback host",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"http://127.0.0.1",
					"profile_url":"http://127.0.0.1/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "private host",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"http://10.0.0.1",
					"profile_url":"http://10.0.0.1/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "metadata ip",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"http://169.254.169.254",
					"profile_url":"http://169.254.169.254/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "localhost host",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"http://localhost",
					"profile_url":"http://localhost/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "metadata hostname",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"http://metadata.google.internal",
					"profile_url":"http://metadata.google.internal/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "post method",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"method":"POST",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "forbidden header",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"headers":{"Host":"evil.test"},
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "invalid username_pattern",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"username_pattern":"[",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "username_pattern too long",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"username_pattern":"` + strings.Repeat("a", 257) + `",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "unknown top-level field",
			src: `{
				"version":1,
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "unknown site field",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"profile_ur":"https://x.test/{username}",
					"check":{"type":"status"},
					"profile":"default"
				}]
			}`,
		},
		{
			name: "unknown check field",
			src: `{
				"sites":[{
					"name":"X",
					"home_url":"https://x.test",
					"profile_url":"https://x.test/{username}",
					"check":{"type":"status","not_found_stauts":[404]},
					"profile":"default"
				}]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := catalog.Load(strings.NewReader(tt.src)); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}

func TestFilter(t *testing.T) {
	sites := []catalog.Site{
		{Name: "GitHub", NSFW: false},
		{Name: "Adult", NSFW: true},
		{Name: "GitLab", NSFW: false},
	}
	got, err := catalog.Filter(sites, nil, false)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (nsfw excluded)", len(got))
	}

	got, err = catalog.Filter(sites, []string{"github"}, true)
	if err != nil {
		t.Fatalf("Filter(github) error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "GitHub" {
		t.Fatalf("Filter(github) = %+v", got)
	}

	if _, err := catalog.Filter(sites, []string{"nope"}, false); err == nil {
		t.Fatal("unknown site: want error")
	}

	nsfwOnly := []catalog.Site{
		{Name: "Adult", NSFW: true},
		{Name: "Adult2", NSFW: true},
	}
	if _, err := catalog.Filter(nsfwOnly, nil, false); err == nil {
		t.Fatal("all NSFW excluded: want error")
	}
}

func TestFilterByProfile(t *testing.T) {
	sites := []catalog.Site{
		{Name: "Seed", Profile: catalog.ProfileDefault},
		{Name: "Dev", Profile: catalog.ProfileDeveloper},
		{Name: "Create", Profile: catalog.ProfileCreative},
		{Name: "Comm", Profile: catalog.ProfileCommunity},
	}
	tests := []struct {
		name     string
		profiles []string
		want     []string
	}{
		{"default", []string{catalog.ProfileDefault}, []string{"Seed"}},
		{
			"developer",
			[]string{catalog.ProfileDeveloper},
			[]string{"Seed", "Dev"},
		},
		{
			"creative",
			[]string{catalog.ProfileCreative},
			[]string{"Seed", "Create"},
		},
		{
			"community",
			[]string{catalog.ProfileCommunity},
			[]string{"Seed", "Comm"},
		},
		{
			"developer+community",
			[]string{catalog.ProfileDeveloper, catalog.ProfileCommunity},
			[]string{"Seed", "Dev", "Comm"},
		},
		{
			"full",
			[]string{catalog.ProfileFull},
			[]string{"Seed", "Dev", "Create", "Comm"},
		},
		{
			"full with sibling",
			[]string{catalog.ProfileDeveloper, catalog.ProfileFull},
			[]string{"Seed", "Dev", "Create", "Comm"},
		},
		{"empty", nil, []string{"Seed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := catalog.FilterByProfile(sites, tt.profiles)
			if err != nil {
				t.Fatalf("FilterByProfile(%v) error = %v", tt.profiles, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf(
					"len = %d, want %d (%v)",
					len(got),
					len(tt.want),
					names(got),
				)
			}
			for i, name := range tt.want {
				if got[i].Name != name {
					t.Fatalf("got[%d] = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}

	if _, err := catalog.FilterByProfile(sites, []string{"nope"}); err == nil {
		t.Fatal("unknown profile: want error")
	}
}

func names(sites []catalog.Site) []string {
	out := make([]string, len(sites))
	for i, s := range sites {
		out[i] = s.Name
	}
	return out
}

func TestLoadSelfTestFields(t *testing.T) {
	const src = `{
		"sites": [{
			"name": "Example",
			"home_url": "https://example.com",
			"profile_url": "https://example.com/{username}",
			"check": { "type": "status" },
			"username_claimed": "alice",
			"username_unclaimed": "zznobody999",
			"profile": "default"
		}]
	}`
	sites, err := catalog.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if sites[0].UsernameClaimed != "alice" ||
		sites[0].UsernameUnclaimed != "zznobody999" {
		t.Fatalf("self-test fields = %+v", sites[0])
	}
}

func TestLoadSeedFile(t *testing.T) {
	sites, err := catalog.LoadFile("../../data/sites.json")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if len(sites) < 400 {
		t.Fatalf("seed catalog size = %d, want >= 400", len(sites))
	}
	counts := map[string]int{}
	withSelfTest := 0
	for _, s := range sites {
		if s.Profile == "" {
			t.Fatalf("site %q missing profile", s.Name)
		}
		counts[s.Profile]++
		if s.UsernameClaimed != "" && s.UsernameUnclaimed != "" {
			withSelfTest++
		}
	}
	for _, p := range []string{
		catalog.ProfileDefault,
		catalog.ProfileDeveloper,
		catalog.ProfileCreative,
		catalog.ProfileCommunity,
	} {
		if counts[p] == 0 {
			t.Fatalf("profile %q count = 0, want > 0", p)
		}
	}
	if withSelfTest < 30 {
		t.Fatalf("sites with self-test fields = %d, want >= 30", withSelfTest)
	}

	full, err := catalog.FilterByProfile(sites, []string{catalog.ProfileFull})
	if err != nil {
		t.Fatalf("FilterByProfile(full) error = %v", err)
	}
	if len(full) != len(sites) {
		t.Fatalf("full length = %d, want %d", len(full), len(sites))
	}

	def, err := catalog.FilterByProfile(sites, nil)
	if err != nil {
		t.Fatalf("FilterByProfile(nil) error = %v", err)
	}
	if len(def) != counts[catalog.ProfileDefault] {
		t.Fatalf(
			"default filter = %d, want %d",
			len(def),
			counts[catalog.ProfileDefault],
		)
	}

	dev, err := catalog.FilterByProfile(
		sites,
		[]string{catalog.ProfileDeveloper},
	)
	if err != nil {
		t.Fatalf("FilterByProfile(developer) error = %v", err)
	}
	wantDev := counts[catalog.ProfileDefault] + counts[catalog.ProfileDeveloper]
	if len(dev) != wantDev {
		t.Fatalf("developer filter = %d, want %d", len(dev), wantDev)
	}
}
