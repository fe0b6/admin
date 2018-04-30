#!/bin/sh

DATE=`date '+%Y-%m-%d_%H:%M:%S'`

USER="root"
PATH=""
CERTDIR=""
HOST=""

DATABASES="db1 db2"

for DNNAME in ${DATABASES}; do
  /usr/local/bin/cockroach dump $DNNAME --certs-dir=$CERTDIR --user=$USER --host=$HOST | /bin/gzip -9 > $PATH$DNNAME.$DATE.sql.gz
done
