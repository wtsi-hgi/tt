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
    creator_id int unsigned NOT NULL ,
    address varchar(4096) NOT NULL,
    type enum('dir', 'file', 'irods', 'openstack', 's3', 'resource') NOT NULL,
    created bigint  NOT NULL,
    description varchar(4096),
    reason tinytext NOT NULL,
    remove bigint NOT NULL,
    warned1 bigint NOT NULL,
    warned2 bigint NOT NULL,
    removed bool NOT NULL default 0,
    license varchar(4096),
    version tinytext,
    name tinytext, 
    url varchar(200),
    download_method varchar(200),
    request_source varchar(200),
    creation_date bigint NOT NULL, 
    UNIQUE(address(170), type)
) ENGINE=INNODB;

CREATE TABLE subscribers (
    user_id int unsigned NOT NULL,
    thing_id int unsigned NOT NULL,
    PRIMARY KEY (user_id, thing_id),
    KEY (user_id),
    KEY (thing_id),
    FOREIGN KEY(user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    FOREIGN KEY(thing_id) REFERENCES things(id)
        ON DELETE CASCADE
) ENGINE=INNODB;