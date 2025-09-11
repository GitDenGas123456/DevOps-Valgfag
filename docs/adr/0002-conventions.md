# ADR-0002: Project Conventions

## Context
To collaborate effectively, we need shared conventions for code, commits, and branches.  
These conventions make it easier to maintain quality, automate CI/CD, and present a professional repo at exam.

#### We could Adopt
We could adopt the following conventions:

### Git / Branching
- `main` branch = always deployable
- Feature branches use prefix: `feat/*`, `fix/*`, `docs/*`, `chore/*`
- Pull Requests required for all merges into `main`

### Commit Messages
- Use [Conventional Commits](https://www.conventionalcommits.org/):
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation only
  - `chore:` for repo setup, config, etc.

### Naming / Casing
- ????? for code, `kebab-case.md` for docs
- Classes: `PascalCase`
- Variables & functions: `snake_case`

