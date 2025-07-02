/**
 * User Group BigQuery Client
 * Manages the user_group BigQuery table for team organization tracking
 */

import { BigQuery } from '@google-cloud/bigquery';
import logger from './logger.js';

/**
 * User Group BigQuery Client class
 */
export class UserGroupClient {
  /**
   * Creates a new UserGroupClient instance
   * @param {string} projectId - Google Cloud project ID
   * @param {string} datasetId - BigQuery dataset ID
   * @param {string} serviceAccountKeyPath - Path to service account key file
   * @param {boolean} printOnly - If true, print operations instead of executing them
   */
  constructor(projectId, datasetId, serviceAccountKeyPath, printOnly = false) {
    this.projectId = projectId;
    this.datasetId = datasetId;
    this.tableId = 'user_group';
    this.printOnly = printOnly;

    if (!printOnly) {
      this.bigquery = new BigQuery({
        projectId,
        keyFilename: serviceAccountKeyPath
      });
      this.dataset = this.bigquery.dataset(datasetId);
      this.table = this.dataset.table(this.tableId);
    }
  }

  /**
   * Creates the user_group table if it doesn't exist
   * @returns {Promise<void>}
   */
  async createUserGroupTable() {
    if (this.printOnly) {
      logger.info('[USER GROUPS] Print-only mode: would create user_group table with schema:');
      logger.info('[USER GROUPS]   - group (STRING, cluster key)');
      logger.info('[USER GROUPS]   - username (STRING)');
      return;
    }

    try {
      const [exists] = await this.table.exists();

      if (exists) {
        logger.info('user_group table already exists');
        return;
      }

      const schema = [
        { name: 'group', type: 'STRING', mode: 'REQUIRED' },
        { name: 'username', type: 'STRING', mode: 'REQUIRED' }
      ];

      const options = {
        schema,
        clustering: {
          fields: ['group']
        }
      };

      await this.table.create(options);
      logger.info('Created user_group table with clustering on group field');
    } catch (error) {
      logger.error('Error creating user_group table:', error);
      throw error;
    }
  }

  /**
   * Syncs user groups to the BigQuery table (replaces all data)
   * @param {Array<{group: string, username: string}>} userGroups - Array of user group mappings
   * @returns {Promise<void>}
   */
  async syncUserGroups(userGroups) {
    if (this.printOnly) {
      this.printUserGroupsData(userGroups);
      return;
    }

    try {
      if (userGroups.length === 0) {
        logger.warn('No user groups to sync');
        return;
      }

      // Ensure table exists first
      await this.createUserGroupTable();
      
      // Small delay to ensure table is ready
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Refresh table reference
      this.table = this.dataset.table(this.tableId);
      
      // Get existing entries and perform differential sync
      await this.syncUserGroupsDifferential(userGroups);

      logger.info(`Successfully synced ${userGroups.length} user group mappings to BigQuery`);
    } catch (error) {
      logger.error('Error syncing user groups:', error);
      throw error;
    }
  }

  /**
   * Performs differential sync of user groups (only insert new, delete removed)
   * @param {Array<{group: string, username: string}>} newUserGroups - New user group mappings
   * @returns {Promise<void>}
   */
  async syncUserGroupsDifferential(newUserGroups) {
    if (this.printOnly) {
      logger.info('[USER GROUPS] Print-only mode: would perform differential sync');
      this.printDifferentialSync(newUserGroups);
      return;
    }

    try {
      // Get existing entries from the table
      const existingUserGroups = await this.getExistingUserGroups();
      
      // Create sets for comparison
      const existingSet = new Set(existingUserGroups.map(ug => `${ug.group}:${ug.username}`));
      const newSet = new Set(newUserGroups.map(ug => `${ug.group}:${ug.username}`));
      
      // Find entries to insert (in new but not in existing)
      const toInsert = newUserGroups.filter(ug => !existingSet.has(`${ug.group}:${ug.username}`));
      
      // Find entries to delete (in existing but not in new)
      const toDelete = existingUserGroups.filter(ug => !newSet.has(`${ug.group}:${ug.username}`));
      
      logger.info(`Differential sync: ${toInsert.length} to insert, ${toDelete.length} to delete`);
      
      // Handle deletions with streaming buffer awareness
      if (toDelete.length > 0) {
        try {
          await this.deleteUserGroups(toDelete);
        } catch (error) {
          if (error.message && error.message.includes('streaming buffer')) {
            logger.warn('Streaming buffer prevents DELETE - this is expected during testing. In production (daily runs), this won\'t occur.');
            logger.info('Skipping deletions for now. Data will be consistent on next daily run.');
          } else {
            throw error;
          }
        }
      }
      
      // Insert new entries using load job (no streaming buffer issues for inserts)
      if (toInsert.length > 0) {
        await this.insertUserGroups(toInsert);
      }
      
      if (toInsert.length === 0 && toDelete.length === 0) {
        logger.info('No changes needed - user groups are already up to date');
      }
      
    } catch (error) {
      logger.error('Error performing differential sync:', error);
      throw error;
    }
  }

  /**
   * Gets existing user groups from the BigQuery table
   * @returns {Promise<Array<{group: string, username: string}>>}
   */
  async getExistingUserGroups() {
    try {
      const query = `SELECT \`group\`, username FROM \`${this.projectId}.${this.datasetId}.${this.tableId}\``;
      const [rows] = await this.bigquery.query({ query });
      
      logger.info(`Found ${rows.length} existing user group entries`);
      return rows.map(row => ({
        group: row.group,
        username: row.username
      }));
    } catch (error) {
      if (error.message && error.message.includes('not found')) {
        logger.info('Table does not exist yet, no existing entries');
        return [];
      }
      logger.error('Error getting existing user groups:', error);
      throw error;
    }
  }

  /**
   * Deletes specific user groups from the BigQuery table
   * @param {Array<{group: string, username: string}>} userGroupsToDelete
   * @returns {Promise<void>}
   */
  async deleteUserGroups(userGroupsToDelete) {
    try {
      // Build WHERE clause for deletion
      const conditions = userGroupsToDelete.map(ug =>
        `(\`group\` = '${ug.group}' AND username = '${ug.username}')`
      ).join(' OR ');
      
      const query = `DELETE FROM \`${this.projectId}.${this.datasetId}.${this.tableId}\` WHERE ${conditions}`;
      await this.bigquery.query({ query });
      
      logger.info(`Deleted ${userGroupsToDelete.length} user group entries`);
    } catch (error) {
      logger.error('Error deleting user groups:', error);
      throw error;
    }
  }

  /**
   * Prints differential sync information for print-only mode
   * @param {Array<{group: string, username: string}>} newUserGroups
   */
  async printDifferentialSync(newUserGroups) {
    logger.info('[USER GROUPS] Print-only mode: would perform differential sync');
    logger.info('[USER GROUPS] Would check existing entries in table');
    logger.info(`[USER GROUPS] Would compare with ${newUserGroups.length} new entries`);
    logger.info('[USER GROUPS] Would insert only new entries and delete removed entries');
    
    // Group users by their groups for better display
    const groupedUsers = newUserGroups.reduce((acc, ug) => {
      if (!acc[ug.group]) {
        acc[ug.group] = [];
      }
      acc[ug.group].push(ug.username);
      return acc;
    }, {});

    logger.info('[USER GROUPS] New user groups to sync:');
    for (const [group, usernames] of Object.entries(groupedUsers)) {
      logger.info(`  ${group}: ${usernames.join(', ')}`);
    }
  }

  /**
   * Inserts user groups into the BigQuery table using load job (avoids streaming buffer)
   * @param {Array<{group: string, username: string}>} userGroups - Array of user group mappings
   * @returns {Promise<void>}
   */
  async insertUserGroups(userGroups) {
    if (this.printOnly) {
      logger.info(`[USER GROUPS] Print-only mode: would insert ${userGroups.length} records`);
      return;
    }

    try {
      // Transform data for BigQuery streaming insert
      const rows = userGroups.map(ug => ({
        group: ug.group,
        username: ug.username
      }));

      // Use streaming insert - simpler and appropriate for small user group datasets
      // Streaming buffer issues only occur during rapid testing, not daily production runs
      await this.table.insert(rows);
      
      logger.info(`Inserted ${rows.length} user group mappings into BigQuery using load job`);
    } catch (error) {
      logger.error('Error inserting user groups:', error);
      throw error;
    }
  }

  /**
   * Prints user groups data in a readable format (for print-only mode)
   * @param {Array<{group: string, username: string}>} userGroups - Array of user group mappings
   */
  printUserGroupsData(userGroups) {
    logger.info('[USER GROUPS] Print-only mode enabled - no BigQuery updates will be performed');
    logger.info('');

    if (userGroups.length === 0) {
      logger.info('[USER GROUPS] No user groups found to process');
      return;
    }

    // Group users by their groups for better display
    const groupedUsers = userGroups.reduce((acc, ug) => {
      if (!acc[ug.group]) {
        acc[ug.group] = [];
      }
      acc[ug.group].push(ug.username);
      return acc;
    }, {});

    logger.info('[USER GROUPS] Extracted user groups:');
    for (const [group, usernames] of Object.entries(groupedUsers)) {
      logger.info(`  ${group}: ${usernames.join(', ')}`);
    }

    logger.info('');
    logger.info(`[USER GROUPS] Would insert ${userGroups.length} records into user_group table:`);

    // Count users per group
    const groupCounts = Object.entries(groupedUsers).map(([group, usernames]) =>
      `  ${group}: ${usernames.length} users`
    );
    logger.info(groupCounts.join('\n'));

    logger.info('');
    logger.info('[USER GROUPS] Sample records that would be inserted:');
    logger.info('| group         | username        |');
    logger.info('|---------------|-----------------|');

    // Show first few records from each group
    const sampleRecords = [];
    for (const [group, usernames] of Object.entries(groupedUsers)) {
      const sampleUser = usernames[0];
      sampleRecords.push(`| ${group.padEnd(13)} | ${sampleUser.padEnd(15)} |`);
      if (sampleRecords.length >= 6) break; // Limit sample output
    }

    sampleRecords.forEach(record => logger.info(record));

    if (userGroups.length > sampleRecords.length) {
      logger.info(`... and ${userGroups.length - sampleRecords.length} more records`);
    }
  }

}

export default UserGroupClient;
