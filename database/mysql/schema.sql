DROP TABLE IF EXISTS subscribers, things, users;

CREATE TABLE users (
    id int unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name varchar(256) NOT NULL,
    email varchar(254) NOT NULL,
    UNIQUE(name),
    UNIQUE(email)
) ENGINE=INNODB;

CREATE TABLE things (
    id int unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
    address varchar(4096) NOT NULL,
    type enum('dir', 'file', 'irods', 'openstack', 's3') NOT NULL,
    created date NOT NULL,
    description text(4096),
    reason tinytext NOT NULL,
    remove date NOT NULL,
    warned1 date,
    warned2 date,
    removed bool NOT NULL default 0,
    UNIQUE(address(170), type)
) ENGINE=INNODB;

CREATE TABLE subscribers (
    user_id int unsigned NOT NULL,
    thing_id int unsigned NOT NULL,
    creator bool NOT NULL default 0,
    PRIMARY KEY (user_id, thing_id),
    KEY (user_id),
    KEY (user_id, creator),
    KEY (thing_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    FOREIGN KEY (thing_id) REFERENCES things(id)
        ON DELETE CASCADE
) ENGINE=INNODB;