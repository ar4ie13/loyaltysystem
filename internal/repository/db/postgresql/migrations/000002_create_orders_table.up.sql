BEGIN TRANSACTION;

CREATE TABLE orders (
                    order_num VARCHAR UNIQUE NOT NULL,
                    status VARCHAR NOT NULL,
                    accrual FLOAT,
                    user_uuid UUID NOT NULL,
                    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                    CONSTRAINT fk_user
                           FOREIGN KEY (user_uuid)
                               REFERENCES users (uuid)


);

COMMIT;