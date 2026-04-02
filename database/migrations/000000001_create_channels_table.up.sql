CREATE TYPE channel_type AS ENUM ('email', 'slack', 'telegram');

CREATE TABLE channels
(
    id          UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    type        channel_type NOT NULL,
    credentials JSONB        NOT NULL,
    name        VARCHAR(255) NOT NULL,
    enabled     BOOLEAN      NOT NULL    DEFAULT true,

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);