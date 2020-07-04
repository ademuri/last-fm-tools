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
  id INTEGER PRIMARY KEY,
  name TEXT,
  artist TEXT,
  album TEXT,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  FOREIGN KEY (album) REFERENCES Album(name)
);
CREATE INDEX idx_track_by_metadata ON Track (artist, album);

CREATE TABLE User (
  name TEXT PRIMARY KEY,
  email TEXT,
  session_key TEXT
);

CREATE TABLE Listen (
  id INTEGER PRIMARY KEY,
  date DATETIME,
  track INTEGER,
  user TEXT,
  FOREIGN KEY (track) REFERENCES Track(id),
  FOREIGN KEY (user) REFERENCES User(name)
);
CREATE INDEX idx_listen_exact ON Listen (user, date, track);

CREATE TABLE Report (
  name TEXT,
  user TEXT,
  email TEXT,
  sent DATETIME,
  FOREIGN KEY (user) REFERENCES User(name),
  CONSTRAINT PK_Report PRIMARY KEY (name, user, email)
)
