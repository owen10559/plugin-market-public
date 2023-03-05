alter user root@"%" identified with mysql_native_password by "DS9uJM00QdOM";

create user plugin_market@"%" identified with mysql_native_password by "MOiIExrT0x87";
grant all on plugin_market.* to plugin_market@"%";
