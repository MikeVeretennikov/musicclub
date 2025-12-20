-- юзеры
CREATE TABLE
    IF NOT EXISTS users (
        id BIGINT PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now () NOT NULL,
        interacted_at TIMESTAMPTZ DEFAULT now () NOT NULL
    );

-- список песен
CREATE TABLE
    IF NOT EXISTS songs (
        id SERIAL PRIMARY KEY,
        title VARCHAR(200) NOT NULL,
        description TEXT,
        link TEXT,
        -- CHECK (
        --     link LIKE '%youtube.com/%'
        --     OR link LIKE '%youtu.be/%'
        --     OR link LIKE '%music.yandex.ru/%'
        -- )
    );

-- кто в какой песне участвует на какой роли
CREATE TABLE
    IF NOT EXISTS song_participations (
        id SERIAL PRIMARY KEY,
        song_id INTEGER NOT NULL REFERENCES songs (id) ON DELETE CASCADE,
        users_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        role TEXT NOT NULL,
        CONSTRAINT unique_song_role_per_users UNIQUE (song_id, users_id, role)
    );

-- сами концерты
CREATE TABLE
    IF NOT EXISTS concerts (
        id SERIAL PRIMARY KEY,
        name VARCHAR(150) NOT NULL,
        date TIMESTAMPTZ DEFAULT now ()
    );

-- треклисты для концертов
CREATE TABLE
    IF NOT EXISTS tracklist_entries (
        id SERIAL PRIMARY KEY,
        concert_id INTEGER NOT NULL REFERENCES concerts (id) ON DELETE CASCADE,
        song_id INTEGER NOT NULL REFERENCES songs (id) ON DELETE CASCADE,
        position INTEGER NOT NULL,
        CONSTRAINT unique_song_position_per_concert UNIQUE (concert_id, position)
    );

-- свободные роли в песнях
CREATE TABLE
    IF NOT EXISTS pending_roles (
        id SERIAL PRIMARY KEY,
        song_id INTEGER NOT NULL REFERENCES songs (id) ON DELETE CASCADE,
        role VARCHAR(200) NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now () NOT NULL
    );

-- логгирование
CREATE TABLE
    IF NOT EXISTS logs (
        id BIGSERIAL PRIMARY KEY,
        -- Time
        created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        -- Who
        users_id BIGINT,
        ip_address INET,
        users_agent TEXT,
        -- Where / what
        app_name TEXT NOT NULL,
        action TEXT NOT NULL,
        endpoint TEXT,
        http_method TEXT,
        http_status INT,
        -- Performance
        duration_ms INT
    );

CREATE INDEX IF NOT EXISTS idx_logs_users_id ON logs (users_id);

CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs (created_at);

CREATE INDEX IF NOT EXISTS idx_logs_app_name ON logs (app_name);

CREATE INDEX IF NOT EXISTS idx_logs_action ON logs (action);