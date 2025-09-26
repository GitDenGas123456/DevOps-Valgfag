Database Access

- Raw SQL queries with string interpolation (though you mostly use parameters, which is good).

- No context usage (context.Context) in queries—could help with timeouts.

- Potential unchecked errors (_ = db.QueryRow(...).Scan(...)) in some places.

Templates

- Proper usage of template.Must and template execution.

- Error handling for template execution is present.

HTTP / Gorilla

- Gorilla sessions are used safely, but session errors are ignored (_ = sess.Save(...)).

- Query parameters and form parsing handled correctly, but not much validation beyond empty check.

- No CSRF protection (something to note if adding security linters).

Security

- Passwords hashed with bcrypt.

- Some error handling is minimal (ignoring db.Exec errors in registration).

- No input sanitization for search queries beyond parameterized queries.

Code Organization

- Global variables (db, tmpl, sessionStore)—could benefit from an app context struct.

- Some code duplication (SearchPageHandler vs APISearchHandler).

Go Idioms

- Mixed error handling styles: sometimes _ = ..., sometimes checking error.

- Some functions could return early to reduce nested blocks.



Special Considerations for Gorilla-Web Projects

- Sessions & Cookies: gosec can flag insecure cookie settings. You should review sessionStore usage and session options.

- SQL Queries: gosec and staticcheck can help detect unsafe queries or forgotten error checks.

- Templates: No linter will catch XSS automatically, but staticcheck can detect format string mistakes.

- Form Parsing & Validation: Currently minimal validation; linters won’t replace manual input checks but can highlight unchecked errors.


Choices for project:

- Correctness & performance (staticcheck)

- Style & consistency (revive)

- Security risks (gosec)

- Error handling oversight (errcheck)

- ineffassign and unparam for cleanup and reducing unused code.