# Release Process

Releases happen automatically when you merge to `main`. No version bumping, no manual tags.

## How It Works

Merge a PR to `main` and the workflow:

1. Reads your commits since the last release
2. Figures out the version bump from conventional commit messages
3. Creates a git tag (like `v0.2.0`)
4. Builds binaries for Linux, macOS, and Windows
5. Publishes a GitHub release with changelog

### Why github-tag-action?

It parses conventional commits automatically and generates changelogs. No config needed, just works. It's been around since 2019 with 1.5k+ stars, so it's not going anywhere.

## Conventional Commits

Version bumps follow [Conventional Commits](https://www.conventionalcommits.org/):

### Patch Release (0.0.X)
Bug fixes and small stuff.

```bash
git commit -m "fix: resolve memory leak in storage layer"
git commit -m "fix: correct metric calculation in HyperLogLog"
git commit -m "chore: update dependencies"
```

Result: `v0.1.0` → `v0.1.1`

### Minor Release (0.X.0)
New features.

```bash
git commit -m "feat: add gRPC compression support"
git commit -m "feat: implement metric aggregation endpoint"
git commit -m "feat: add configuration file support"
```

Result: `v0.1.0` → `v0.2.0`

### Major Release (X.0.0)
Breaking changes.

Either add `!` after the type:
```bash
git commit -m "feat!: change API response format to JSON"
```

Or put `BREAKING CHANGE:` in the body:
```bash
git commit -m "feat: restructure storage interface

BREAKING CHANGE: Storage interface now requires context parameter
in all methods. Existing implementations must be updated."
```

Result: `v0.1.0` → `v1.0.0`

## Commit Types

The action recognizes these commit types:

| Type | Description | Version Bump |
|------|-------------|--------------|
| `fix:` | Bug fixes | Patch |
| `feat:` | New features | Minor |
| `feat!:` or `BREAKING CHANGE:` | Breaking changes | Major |
| `docs:` | Documentation only | None |
| `style:` | Code style changes | None |
| `refactor:` | Code refactoring | None |
| `perf:` | Performance improvements | Patch |
| `test:` | Test changes | None |
| `chore:` | Build/tooling changes | None |

## Release Workflow

```bash
# Create feature branch
git checkout -b feat/add-metrics-api

# Make changes and commit
git commit -m "feat: add metrics aggregation API"

# Push and create PR
git push origin feat/add-metrics-api

# Merge to main via GitHub
# → Release v0.2.0 created automatically
```

### Multiple Commits in a PR
The workflow looks at all commits:
- Any `BREAKING CHANGE` → Major bump
- Otherwise any `feat:` → Minor bump  
- Otherwise any `fix:` or `perf:` → Patch bump

### Skipping Releases
Push to `main` without triggering a release:

```bash
git commit -m "docs: update README [skip ci]"
```

Or just edit files in ignored paths (`**.md`, `docs/`, `openspec/`).

## Built Artifacts

Each release includes binaries for:

| Platform | Architecture | Filename |
|----------|-------------|----------|
| Linux | AMD64 | `otlp_cardinality_checker-linux-amd64` |
| Linux | ARM64 | `otlp_cardinality_checker-linux-arm64` |
| macOS | AMD64 (Intel) | `otlp_cardinality_checker-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `otlp_cardinality_checker-darwin-arm64` |
| Windows | AMD64 | `otlp_cardinality_checker-windows-amd64.exe` |

All binaries are built with:
- `-ldflags="-s -w"` for smaller binary size (strips debug info)

## Manual Release (Emergency)

If the automated workflow fails, create a release manually:

```bash
# 1. Tag the commit
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

# 2. Build binaries locally
make release-build  # or see workflow for commands

# 3. Create release via GitHub CLI
gh release create v0.2.0 \
  --title "Release v0.2.0" \
  --notes "Emergency release" \
  dist/*
```

## Initial Release

First merge to `main` creates `v0.1.0`. Want to start at a different version? Tag it manually first:

```bash
git tag -a v0.0.0 -m "Initial version placeholder"
git push origin v0.0.0
```

Next merge bumps from there based on commit type.

## Troubleshooting

### No release created
Check that commits use conventional format (`fix:`, `feat:`, etc.), files aren't in ignored paths, and the workflow didn't error out (Actions tab).

### Wrong version bump
The action picks the highest bump from all commits. Breaking changes win over features, features win over fixes. Squash your PR if you want one clean version bump.

### Workflow fails
- **Go build fails:** Your `go.mod` is broken
- **Permission denied:** Workflow needs `contents: write` permission
- **Tag already exists:** That version was already released

## Workflow Configuration

See [.github/workflows/release.yml](.github/workflows/release.yml) for the complete workflow definition.

Key configuration options:
```yaml
default_bump: patch        # Default if no conventional commits found
release_branches: main     # Only trigger on main branch
tag_prefix: v             # Tags will be v0.1.0, v0.2.0, etc.
```
