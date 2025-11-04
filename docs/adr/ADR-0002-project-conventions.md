# ADR-0002: Project Conventions

## Context
To collaborate effectively, we need shared conventions for code, commits, and branches.  
These conventions make it easier to maintain quality, automate CI/CD, and present a professional repo at exam.

#### We consistently

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

We follow [Effective Go](https://go.dev/doc/effective_go) idioms:

- **Packages**: short, all-lowercase, single word.  
  No underscores or mixedCaps (e.g. `auth`, `storage`).

- **Identifiers (variables, functions, types, methods)**: use MixedCaps (CamelCase), no underscores.  
  - Exported (public): start with uppercase, e.g. `User`, `SaveFile`.  
  - Unexported (private): start with lowercase, e.g. `dbConn`, `hashPassword`.

- **Getters/Setters**: do not prefix getters with `Get`.  
  - Use `Owner()` instead of `GetOwner()`.  
  - For setters, use `SetOwner(user)`.

- **Documentation files**: use kebab-case (all lowercase, hyphen separated), e.g. `project-conventions.md`.
