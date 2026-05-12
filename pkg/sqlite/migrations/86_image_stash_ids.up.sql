CREATE TABLE `image_stash_ids` (
  `image_id` integer,
  `endpoint` varchar(255),
  `stash_id` varchar(36),
  `updated_at` datetime not null default '1970-01-01T00:00:00Z',
  foreign key(`image_id`) references `images`(`id`) on delete CASCADE
);
