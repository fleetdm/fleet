#
# SQL Export
# Created by Querious (1055)
# Created: November 7, 2016 at 11:39:23 PM GMT+8
# Encoding: Unicode (UTF-8)
#

use `kolide`;

SET @PREVIOUS_FOREIGN_KEY_CHECKS = @@FOREIGN_KEY_CHECKS;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `org_infos`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `sessions`;
DROP TABLE IF EXISTS `queries`;
DROP TABLE IF EXISTS `password_reset_requests`;
DROP TABLE IF EXISTS `packs`;
DROP TABLE IF EXISTS `pack_targets`;
DROP TABLE IF EXISTS `pack_queries`;
DROP TABLE IF EXISTS `options`;
DROP TABLE IF EXISTS `labels`;
DROP TABLE IF EXISTS `label_query_executions`;
DROP TABLE IF EXISTS `invites`;
DROP TABLE IF EXISTS `hosts`;
DROP TABLE IF EXISTS `distributed_query_executions`;
DROP TABLE IF EXISTS `distributed_query_campaigns`;
DROP TABLE IF EXISTS `distributed_query_campaign_targets`;
DROP TABLE IF EXISTS `app_configs`;


SET FOREIGN_KEY_CHECKS = @PREVIOUS_FOREIGN_KEY_CHECKS;
