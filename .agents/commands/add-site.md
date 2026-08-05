# Add site

Add or update a site in the embedded catalog.

1. Read and follow [`.agents/skills/add-catalog-site/SKILL.md`](../skills/add-catalog-site/SKILL.md).
2. Ask for any missing details: site name, home/profile URLs, check type, profile
   wave, and whether self-test usernames are available.
3. Edit `data/sites.json`, then run catalog tests, rebuild, and scoped
   `-validate-catalog -site <Name>` when fixtures exist.
4. Report the site name and validate result (PASS / FAIL / SKIP).
