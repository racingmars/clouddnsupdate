#!/bin/bash

if [ -z "$USERNAME" ]; then
    echo USERNAME variable must be set.
    exit 1
fi

if [ -z "$PASSWORD" ]; then
    echo PASSWORD variable must be set.
    exit 1
fi

if [ -z "$PROJECT" ]; then
    echo PROJECT variable must be set.
    exit 1
fi

if [ -z "$ZONE" ]; then
    echo ZONE variable must be set.
    exit 1
fi

if [ -z "$DOMAIN" ]; then
    echo DOMAIN variable must be set.
    exit 1
fi

gcloud functions deploy nic \
    --memory=128mb \
    --runtime=go111 \
    --trigger-http \
    --entry-point=Update \
    --set-env-vars=MRW_USERNAME=${USERNAME},MRW_PASSWORD=${PASSWORD},MRW_PROJECT=${PROJECT},MRW_ZONE=${ZONE},MRW_DOMAIN=${DOMAIN}

