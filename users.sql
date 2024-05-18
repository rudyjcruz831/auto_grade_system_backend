

create table users (
    user_id VARCHAR(100) PRIMARY KEY,
    username VARCHAR (50) UNIQUE NOT NULL,
    email VARCHAR (355) UNIQUE NOT NULL,
    best_score INT
);


-- I dont know if this table is need to be a part
-- CREATE TABLE metadata (
--     id SERIAL PRIMARY KEY,
--     user_id VARCHAR(100) NOT NULL REFERENCES users(user_id),
    
-- );
INSERT INTO users (user_id, username, email, best_score) VALUES ('1','narayan', 'narayan.balasubramanian@sjsu.edu', 0);

-- this for seeding user but i need to hash the password
-- INSERT INTO users (user_id, username, email, password, )
-- VALUES (1, 'John', 'Doe');
