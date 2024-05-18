

create table users (
    user_id VARCHAR(100) PRIMARY KEY,
	username VARCHAR (50) UNIQUE NOT NULL,
    email VARCHAR (355) UNIQUE NOT NULL
);


-- I dont know if this table is need to be a part
-- CREATE TABLE metadata (
--     id SERIAL PRIMARY KEY,
--     user_id VARCHAR(100) NOT NULL REFERENCES users(user_id),
    
-- );
INSERT INTO users (username, email) VALUES ('admin', 'admin@example.com');

-- this for seeding user but i need to hash the password
-- INSERT INTO users (user_id, username, email, password, )
-- VALUES (1, 'John', 'Doe');