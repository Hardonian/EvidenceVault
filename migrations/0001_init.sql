create extension if not exists pgcrypto;
create table if not exists tenants (id text primary key, name text not null, plan text not null default 'free', created_at timestamptz not null default now());
create table if not exists users (id text primary key, email text not null unique, name text not null, created_at timestamptz not null default now());
create table if not exists memberships (tenant_id text not null references tenants(id), user_id text not null references users(id), role text not null, primary key (tenant_id,user_id));
create table if not exists evidence_items (
id text primary key,
tenant_id text not null references tenants(id),
title text not null,
category text not null,
status text not null check (status in ('active','expiring','expired','missing','archived')),
owner_name text not null default '',
owner_email text not null default '',
issue_date date,
expiry_date date,
reminder_days_before int not null default 30,
source_file_path text not null default '',
notes text not null default '',
created_at timestamptz not null default now(),
updated_at timestamptz not null default now());
create table if not exists evidence_files (id text primary key default gen_random_uuid()::text, tenant_id text not null references tenants(id), evidence_id text not null references evidence_items(id), file_path text not null, content_type text not null default 'application/octet-stream', size_bytes bigint not null default 0, created_at timestamptz not null default now());
create table if not exists reminder_logs (id bigserial primary key, evidence_id text not null references evidence_items(id), tenant_id text not null references tenants(id), reminder_date date not null, channel text not null, status text not null, created_at timestamptz not null default now(), unique(evidence_id, reminder_date, channel));
create table if not exists proofpacks (id text primary key default gen_random_uuid()::text, tenant_id text not null references tenants(id), payload jsonb not null, created_at timestamptz not null default now());
create table if not exists stripe_customers (tenant_id text primary key references tenants(id), stripe_customer_id text not null unique, created_at timestamptz not null default now());
create table if not exists stripe_events (stripe_event_id text primary key, event_type text not null, payload jsonb not null, status text not null default 'received' check (status in ('received','processed','failed')), processed_at timestamptz, created_at timestamptz not null default now());
create table if not exists audit_logs (id bigserial primary key, tenant_id text not null references tenants(id), user_id text not null default '', action text not null, entity_type text not null, entity_id text not null, metadata jsonb not null default '{}'::jsonb, created_at timestamptz not null default now());
