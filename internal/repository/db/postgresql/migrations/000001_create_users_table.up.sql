BEGIN TRANSACTION;

CREATE TABLE users (
                       uuid UUID PRIMARY KEY DEFAULT (gen_random_uuid()),
                       login VARCHAR UNIQUE NOT NULL,
                       password_hash VARCHAR NOT NULL,
                       created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                       updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP)
);

COMMIT;