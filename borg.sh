#!/bin/sh

NAME=`/bin/date '+%Y-%m-%d'`
REPONAME=$1
BACKUPPATHS=$2

/usr/bin/borg create $REPONAME::$NAME $BACKUPPATHS
/usr/bin/borg prune --keep-daily=7 --keep-weekly=4 --keep-monthly=12 --keep-yearly=10 $REPONAME

