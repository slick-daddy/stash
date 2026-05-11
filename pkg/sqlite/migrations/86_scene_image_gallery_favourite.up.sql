ALTER TABLE `scenes` ADD COLUMN `favourite` boolean not null default '0';
ALTER TABLE `images` ADD COLUMN `favourite` boolean not null default '0';
ALTER TABLE `galleries` ADD COLUMN `favourite` boolean not null default '0';