create table if not exists "user"
(
    id   integer not null
        primary key,
    name varchar not null
);

alter table "user"
    owner to postgres;

create table if not exists categories
(
    id   integer not null
        primary key,
    name varchar
);

alter table categories
    owner to postgres;

create table if not exists posts
(
    id          integer not null
        primary key,
    name        varchar,
    user_id     integer
        references "user",
    category_id integer
        references categories
);

alter table posts
    owner to postgres;