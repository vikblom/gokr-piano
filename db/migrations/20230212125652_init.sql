-- migrate:up
create table sessions (
    duration    interval,
    date        timestamp not null default now()
);

-- migrate:down
drop table if exists sessions;
