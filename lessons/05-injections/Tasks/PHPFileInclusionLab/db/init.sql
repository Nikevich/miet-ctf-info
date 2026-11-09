CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL
);

CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    title VARCHAR(128) NOT NULL,
    category VARCHAR(32) NOT NULL,
    points INTEGER NOT NULL
);

INSERT INTO users (login, password, display_name) VALUES
    ('student', 'ctf-student-2026', 'Студент');

INSERT INTO tasks (title, category, points) VALUES
    ('Cookie collector', 'web', 100),
    ('Header hunt', 'web', 150),
    ('Classic cipher', 'crypto', 100),
    ('Packet trail', 'forensics', 200);
