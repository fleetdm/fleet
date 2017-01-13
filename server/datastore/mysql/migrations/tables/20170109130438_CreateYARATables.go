package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170109130438, Down_20170109130438)
}

func Up_20170109130438(tx *sql.Tx) error {
	sqlStatement :=
		"CREATE TABLE `yara_signatures` ( " +
			"  `id` int(11) NOT NULL AUTO_INCREMENT, " +
			"  `signature_name` varchar(128) NOT NULL DEFAULT '', " +
			"  PRIMARY KEY (`id`), " +
			"  UNIQUE KEY `idx_yara_signatures_unique_name` (`signature_name`) USING BTREE " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8; "
	if _, err := tx.Exec(sqlStatement); err != nil {
		return err
	}

	sqlStatement =
		"CREATE TABLE `yara_file_paths` ( " +
			"  `file_integrity_monitoring_id` int(11) NOT NULL DEFAULT '0', " +
			"  `yara_signature_id` int(11) NOT NULL DEFAULT '0', " +
			"  PRIMARY KEY (`file_integrity_monitoring_id`,`yara_signature_id`), " +
			"  KEY `fk_yara_signature_id` (`yara_signature_id`), " +
			"  CONSTRAINT `fk_file_integrity_monitoring_id` FOREIGN KEY (`file_integrity_monitoring_id`) REFERENCES `file_integrity_monitorings` (`id`), " +
			"  CONSTRAINT `fk_yara_signature_id` FOREIGN KEY (`yara_signature_id`) REFERENCES `yara_signatures` (`id`) " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8; "
	if _, err := tx.Exec(sqlStatement); err != nil {
		return err
	}
	sqlStatement =
		"CREATE TABLE `yara_signature_paths` ( " +
			"  `id` int(11) NOT NULL AUTO_INCREMENT, " +
			"  `file_path` varchar(255) NOT NULL DEFAULT '', " +
			"  `yara_signature_id` int(11) NOT NULL DEFAULT '0', " +
			"  PRIMARY KEY (`id`), " +
			"  KEY `fk_yara_signature` (`yara_signature_id`), " +
			"  CONSTRAINT `fk_yara_signature` FOREIGN KEY (`yara_signature_id`) REFERENCES `yara_signatures` (`id`) ON DELETE CASCADE " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8; "
	if _, err := tx.Exec(sqlStatement); err != nil {
		return err
	}

	return nil
}

func Down_20170109130438(tx *sql.Tx) error {
	sqlStatement := "DROP TABLE IF EXISTS `yara_signature_paths`;"
	_, err := tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	sqlStatement = "DROP TABLE IF EXISTS `yara_file_paths`;"
	_, err = tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	sqlStatement = "DROP TABLE IF EXISTS `yara_signatures`;"
	_, err = tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	return nil
}
