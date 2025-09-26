# ADR-0003: Repository Layout & Routing Strategy

## Context
We’re all new to Go, so to avoid chaos when we start merging, we need one clear folder structure that’s easy to follow and follows our Go conventions.

## Decision
We go with this structure:

.
├── cmd/
│   └── server/
│       └── main.go              # Entry point, defines all routes
│
├── handlers/                    # Split logic by feature
│   ├── auth.go                  # login, register, logout
│   ├── search.go                # / (search page) + /api/search
│   └── pages.go                 # about page, health check, etc.
│
├── templates/                   # Go HTML templates
│   ├── layout.html
│   ├── search.html
│   ├── login.html
│   ├── register.html
│   └── about.html
│
├── static/                      # CSS/JS/images
│   └── style.css
│
├── internal/
│   └── db/
│       ├── schema.sql           # schema/migrations
│       └── seed.go              # optional bcrypt seeding
│
├── data/
│   └── seed/
│       └── whoknows.db          # demo DB only (ignored in prod)
│
├── docs/
│   └── adr/
│       ├── ADR-0001-framework-choice.md
│       └── ADR-0002-project-conventions.md
│
├── Dockerfile
├── .dockerignore
├── .gitignore
├── go.mod
├── go.sum
└── README.md

**Routing strategy:**
- Keep the list of routes in `cmd/server/main.go` so it’s easy to see the whole app.  
- Actual handler logic lives in `handlers/*.go`.  
- Static files served from `/static/*`.  
- Templates rendered with Go’s `html/template`.  
- Config comes from env vars: `PORT`, `DATABASE_PATH`, `SESSION_KEY`.

## Rationale
- This matches Go’s own guide for `cmd/` and `internal/`.  
- Everyone sees all routes in one file (good for beginners).  
- Avoids a giant single `main.go` because logic is moved out to handlers.  
- Easy to Dockerize and copy the right stuff (templates/static/data).  

## Alternatives Considered
1. **One big main.go**  
   - ✅ Dead simple to start  
   - ❌ Becomes unmaintainable very fast  

2. **Full enterprise layout** (`cmd/`, `internal/`, `pkg/`, services, middleware, etc.)  
   - ✅ Very idiomatic and scalable  
   - ❌ Overkill for this course, too much to learn at once  

## Consequences
### Pros
- Clear entrypoint, easy to navigate  
- Handlers are testable and diffs stay small  
- Portable builds with `modernc.org/sqlite`  
- Demo DB is isolated and won’t sneak into prod  

### Cons
- `main.go` will still grow since all routes are listed there (we’ll live with that for now)  
