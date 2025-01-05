#!/bin/bash
docker compose -f compose-prod.yaml --env-file=.env/production/mail --env-file=.env/production/database up
