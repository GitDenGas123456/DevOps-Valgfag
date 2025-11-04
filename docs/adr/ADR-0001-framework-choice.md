# ADR-0001: Framework Choice

## Context
We must replace the legacy **Flask (Python 2)** application with a modern micro-web framework.  
The legacy stack (Flask + Python 2 + MD5 auth) is outdated, insecure, and does not meet course requirements (OpenAPI, CI/CD pipelines, containerization).

⚠️ Constraints from the course:
- Excluded frameworks: Flask, Express, Spring Boot  
- Should not use full-stack web frameworks like Django, Rails, etc.  
- SQLite will remain the database for the first weeks  
- The focus is DevOps, so the framework should be lightweight and easy to adopt  

**Our new framework must:**
- Be lightweight and easy to adopt  
- Support OpenAPI specification (course requirement)  
- Integrate well with CI/CD pipelines  
- Deploy cleanly to a VM or Azure Web App  
- Use SQLite initially 

---

## Decision
We choose **Go** as our language and runtime, combined with:  
- **gorilla/mux** → a lightweight HTTP router (multiplexer) for mapping requests like `GET /login` or `POST /api/register` to handlers  
- **modernc.org/sqlite** → a pure-Go SQLite driver (no CGO), which makes Docker builds portable and CI/CD pipelines simpler  
- **gorilla/sessions + bcrypt** → for secure authentication and login sessions  
- **html/template + static serving** → for server-side rendering with reusable layouts  
- **env-based configuration** → via `PORT` and `DATABASE_PATH`  

📌 To lower the learning curve for the team (all beginners in Go), we will **keep route definitions in `main.go`** so it’s easy to see the application structure.  
Handler logic may later be refactored into separate files, but the route map will remain in `main.go` for clarity.

This choice provides a minimal but flexible foundation: Go offers performance and concurrency, mux handles routing, and modernc/sqlite provides a simple embedded DB layer.

---

## Considerations

Two team members created early prototypes:  
- **Teammate 1 prototype**: gorilla/mux + mattn/sqlite3 + gorilla/sessions + bcrypt. 
- **Teammate 2 prototype**: gorilla/mux + modernc/sqlite + templates/static + env-config. 

We combined the best parts into our final stack: gorilla/mux for routing, modernc/sqlite for DB portability, gorilla/sessions + bcrypt for auth, and Go templates + static serving for server-side rendering.

### FastAPI (Python 3)
- ✅ Built-in OpenAPI and async support  
- ✅ Strong popularity, great docs  
- ✅ Member has prior experience with framework  
- ❌ More setup required for templating (Jinja2)  
- ❌ Heavier dependency stack, less minimal than required  
- ❌ Anders said no

### Gin (Go)
- ✅ High performance, idiomatic web framework  
- ✅ Good OpenAPI tooling  
- ❌ Slightly steeper learning curve  
- ❌ More “framework-like” than needed  

### Gorilla/mux (Go) — **Chosen**
- ✅ Lightweight and idiomatic router  
- ✅ Easy to read and learn  
- ✅ Pairs cleanly with modernc/sqlite (no CGO)  
- ✅ Flexible: sessions, templates, JSON APIs can be added gradually  
- ❌ In maintenance mode, but still widely used and stable  

---

## Consequences

### Pros
- ✅ CI/CD friendly: one statically compiled binary, easy Docker builds  
- ✅ Secure auth: bcrypt + gorilla/sessions instead of MD5  
- ✅ Simple migration path: reuse search logic and templates from Flask while learning Go  
- ✅ Good exam fit: shows modernization (Python 2 → Go) and industry-relevant practices  
- ✅ SQLite-first: works with lightweight VM setups and demo data; future-ready for Postgres  
- ✅ Route definitions in `main.go` make it easier for beginners to navigate the codebase  

### Cons / Risks
- ⚠️ Learning curve: team must learn Go idioms and template syntax  
- ⚠️ Gorilla/mux is stable but no longer actively developed; long-term, chi or Gin may be better choices  
- ⚠️ OpenAPI requires extra tooling (`go-swagger`, `oapi-codegen`), not built-in like FastAPI  
- ⚠️ If the app grows, `main.go` could become large; we may later refactor routes/handlers into packages