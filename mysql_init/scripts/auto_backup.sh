while true
do
    mysqldump -u root plugin_market > /root/data/backup.sql
    sleep 1h
done
