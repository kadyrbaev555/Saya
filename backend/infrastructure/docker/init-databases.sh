#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 -v auth_password="$DB_PASSWORD" -v survey_password="$SURVEY_DB_PASSWORD" -v assessment_password="$ASSESSMENT_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE saya_auth LOGIN PASSWORD %L', :'auth_password') WHERE NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='saya_auth')\gexec
SELECT format('CREATE ROLE saya_survey LOGIN PASSWORD %L', :'survey_password') WHERE NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='saya_survey')\gexec
SELECT format('CREATE ROLE saya_assessment LOGIN PASSWORD %L', :'assessment_password') WHERE NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='saya_assessment')\gexec
SELECT 'CREATE DATABASE saya_auth OWNER saya_auth' WHERE NOT EXISTS(SELECT 1 FROM pg_database WHERE datname='saya_auth')\gexec
SELECT 'CREATE DATABASE saya_survey OWNER saya_survey' WHERE NOT EXISTS(SELECT 1 FROM pg_database WHERE datname='saya_survey')\gexec
SELECT 'CREATE DATABASE saya_assessment OWNER saya_assessment' WHERE NOT EXISTS(SELECT 1 FROM pg_database WHERE datname='saya_assessment')\gexec



REVOKE CONNECT ON DATABASE saya_auth FROM PUBLIC;
REVOKE CONNECT ON DATABASE saya_survey FROM PUBLIC;
REVOKE CONNECT ON DATABASE saya_assessment FROM PUBLIC;
SQL
