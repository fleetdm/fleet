package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241122171434, Down_20241122171434)
}

func Up_20241122171434(tx *sql.Tx) error {
	// Duplicate indexes identified after running pt-duplicate-key-checker
	// https://docs.percona.com/percona-toolkit/pt-duplicate-key-checker.html

	// # ########################################################################
	// # fleet.app_config_json
	// # ########################################################################
	//
	// # Uniqueness of id ignored because PRIMARY is a duplicate constraint
	// # id is a duplicate of PRIMARY
	// # Key definitions:
	// #   UNIQUE KEY `id` (`id`)
	// #   PRIMARY KEY (`id`),
	// # Column types:
	// #	  `id` int unsigned not null default '1'
	// # To remove this duplicate index, execute:
	// ALTER TABLE `fleet`.`app_config_json` DROP INDEX `id`;
	//
	// # ########################################################################
	// # fleet.host_users
	// # ########################################################################
	//
	// # idx_uid_username is a duplicate of PRIMARY
	// # Key definitions:
	// #   UNIQUE KEY `idx_uid_username` (`host_id`,`uid`,`username`)
	// #   PRIMARY KEY (`host_id`,`uid`,`username`),
	// # Column types:
	// #	  `host_id` int unsigned not null
	// #	  `uid` int unsigned not null
	// #	  `username` varchar(255) collate utf8mb4_unicode_ci not null
	// # To remove this duplicate index, execute:
	// ALTER TABLE `fleet`.`host_users` DROP INDEX `idx_uid_username`;
	//
	// # ########################################################################
	// # fleet.migration_status_tables
	// # ########################################################################
	//
	// # Uniqueness of id ignored because PRIMARY is a duplicate constraint
	// # id is a duplicate of PRIMARY
	// # Key definitions:
	// #   UNIQUE KEY `id` (`id`)
	// #   PRIMARY KEY (`id`),
	// # Column types:
	// #	  `id` bigint unsigned not null auto_increment
	// # To remove this duplicate index, execute:
	// ALTER TABLE `fleet`.`migration_status_tables` DROP INDEX `id`;
	//
	// # ########################################################################
	// # fleet.policy_membership
	// # ########################################################################
	//
	// # idx_policy_membership_policy_id is a left-prefix of PRIMARY
	// # Key definitions:
	// #   KEY `idx_policy_membership_policy_id` (`policy_id`),
	// #   PRIMARY KEY (`policy_id`,`host_id`),
	// # Column types:
	// #	  `policy_id` int unsigned not null
	// #	  `host_id` int unsigned not null
	// # To remove this duplicate index, execute:
	// ALTER TABLE `fleet`.`policy_membership` DROP INDEX `idx_policy_membership_policy_id`;
	//
	// # ########################################################################
	// # fleet.software
	// # ########################################################################
	//
	// # Key software_listing_idx ends with a prefix of the clustered index
	// # Key definitions:
	// #   KEY `software_listing_idx` (`name`,`id`),
	// #   PRIMARY KEY (`id`),
	// # Column types:
	// #	  `name` varchar(255) collate utf8mb4_unicode_ci not null
	// #	  `id` bigint unsigned not null auto_increment
	// # To shorten this duplicate clustered index, execute:
	// ALTER TABLE `fleet`.`software` DROP INDEX `software_listing_idx`, ADD INDEX `software_listing_idx` (`name`);
	//
	// # ########################################################################
	// # fleet.software_cve
	// # ########################################################################
	//
	// # software_cve_software_id is a left-prefix of unq_software_id_cve
	// # Key definitions:
	// #   KEY `software_cve_software_id` (`software_id`)
	// #   UNIQUE KEY `unq_software_id_cve` (`software_id`,`cve`),
	// # Column types:
	// #	  `software_id` bigint unsigned default null
	// #	  `cve` varchar(255) collate utf8mb4_unicode_ci not null
	// # To remove this duplicate index, execute:
	// ALTER TABLE `fleet`.`software_cve` DROP INDEX `software_cve_software_id`;

	_, err := tx.Exec(
		"ALTER TABLE `app_config_json` DROP INDEX `id`;" +
			"ALTER TABLE `host_users` DROP INDEX `idx_uid_username`;" +
			"ALTER TABLE `migration_status_tables` DROP INDEX `id`;" +
			"ALTER TABLE `policy_membership` DROP INDEX `idx_policy_membership_policy_id`;" +
			"ALTER TABLE `software` DROP INDEX `software_listing_idx`, ADD INDEX `software_listing_idx` (`name`);" +
			"ALTER TABLE `software_cve` DROP INDEX `software_cve_software_id`;",
	)
	if err != nil {
		return fmt.Errorf("failed to remove duplicate indexes: %w", err)
	}
	return nil
}

func Down_20241122171434(tx *sql.Tx) error {
	return nil
}
