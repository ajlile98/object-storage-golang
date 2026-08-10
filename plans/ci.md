Here's a pragmatic plan, scoped to what this project actually needs today. (Not making any changes — just the plan; I'll scaffold files when you ask.)

What you actually need now: CI on every push/PR
One GitHub Actions workflow that runs your existing checks. This is 90% of the value.

.github/workflows/ci.yml — triggers on push and pull_request, single job:

actions/checkout
actions/setup-go (pin your Go version; enable built-in module caching)
gofmt -l . → fail if it prints anything (catches unformatted code)
go vet Personal.
go build Personal.
go test -race -count=1 Personal.
That's the whole thing. It mirrors what you already run locally, so no new concepts — just enforced automatically. The -race flag is worth it since you have concurrent HTTP + SQLite access.

Nice-to-have, low effort (add when the above is green)
golangci-lint via its official action. It bundles staticcheck, errcheck, etc. Start with defaults; don't hand-tune a config until it nags you. This is the one "extra" I'd actually recommend for a learning project — it teaches idiomatic Go.
Branch protection on main: require the CI check to pass before merge. This is a repo setting, not a file, but it's what makes CI meaningful.
Dependabot (.github/dependabot.yml) for Go modules + Actions — one small file, keeps deps current with near-zero effort.
Releases: skip for now
You're not deploying anywhere, so a release pipeline is solving a problem you don't have. Deferring it is the right call. When you do want artifacts:

Tag-based trigger (on: push: tags: ['v*']).
GoReleaser builds cross-platform binaries + a GitHub Release from a tag. It's the standard, but it's a real config surface — don't add it until you have something to distribute.
If/when you containerize, add a step to build and push an image (GHCR) on tag. Also later.
What to consciously NOT do
No deploy/CD stage (nowhere to deploy).
No matrix of OS/Go versions (one version is fine for a personal project).
No coverage gates/badges, no code-signing, no multi-stage promotion — all overkill here.
Suggested order
ci.yml with fmt/vet/build/test.
Turn on branch protection requiring that check.
Add golangci-lint + Dependabot.
(Later, only when deploying) GoReleaser and/or container build on tags.