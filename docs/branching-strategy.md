# Branching Strategy

This repository follows a lightweight GitFlow-inspired workflow that keeps `main` ready for release, collects completed work on a shared integration branch, and isolates in-progress work on short-lived topic branches. The conventions below extend the rules from [ADR-0002](adr/0002-conventions.md).

## Branch Roles
- **`main`** - Production-ready history. Only updated from `develop` after review and release approval. CI must be green before merging.
- **`develop`** - Integration branch for the next release. All feature, bugfix, and documentation work merges here first via pull requests.
- **Topic branches** - Short-lived branches created from `develop`:
  - Features: `feat/<summary>`
  - Bug fixes: `fix/<summary>`
  - Documentation: `docs/<summary>`
  - Chores/refactors: `chore/<summary>`
- **Hotfix branches** - Rare emergency fixes forked from `main` when a production issue cannot wait for the regular cycle. Use `hotfix/<summary>` and merge back into both `main` and `develop`.

## Working on Changes
1. Sync local history and check out `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   ```
2. Create a topic branch using the appropriate prefix:
   ```bash
   git checkout -b feat/add-search-filter
   ```
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/) as defined in ADR-0002.
4. Push the branch and open a pull request targeting `develop`.
5. Address review feedback, ensure checks pass, then squash/merge the PR.

## Promoting Work to `main`
1. When `develop` is stable and ready to release, create a release pull request:
   ```bash
   git checkout develop
   git pull
   git checkout -b release/2024-05-30
   git push -u origin release/2024-05-30
   ```
2. Open a PR from the release branch (or directly from `develop`) into `main`.
3. After approval and successful CI, merge into `main` using a merge commit to preserve history.
4. Tag the release (e.g., `v2024.05.30`) and back-merge `main` into `develop` to keep both branches aligned:
   ```bash
   git checkout main
   git pull
   git tag v2024.05.30
   git push origin main --tags
   git checkout develop
   git merge main
   git push origin develop
   ```

## Additional Notes
- Keep topic branches focused and short-lived; delete them on the remote after merging.
- Rebase topic branches on top of `develop` to reduce merge conflicts, but never rewrite history of shared branches (`main`, `develop`).
- Document any release notes or migration steps in the PR that promotes `develop` to `main`.

