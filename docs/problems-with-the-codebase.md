Problems with the Legacy Codebase

🔴 Critical
	1.	SQL Injection

Symptom: User input is put straight into SQL queries.
Why bad: Hackers can type things like ' OR '1'='1 to log in without a password or delete tables.

Example (app.py ~line 70):

def get_user_id(username):
    """Convenience method to look up the id for a username."""
    rv = g.db.execute("SELECT id FROM users WHERE username = '%s'" % username).fetchone()
    return rv[0] if rv else None


Fix: Use ? placeholders or an ORM like SQLAlchemy.



2. Weak Password Security

Symptom: Passwords hashed with MD5, no salt.
Why bad: MD5 is old and easy to crack.
Example (app.py ~line 250):

def hash_password(password):
    """Hash a password using md5 encryption."""
    password_bytes = password.encode('utf-8')
    hash_object = hashlib.md5(password_bytes)
    return hash_object.hexdigest()

Fix: Use bcrypt or argon2.


3.	Hard-coded Secrets

Symptom: Secret key is written directly in code.
Why bad: Everyone has the same key, easy to guess.

Example (src/backend/app.py, top of file):

SECRET_KEY = 'development key'
app.secret_key = SECRET_KEY

Fix: Load from environment variables.


🟡 Medium Issues

4. Outdated Stack

Symptom: Code is for Python 2.7 + Flask 0.5.
Why bad: Both are unsupported and insecure.
Evidence: README says to run python2 app.py.

python2 app.py 


Fix: Rewrite in Python 3 with a supported framework.



5. 	One Big File

Symptom: All routes, DB calls, helpers are in app.py.
Why bad: Hard to maintain or test.
Example (app.py register route):

@app.route('/api/register', methods=['POST'])
def api_register():
    g.db.execute("INSERT INTO users (username, email, password) values ('%s', '%s', '%s')" % 
                 (request.form['username'], request.form['email'], hash_password(request.form['password'])))
    g.db.commit()
    return redirect(url_for('login'))

Mitigation: Split into routers, services, models.


🟢 Minor Issues

6.	Restart Script

Symptom: run_forever.sh just restarts app in an endless loop.
Why bad: Hides real crashes, wastes resources.
Example (run_forever.sh):

while true
do
  python2 $PYTHON_SCRIPT_PATH
  sleep 1
done


Mitigation: Use Docker, systemd, or Gunicorn.