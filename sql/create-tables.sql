CREATE TABLE IF NOT EXISTS Artist (
  name TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS Album (
  name TEXT,
  artist TEXT,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  CONSTRAINT PK_Album PRIMARY KEY (name, artist)
);

CREATE TABLE IF NOT EXISTS Track (
  id INTEGER IDENTITY(1, 1) PRIMARY KEY,
  name TEXT,
  artist TEXT,
  album TEXT,
  album_position INTEGER,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  FOREIGN KEY (album) REFERENCES Album(name)
);

CREATE TABLE IF NOT EXISTS User (
  id TEXT,
  name TEXT PRIMARY KEY,
  email TEXT
);

CREATE TABLE IF NOT EXISTS Listen (
  id INTEGER IDENTITY(1, 1) PRIMARY KEY,
  date DATETIME,
  track INTEGER,
  user INTEGER,
  FOREIGN KEY (track) REFERENCES Track(id),
  FOREIGN KEY (user) REFERENCES User(id)
);
