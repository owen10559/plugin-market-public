use plugin_market;

create view view_plugins_info as
select `p`.`name` AS `name`,`p`.`description` AS `description`,`u`.`name` AS `author_name` from (`table_plugins` `p` join `table_users` `u`) where ((`p`.`author_id` = `u`.`id`) and (`p`.`delete_at` is NULL))