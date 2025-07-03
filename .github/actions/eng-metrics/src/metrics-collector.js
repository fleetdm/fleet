/**
 * Engineering metrics collector module
 * Orchestrates the collection and uploading of comprehensive GitHub engineering metrics:
 * - Time to First Review (currently implemented)
 * - Time to Merge (planned)
 * - Time to QA Ready (planned)
 * - Time to Production Ready (planned)
 */

import GitHubClient from './github-client.js';
import BigQueryClient from './bigquery-client.js';
import { UserGroupClient } from './user-group-client.js';
import { parseProductGroups } from './markdown-parser.js';
import { filterValidUserGroups } from './github-validator.js';
import logger from './logger.js';

/**
 * Metrics collector class
 */
export class MetricsCollector {
  /**
   * Creates a new metrics collector
   * @param {Object} config - Configuration object
   */
  constructor(config) {
    this.config = config;
    this.githubClient = null;
    this.bigqueryClient = null;
    this.userGroupClient = null;
  }

  /**
   * Initializes the metrics collector
   */
  async initialize() {
    try {
      logger.info('Initializing metrics collector');

      // Initialize GitHub client
      this.githubClient = new GitHubClient(this.config.githubToken);

      // Initialize BigQuery client only if not in print-only mode
      if (!this.config.printOnly) {
        this.bigqueryClient = new BigQueryClient(this.config.serviceAccountKeyPath);
      } else {
        logger.info('Running in print-only mode, BigQuery client not initialized');
      }

      // Initialize User Group client if user group processing is enabled
      if (this.config.userGroupEnabled) {
        if (!this.config.printOnly) {
          // Get project ID from BigQuery client
          const projectId = this.bigqueryClient.getProjectId();
          this.userGroupClient = new UserGroupClient(
            projectId,
            this.config.bigQueryDatasetId,
            this.config.serviceAccountKeyPath,
            this.config.printOnly
          );
        } else {
          // For print-only mode, we don't need a real project ID
          this.userGroupClient = new UserGroupClient(
            'print-only-project',
            this.config.bigQueryDatasetId,
            this.config.serviceAccountKeyPath,
            this.config.printOnly
          );
        }
        logger.info('User group client initialized');
      }

      logger.info('Metrics collector initialized');
    } catch (err) {
      logger.error('Failed to initialize metrics collector', {}, err);
      throw err;
    }
  }

  /**
   * Collects metrics for a single repository
   * @param {string} repository - Repository in the format owner/repo
   * @returns {Array} Array of engineering metrics
   */
  async collectRepositoryMetrics(repository) {
    const [owner, repo] = repository.split('/');
    if (!owner || !repo) {
      const err = new Error(`Invalid repository format: ${repository}`);
      logger.error(`Error collecting metrics for ${repository}`, {}, err);
      return [];
    }
    logger.info(`Collecting metrics for ${repository}`);

    try {
      // Calculate the date to fetch PRs from (lookbackDays ago)
      const since = new Date();
      since.setDate(since.getDate() - this.config.lookbackDays);

      // Fetch PRs updated since the lookback date
      const pullRequests = await this.githubClient.fetchPullRequests(
        owner,
        repo,
        'all',
        since,
        this.config.targetBranch
      );

      logger.info(`Found ${pullRequests.length} PRs for ${repository}`);

      // Collect metrics for each PR
      const metrics = [];

      for (const pr of pullRequests) {
        try {
          // Fetch PR timeline events (shared for all metrics)
          const timelineEvents = await this.githubClient.fetchPRTimelineEvents(
            owner,
            repo,
            pr.number
          );

          // Fetch PR review events (needed for Time to First Review)
          const rawReviewEvents = await this.githubClient.fetchPRReviewEvents(
            owner,
            repo,
            pr.number
          );

          // Filter bot reviews if configured
          const reviewEvents = this.githubClient.filterBotReviews(
            rawReviewEvents,
            this.config.excludeBotReviews
          );

          // Collect enabled metrics for this PR
          const prMetrics = await this.collectPRMetrics(pr, timelineEvents, reviewEvents);
          metrics.push(...prMetrics);
        } catch (err) {
          logger.error(`Error collecting metrics for PR ${repository}#${pr.number}`, {}, err);
        }
      }

      logger.info(`Collected ${metrics.length} metrics for ${repository}`);
      return metrics;
    } catch (err) {
      logger.error(`Error collecting metrics for ${repository}`, {}, err);
      return [];
    }
  }

  /**
   * Collects enabled metrics for a single PR
   * @param {Object} pr - Pull request object
   * @param {Array} timelineEvents - PR timeline events
   * @param {Array} reviewEvents - PR review events
   * @returns {Array} Array of metrics for this PR
   */
  async collectPRMetrics(pr, timelineEvents, reviewEvents) {
    const metrics = [];

    // Collect Time to First Review if enabled
    if (this.config.metrics.timeToFirstReview.enabled) {
      try {
        const pickupTimeMetrics = this.githubClient.calculatePickupTime(
          pr,
          timelineEvents,
          reviewEvents
        );

        if (pickupTimeMetrics) {
          metrics.push(pickupTimeMetrics);
        }
      } catch (err) {
        logger.error(`Error calculating Time to First Review for PR #${pr.number}`, {}, err);
      }
    }

    // Collect Time to Merge if enabled
    if (this.config.metrics.timeToMerge.enabled) {
      try {
        const mergeTimeMetrics = this.githubClient.calculateTimeToMerge(
          pr,
          timelineEvents,
          reviewEvents
        );

        if (mergeTimeMetrics) {
          metrics.push(mergeTimeMetrics);
        }
      } catch (err) {
        logger.error(`Error calculating Time to Merge for PR #${pr.number}`, {}, err);
      }
    }

    return metrics;
  }

  /**
   * Collects metrics for all repositories
   * @returns {Array} Array of engineering metrics
   */
  async collectMetrics() {
    try {
      logger.info('Collecting metrics for all repositories');

      const allMetrics = [];

      // Collect metrics for each repository
      for (const repository of this.config.repositories) {
        const metrics = await this.collectRepositoryMetrics(repository);
        allMetrics.push(...metrics);
      }

      logger.info(`Collected ${allMetrics.length} metrics in total`);
      return allMetrics;
    } catch (err) {
      logger.error('Error collecting metrics', {}, err);
      throw err;
    }
  }

  /**
   * Processes user groups from the markdown file
   * @returns {Promise<void>}
   */
  async processUserGroups() {
    if (!this.config.userGroupEnabled) {
      logger.info('User group processing is disabled');
      return;
    }

    try {
      logger.info('Processing user groups');

      // Parse user groups from markdown file
      const userGroups = parseProductGroups(this.config.userGroupFilepath);

      if (userGroups.length === 0) {
        logger.warn('No user groups found in markdown file');
        return;
      }

      // Validate GitHub usernames
      const validUserGroups = await filterValidUserGroups(
        this.config.githubToken,
        userGroups
      );

      if (validUserGroups.length === 0) {
        logger.warn('No valid user groups found after validation');
        return;
      }

      // Sync user groups to BigQuery
      await this.userGroupClient.syncUserGroups(validUserGroups);

      logger.info(`Successfully processed ${validUserGroups.length} user group mappings`);
    } catch (err) {
      logger.error('Error processing user groups', {}, err);
      throw err;
    }
  }

  /**
   * Prints metrics to the console in a readable format
   * @param {Array} metrics - Array of engineering metrics
   */
  printMetrics(metrics) {
    try {
      if (!metrics || metrics.length === 0) {
        logger.warn('No metrics to print');
        return;
      }

      logger.info(`Printing ${metrics.length} metrics to console`);

      // Group metrics by type for organized display
      const metricsByType = this.groupMetricsByType(metrics);

      console.log('\n=== Engineering Metrics ===\n');

      // Print each metric type separately
      for (const [metricType, typeMetrics] of Object.entries(metricsByType)) {
        if (typeMetrics.length === 0) continue;

        console.log(`--- ${this.getMetricTypeDisplayName(metricType)} (${typeMetrics.length} metrics) ---\n`);

        // Sort metrics by time (descending)
        const sortedMetrics = [...typeMetrics].sort((a, b) => {
          const timeFieldA = this.getTimeFieldForMetricType(metricType, a);
          const timeFieldB = this.getTimeFieldForMetricType(metricType, b);
          return timeFieldB - timeFieldA;
        });

        // Print each metric
        sortedMetrics.forEach((metric, index) => {
          this.printSingleMetric(metric, index + 1);
        });

        // Print summary statistics for this metric type
        this.printMetricTypeSummary(metricType, typeMetrics);
        console.log('');
      }

      logger.info('Metrics printed successfully');
    } catch (err) {
      logger.error('Error printing metrics', {}, err);
      throw err;
    }
  }

  /**
   * Prints a single metric to the console
   * @param {Object} metric - Single metric object
   * @param {number} index - Index for display
   */
  printSingleMetric(metric, index) {
    console.log(`[${index}] PR: ${metric.repository}#${metric.prNumber}`);
    console.log(`    URL: ${metric.prUrl}`);
    console.log(`    Creator: ${metric.prCreator}`);
    console.log(`    Ready Time: ${metric.readyTime.toISOString()}${metric.readyEventType ? ` (${metric.readyEventType})` : ''}`);

    if (metric.metricType === 'time_to_first_review') {
      const hours = Math.floor(metric.pickupTimeSeconds / 3600);
      const minutes = Math.floor((metric.pickupTimeSeconds % 3600) / 60);
      const seconds = metric.pickupTimeSeconds % 60;

      console.log(`    First Review Time: ${metric.firstReviewTime.toISOString()}`);
      console.log(`    Pickup Time: ${hours}h ${minutes}m ${seconds}s (${metric.pickupTimeSeconds} seconds)`);
    } else if (metric.metricType === 'time_to_merge') {
      const hours = Math.floor(metric.mergeTimeSeconds / 3600);
      const minutes = Math.floor((metric.mergeTimeSeconds % 3600) / 60);
      const seconds = metric.mergeTimeSeconds % 60;

      console.log(`    Merge Time: ${metric.mergeTime.toISOString()}`);
      console.log(`    Time to Merge: ${hours}h ${minutes}m ${seconds}s (${metric.mergeTimeSeconds} seconds)`);
    }

    console.log('');
  }

  /**
   * Prints summary statistics for a metric type
   * @param {string} metricType - Type of metric
   * @param {Array} metrics - Array of metrics of this type
   */
  printMetricTypeSummary(metricType, metrics) {
    const timeField = metricType === 'time_to_first_review' ? 'pickupTimeSeconds' : 'mergeTimeSeconds';
    const totalTime = metrics.reduce((sum, metric) => sum + metric[timeField], 0);
    const avgTime = totalTime / metrics.length;
    const avgHours = Math.floor(avgTime / 3600);
    const avgMinutes = Math.floor((avgTime % 3600) / 60);
    const avgSeconds = Math.floor(avgTime % 60);

    console.log(`=== ${this.getMetricTypeDisplayName(metricType)} Summary ===`);
    console.log(`Total PRs: ${metrics.length}`);
    console.log(`Average Time: ${avgHours}h ${avgMinutes}m ${avgSeconds}s (${Math.floor(avgTime)} seconds)`);
  }

  /**
   * Gets display name for metric type
   * @param {string} metricType - Type of metric
   * @returns {string} Display name
   */
  getMetricTypeDisplayName(metricType) {
    switch (metricType) {
    case 'time_to_first_review':
      return 'Time to First Review';
    case 'time_to_merge':
      return 'Time to Merge';
    default:
      return metricType;
    }
  }

  /**
   * Gets the time field value for sorting metrics
   * @param {string} metricType - Type of metric
   * @param {Object} metric - Metric object
   * @returns {number} Time value in seconds
   */
  getTimeFieldForMetricType(metricType, metric) {
    switch (metricType) {
    case 'time_to_first_review':
      return metric.pickupTimeSeconds;
    case 'time_to_merge':
      return metric.mergeTimeSeconds;
    default:
      return 0;
    }
  }

  /**
   * Uploads metrics to BigQuery, grouped by metric type
   * @param {Array} metrics - Array of engineering metrics
   */
  async uploadMetrics(metrics) {
    try {
      if (!metrics || metrics.length === 0) {
        logger.warn('No metrics to upload');
        return;
      }

      logger.info(`Uploading ${metrics.length} metrics to BigQuery`);

      // Group metrics by type
      const metricsByType = this.groupMetricsByType(metrics);

      // Upload each metric type to its respective table
      for (const [metricType, typeMetrics] of Object.entries(metricsByType)) {
        if (typeMetrics.length === 0) continue;

        const tableName = this.getTableNameForMetricType(metricType);
        logger.info(`Uploading ${typeMetrics.length} ${metricType} metrics to table ${tableName}`);

        await this.bigqueryClient.uploadMetrics(
          this.config.bigQueryDatasetId,
          tableName,
          typeMetrics
        );
      }

      logger.info('All metrics uploaded successfully');
    } catch (err) {
      logger.error('Error uploading metrics to BigQuery', {}, err);
      throw err;
    }
  }

  /**
   * Groups metrics by their type
   * @param {Array} metrics - Array of metrics
   * @returns {Object} Object with metric types as keys and arrays of metrics as values
   */
  groupMetricsByType(metrics) {
    return metrics.reduce((groups, metric) => {
      const type = metric.metricType;
      if (!groups[type]) {
        groups[type] = [];
      }
      groups[type].push(metric);
      return groups;
    }, {});
  }

  /**
   * Gets the table name for a specific metric type
   * @param {string} metricType - Type of metric
   * @returns {string} Table name
   */
  getTableNameForMetricType(metricType) {
    switch (metricType) {
    case 'time_to_first_review':
      return this.config.metrics.timeToFirstReview.tableName;
    case 'time_to_merge':
      return this.config.metrics.timeToMerge.tableName;
    default:
      throw new Error(`Unknown metric type: ${metricType}`);
    }
  }

  /**
   * Runs the metrics collection and upload process
   */
  async run() {
    try {
      logger.info('Starting engineering metrics collection');

      // Initialize the metrics collector
      await this.initialize();

      // Process user groups if enabled
      await this.processUserGroups();

      // Collect metrics
      const metrics = await this.collectMetrics();

      if (this.config.printOnly) {
        // Print metrics to console
        this.printMetrics(metrics);
      } else {
        // Upload metrics to BigQuery
        await this.uploadMetrics(metrics);
      }

      logger.info('Engineering metrics collection completed successfully');
      return metrics;
    } catch (err) {
      logger.error('Error running engineering metrics collection', {}, err);
      throw err;
    }
  }
}

export default MetricsCollector;
