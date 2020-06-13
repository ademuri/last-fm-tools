-- Copyright 2020 Google LLC
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

CREATE TABLE Artist (
  name TEXT PRIMARY KEY
);

CREATE TABLE Album (
  name TEXT,
  artist TEXT,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  CONSTRAINT PK_Album PRIMARY KEY (artist, name)
);

CREATE TABLE Track (
  id INTEGER IDENTITY(1, 1) PRIMARY KEY,
  name TEXT,
  artist TEXT,
  album TEXT,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  FOREIGN KEY (album) REFERENCES Album(name)
);
CREATE INDEX idx_track_by_metadata ON Track (artist, album, name);

CREATE TABLE User (
  name TEXT PRIMARY KEY,
  email TEXT
);

CREATE TABLE Listen (
  id INTEGER IDENTITY(1, 1) PRIMARY KEY,
  date DATETIME,
  track INTEGER,
  user INTEGER,
  FOREIGN KEY (track) REFERENCES Track(id),
  FOREIGN KEY (user) REFERENCES User(id)
);
CREATE INDEX idx_listen_date ON Listen (user, date);
CREATE INDEX idx_listen_exact ON Listen (user, track, date);
