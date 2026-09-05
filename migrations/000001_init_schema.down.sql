-- Down Migration: Drop PostgreSQL Schema for Ripple

DROP TABLE IF EXISTS dlq_messages;
DROP TABLE IF EXISTS event_log;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS workflows;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS projects;
