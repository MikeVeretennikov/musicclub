-- Core extensions.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- Enumerations.
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'song_link_type'
) THEN CREATE TYPE song_link_type AS ENUM ('youtube', 'yandex_music', 'soundcloud');
END IF;
END $$;
-- Users and membership.
CREATE TABLE IF NOT EXISTS app_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    tg_user_id BIGINT UNIQUE DEFAULT NULL,
    is_chat_member BOOLEAN NOT NULL DEFAULT FALSE,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Permission flags for current user session snapshot.
CREATE TABLE IF NOT EXISTS user_permissions (
    user_id UUID PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
    edit_own_participation BOOLEAN NOT NULL DEFAULT FALSE,
    edit_any_participation BOOLEAN NOT NULL DEFAULT FALSE,
    edit_own_songs BOOLEAN NOT NULL DEFAULT FALSE,
    edit_any_songs BOOLEAN NOT NULL DEFAULT FALSE,
    edit_events BOOLEAN NOT NULL DEFAULT FALSE,
    edit_tracklists BOOLEAN NOT NULL DEFAULT FALSE
);
-- Songs catalog.
CREATE TABLE IF NOT EXISTS song (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    description TEXT,
    link_kind song_link_type NOT NULL,
    link_url TEXT NOT NULL,
    created_by UUID REFERENCES app_user(id) ON DELETE
    SET NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS song_role (
    song_id UUID NOT NULL REFERENCES song(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    PRIMARY KEY (song_id, role)
);
CREATE TABLE IF NOT EXISTS song_role_assignment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    song_id UUID NOT NULL REFERENCES song(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT song_role_exists FOREIGN KEY (song_id, role) REFERENCES song_role(song_id, role) ON DELETE CASCADE,
    CONSTRAINT song_role_assignment_unique UNIQUE (song_id, role, user_id)
);
CREATE INDEX IF NOT EXISTS idx_song_role_assignment_song ON song_role_assignment (song_id);
CREATE INDEX IF NOT EXISTS idx_song_role_assignment_user ON song_role_assignment (user_id);
-- Events and tracklists.
CREATE TABLE IF NOT EXISTS event (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    start_at TIMESTAMPTZ,
    location TEXT,
    notify_day_before BOOLEAN NOT NULL DEFAULT FALSE,
    notify_hour_before BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES app_user(id) ON DELETE
    SET NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS event_track_item (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES event(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    song_id UUID REFERENCES song(id) ON DELETE
    SET NULL,
        custom_title TEXT,
        custom_artist TEXT,
        CONSTRAINT track_item_requires_title CHECK (
            song_id IS NOT NULL
            OR custom_title IS NOT NULL
        ),
        CONSTRAINT track_item_position UNIQUE (event_id, position),
        CONSTRAINT track_item_identity UNIQUE (event_id, id)
);
-- Performers for events (optionally tied to a specific track item).
CREATE TABLE IF NOT EXISTS event_participant (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    track_item_id UUID,
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_event_participant_event FOREIGN KEY (event_id) REFERENCES event(id) ON DELETE CASCADE,
    CONSTRAINT fk_event_participant_track_item FOREIGN KEY (event_id, track_item_id) REFERENCES event_track_item(event_id, id) ON DELETE CASCADE,
    CONSTRAINT uniq_event_participation UNIQUE (event_id, role, user_id, track_item_id)
);
CREATE INDEX IF NOT EXISTS idx_event_start_at ON event (start_at);
CREATE INDEX IF NOT EXISTS idx_event_participant_event ON event_participant (event_id);
CREATE INDEX IF NOT EXISTS idx_event_participant_user ON event_participant (user_id);
-- Auth with telegram bot
CREATE TABLE IF NOT EXISTS tg_auth_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    tg_user_id BIGINT,
    success BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tg_auth_user_tg_user ON tg_auth_user (tg_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tg_auth_session_user ON tg_auth_user (tg_user_id);
-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);