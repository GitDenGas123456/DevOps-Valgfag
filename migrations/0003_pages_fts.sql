-- 0003_pages_fts.sql
-- FTS5 virtual table for full-text search on pages

-- Oprettelse af FTS5-tabel knyttet til pages.id via content='pages' og content_rowid='id'
CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts
USING fts5(
  title,
  content,
  url,
  language,
  content='pages',
  content_rowid='id'
);

-- Seed eksisterende data ind i FTS-tabellen
INSERT INTO pages_fts(rowid, title, content, url, language)
SELECT id, title, content, url, language
FROM pages;

-- Trigger: efter INSERT på pages
CREATE TRIGGER IF NOT EXISTS pages_ai
AFTER INSERT ON pages
BEGIN
  INSERT INTO pages_fts(rowid, title, content, url, language)
  VALUES (new.id, new.title, new.content, new.url, new.language);
END;

-- Trigger: efter UPDATE på pages
CREATE TRIGGER IF NOT EXISTS pages_au
AFTER UPDATE ON pages
BEGIN
  -- Slet gammel FTS-række
  DELETE FROM pages_fts WHERE rowid = old.id;
  -- Indsæt opdateret række
  INSERT INTO pages_fts(rowid, title, content, url, language)
  VALUES (new.id, new.title, new.content, new.url, new.language);
END;

-- Trigger: efter DELETE på pages
CREATE TRIGGER IF NOT EXISTS pages_ad
AFTER DELETE ON pages
BEGIN
  DELETE FROM pages_fts WHERE rowid = old.id;
END;
