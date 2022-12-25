-- Table: public.categories

-- DROP TABLE public.categories;

CREATE TABLE public.categories
(
    id integer NOT NULL,
    name text COLLATE pg_catalog."default",
    CONSTRAINT categories_pkey PRIMARY KEY (id)
)

    TABLESPACE pg_default;

ALTER TABLE public.categories
    OWNER to postgres;


-- Table: public.posts

-- DROP TABLE public.posts;

CREATE TABLE public.posts
(
    id integer NOT NULL,
    name text COLLATE pg_catalog."default",
    user_id integer
)

    TABLESPACE pg_default;

ALTER TABLE public.posts
    OWNER to postgres;

-- Table: public.posts_to_categories

-- DROP TABLE public.posts_to_categories;

CREATE TABLE public.posts_to_categories
(
    post_id integer NOT NULL,
    category_id integer NOT NULL
)

    TABLESPACE pg_default;

ALTER TABLE public.posts_to_categories
    OWNER to postgres;


-- Table: public.user

-- DROP TABLE public."user";

CREATE TABLE public."users"
(
    id integer NOT NULL,
    name text COLLATE pg_catalog."default",
    CONSTRAINT user_pkey PRIMARY KEY (id)
)

    TABLESPACE pg_default;

ALTER TABLE public."users"
    OWNER to postgres;


INSERT INTO public.users(
    id, name)
VALUES (1, 'userA'), (2, 'userB'), (3, 'userC');

INSERT INTO public.posts(
    id, name, user_id)
VALUES (1, 'postA', 1), (1, 'postB', 1), (1, 'postC', 2);

INSERT INTO public.categories(
    id, name)
VALUES (1, 'A'), (2, 'B'), (3, 'C'), (4, 'D');

INSERT INTO public.posts_to_categories(
    post_id, category_id)
VALUES (1, 2), (1, 3), (2, 3), (3, 1);