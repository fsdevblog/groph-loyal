CREATE TABLE users (
    ID BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    username VARCHAR(15) NOT NULL,
    encrypted_password VARCHAR(60) NOT NULL
);
CREATE UNIQUE INDEX idx_uniq_username ON users(username);