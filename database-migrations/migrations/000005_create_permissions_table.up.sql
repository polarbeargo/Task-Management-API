CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL
);
