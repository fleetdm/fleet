package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170109094020, Down_20170109094020)
}

func Up_20170109094020(tx *sql.Tx) error {
	sqlStatement :=
		"CREATE TABLE `file_integrity_monitorings` ( " +
			"  `id` int(10) NOT NULL AUTO_INCREMENT, " +
			"  `section_name` varchar(255) NOT NULL DEFAULT '', " +
			"  `description` varchar(255) NOT NULL DEFAULT ''," +
			"  PRIMARY KEY (`id`)," +
			"  UNIQUE KEY `idx_unique_section_name` (`section_name`) USING BTREE" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	sqlStatement =
		"CREATE TABLE `file_integrity_monitoring_files` (" +
			"  `id` int(10) NOT NULL AUTO_INCREMENT," +
			"  `file` varchar(255) NOT NULL DEFAULT ''," +
			"  `file_integrity_monitoring_id` int(10) NOT NULL DEFAULT '0'," +
			"  PRIMARY KEY (`id`)," +
			"  UNIQUE KEY `idx_fim_unique_file_name` (`file`) USING BTREE," +
			"  KEY `fk_file_integrity_monitoring` (`file_integrity_monitoring_id`)," +
			"  CONSTRAINT `fk_file_integrity_monitoring` FOREIGN KEY (`file_integrity_monitoring_id`) REFERENCES `file_integrity_monitorings` (`id`) ON DELETE CASCADE" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err = tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	return nil
}

func Down_20170109094020(tx *sql.Tx) error {
	sqlStatement := "DROP TABLE IF EXISTS `file_integrity_monitoring_files`; "
	_, err := tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	sqlStatement = "DROP TABLE IF EXISTS `file_integrity_monitorings`;"
	_, err = tx.Exec(sqlStatement)
	if err != nil {
		return err
	}
	return nil
}
