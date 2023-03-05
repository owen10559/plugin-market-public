mysql -u root < /docker-entrypoint-initdb.d/scripts/db_init.sql

if [ -f /root/data/backup.sql ]
then
    mysql -u root plugin_market < /root/data/backup.sql
else
    mysql -u root plugin_market < /docker-entrypoint-initdb.d/scripts/table_init.sql
fi

mysql -u root < /docker-entrypoint-initdb.d/scripts/view_init.sql
mysql -u root < /docker-entrypoint-initdb.d/scripts/user_init.sql
# sh /docker-entrypoint-initdb.d/scripts/auto_backup.sh &
