-- Drop and recreate users table
DROP TABLE IF EXISTS users;

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL
);

-- bcrypt hash for "password"
-- You can verify this by running: bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
INSERT INTO users (username, email, password)
  VALUES ('admin', 'keamonk1@stud.kea.dk', '$2a$10$wHgFJ4EvAty4/nXZ7LxROulqfEUvvVdHRK3g.B40VgTfZ2.PU6vSm');

-- Drop and recreate pages table
DROP TABLE IF EXISTS pages;

CREATE TABLE IF NOT EXISTS pages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT UNIQUE,
  url TEXT UNIQUE,
  language TEXT NOT NULL CHECK(language IN ('en', 'da')) DEFAULT 'en',
  last_updated TIMESTAMP,
  content TEXT NOT NULL
);

-- Sample content
INSERT INTO pages (title, url, language, last_updated, content)
VALUES
  ('Welcome', '/welcome', 'en', CURRENT_TIMESTAMP, 'Welcome to WhoKnows, the best search engine!'),
  ('About Us', '/about', 'en', CURRENT_TIMESTAMP, 'We intend to build the world’s best search engine.');

