#!/bin/bash

PUID=${PUID:-911}
PGID=${PGID:-911}

groupmod -o -g "$PGID" abc
usermod -o -u "$PUID" abc

echo '
-------------------------------------
GID/UID
-------------------------------------'
echo "
User uid:    $(id -u abc)
User gid:    $(id -g abc)
-------------------------------------
"

chown abc:abc /app
chown abc:abc /config

echo "Booting app"
cd /app && ./main