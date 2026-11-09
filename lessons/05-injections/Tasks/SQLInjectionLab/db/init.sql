CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    private_note TEXT NOT NULL
);

-- Пароли хранятся в открытом виде намеренно: это учебный антипример.
INSERT INTO users (login, password, display_name, private_note) VALUES
    ('student', 'ctf-student-2026', 'Студент', 'Ваш профиль создан для практики.'),
    ('admin', md5(random()::text || clock_timestamp()::text), 'Администратор',
     'CTF{sql_queries_must_be_parameterized}');
