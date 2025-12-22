/**
 * BigQuery client module for PR pickup time metrics collector
 * Handles authentication and data upload to BigQuery
 */

import { BigQuery } from '@google-cloud/bigquery';
import fs from 'fs';
import logger from './logger.js';

/**
 * BigQuery client class
 */
export class BigQueryClient {
  /**
   * Creates a new BigQuery client
   * @param {string} keyFilePath - Path to the service account key file
   */
  constructor(keyFilePath) {
    this.bigquery = null;
    this.projectId = null;
    this.initialize(keyFilePath);
  }

  /**
   * Initializes the BigQuery client
   * @param {string} keyFilePath - Path to the service account key file
   */
  initialize(keyFilePath) {
    // Check if the key file exists
    if (!fs.existsSync(keyFilePath)) {
      const err = new Error(
        `Service account key file not found at ${keyFilePath}`
      );
      logger.error('Failed to initialize BigQuery client', {}, err);
      throw err;
    }

    try {
      // Read and parse the service account key to extract project_id
      const keyFileContent = fs.readFileSync(keyFilePath, 'utf8');
      const serviceAccountKey = JSON.parse(keyFileContent);

      if (!serviceAccountKey.project_id) {
        throw new Error(
          'Service account key file must contain a project_id field'
        );
      }

      this.projectId = serviceAccountKey.project_id;
      logger.info(
        `Extracted project ID from service account key: ${this.projectId}`
      );

      this.bigquery = new BigQuery({
        keyFilename: keyFilePath,
        projectId: this.projectId,
      });

      logger.info('BigQuery client initialized');
    } catch (err) {
      logger.error('Failed to initialize BigQuery client', {}, err);
      throw err;
    }
  }

  /**
   * Gets the project ID extracted from the service account key
   * @returns {string} Project ID
   */
  getProjectId() {
    return this.projectId;
  }

  /**
   * Creates a table if it doesn't exist with table-specific configuration
   * @param {string} datasetId - BigQuery dataset ID
   * @param {string} tableId - BigQuery table ID
   * @param {Object} schema - BigQuery table schema
   * @param {string} metricType - Type of metric (e.g., 'time_to_first_review', 'time_to_merge')
   */
  async createTableIfNotExists(datasetId, tableId, schema, metricType) {
    try {
      // Get a reference to the dataset
      const dataset = this.bigquery.dataset(datasetId);

      // Check if the dataset exists, create it if it doesn't
      const [datasetExists] = await dataset.exists();

      if (!datasetExists) {
        logger.info(`Dataset ${datasetId} does not exist, creating it`);
        await dataset.create();
        logger.info(`Dataset ${datasetId} created`);
      }

      // Get a reference to the table
      const table = dataset.table(tableId);

      // Check if the table exists, create it if it doesn't
      const [tableExists] = await table.exists();

      if (!tableExists) {
        logger.info(`Table ${tableId} does not exist, creating it`);

        // Get table-specific configuration based on metric type
        const tableConfig = this.getConfigurationForMetricType(metricType);

        const options = {
          schema,
          timePartitioning: tableConfig.timePartitioning,
          clustering: tableConfig.clustering,
        };

        await table.create(options);
        logger.info(`Table ${tableId} created`);
      }
    } catch (err) {
      logger.error(`Error creating table ${datasetId}.${tableId}`, {}, err);
      throw err;
    }
  }

  /**
   * Gets table-specific configuration for partitioning and clustering based on metric type
   * @param {string} metricType - Type of metric (e.g., 'time_to_first_review', 'time_to_merge')
   * @returns {Object} Table configuration
   */
  getConfigurationForMetricType(metricType) {
    switch (metricType) {
    case 'time_to_first_review':
      return {
        timePartitioning: {
          type: 'DAY',
          field: 'first_review_time',
        },
        clustering: {
          fields: ['pr_creator', 'pr_number'],
        },
      };

    case 'time_to_merge':
      return {
        timePartitioning: {
          type: 'DAY',
          field: 'merge_time',
        },
        clustering: {
          fields: ['pr_creator', 'pr_number'],
        },
      };

    default:
      throw new Error(
        `Unknown metric type for table configuration: ${metricType}`
      );
    }
  }

  /**
   * Gets the BigQuery table schema for a specific metric type
   * @param {string} metricType - Type of metric (e.g., 'time_to_first_review', 'time_to_merge')
   * @returns {Object} BigQuery table schema
   */
  getSchemaForMetricType(metricType) {
    switch (metricType) {
    case 'time_to_first_review':
      return {
        fields: [
          { name: 'review_date', type: 'DATE', mode: 'REQUIRED' },
          { name: 'pr_creator', type: 'STRING', mode: 'REQUIRED' },
          { name: 'pr_url', type: 'STRING', mode: 'REQUIRED' },
          { name: 'pickup_time_seconds', type: 'INTEGER', mode: 'REQUIRED' },
          { name: 'repository', type: 'STRING', mode: 'REQUIRED' },
          { name: 'pr_number', type: 'INTEGER', mode: 'REQUIRED' },
          { name: 'target_branch', type: 'STRING', mode: 'REQUIRED' },
          { name: 'ready_time', type: 'TIMESTAMP', mode: 'REQUIRED' },
          { name: 'first_review_time', type: 'TIMESTAMP', mode: 'REQUIRED' },
        ],
      };

    case 'time_to_merge':
      return {
        fields: [
          { name: 'merge_date', type: 'DATE', mode: 'REQUIRED' },
          { name: 'pr_creator', type: 'STRING', mode: 'REQUIRED' },
          { name: 'pr_url', type: 'STRING', mode: 'REQUIRED' },
          { name: 'merge_time_seconds', type: 'INTEGER', mode: 'REQUIRED' },
          { name: 'repository', type: 'STRING', mode: 'REQUIRED' },
          { name: 'pr_number', type: 'INTEGER', mode: 'REQUIRED' },
          { name: 'target_branch', type: 'STRING', mode: 'REQUIRED' },
          { name: 'ready_time', type: 'TIMESTAMP', mode: 'REQUIRED' },
          { name: 'merge_time', type: 'TIMESTAMP', mode: 'REQUIRED' },
        ],
      };

    default:
      throw new Error(`Unknown metric type: ${metricType}`);
    }
  }

  /**
   * Transforms metrics to BigQuery row format based on metric type
   * @param {Object} metrics - Metrics object
   * @returns {Object} BigQuery row
   */
  transformMetricsToRow(metrics) {
    switch (metrics.metricType) {
    case 'time_to_first_review':
      return {
        review_date: metrics.reviewDate,
        pr_creator: metrics.prCreator,
        pr_url: metrics.prUrl,
        pickup_time_seconds: metrics.pickupTimeSeconds,
        repository: metrics.repository,
        pr_number: metrics.prNumber,
        target_branch: metrics.targetBranch,
        ready_time: metrics.readyTime.toISOString(),
        first_review_time: metrics.firstReviewTime.toISOString(),
      };

    case 'time_to_merge':
      return {
        merge_date: metrics.mergeDate,
        pr_creator: metrics.prCreator,
        pr_url: metrics.prUrl,
        merge_time_seconds: metrics.mergeTimeSeconds,
        repository: metrics.repository,
        pr_number: metrics.prNumber,
        target_branch: metrics.targetBranch,
        ready_time: metrics.readyTime.toISOString(),
        merge_time: metrics.mergeTime.toISOString(),
      };

    default:
      throw new Error(`Unknown metric type: ${metrics.metricType}`);
    }
  }

  /**
   * Checks if metrics already exist in BigQuery for the given PR numbers
   * @param {string} datasetId - BigQuery dataset ID
   * @param {string} tableId - BigQuery table ID
   * @param {Array} prNumbers - Array of PR numbers to check
   * @returns {Object} Object with PR numbers as keys and boolean values indicating if they exist
   */
  async checkExistingMetrics(datasetId, tableId, prNumbers) {
    try {
      // Get a reference to the table
      const table = this.bigquery.dataset(datasetId).table(tableId);

      // Check if the table exists
      const [tableExists] = await table.exists();
      if (!tableExists) {
        // If the table doesn't exist, no metrics exist
        return prNumbers.reduce((acc, prNumber) => {
          acc[prNumber] = false;
          return acc;
        }, {});
      }

      // Create a parameterized query to check for existing PR numbers
      const query = `
        SELECT pr_number
        FROM \`${this.projectId}.${datasetId}.${tableId}\`
        WHERE pr_number IN UNNEST(@prNumbers)
      `;

      const options = {
        query,
        params: {
          prNumbers,
        },
      };

      // Run the query
      const [rows] = await this.bigquery.query(options);

      // Create a map of existing PR numbers
      const existingPRs = rows.reduce((acc, row) => {
        acc[row.pr_number] = true;
        return acc;
      }, {});

      // Return a map of all PR numbers with their existence status
      return prNumbers.reduce((acc, prNumber) => {
        acc[prNumber] = !!existingPRs[prNumber];
        return acc;
      }, {});
    } catch (err) {
      logger.error(
        `Error checking existing metrics in BigQuery ${datasetId}.${tableId}`,
        {},
        err
      );
      // If there's an error, assume no metrics exist
      return prNumbers.reduce((acc, prNumber) => {
        acc[prNumber] = false;
        return acc;
      }, {});
    }
  }

  /**
   * Uploads metrics to BigQuery
   * @param {string} datasetId - BigQuery dataset ID
   * @param {string} tableId - BigQuery table ID
   * @param {Array} metrics - Array of metrics
   */
  async uploadMetrics(datasetId, tableId, metrics) {
    try {
      if (!metrics || metrics.length === 0) {
        logger.warn('No metrics to upload');
        return;
      }

      logger.info(
        `Uploading ${metrics.length} metrics to BigQuery table ${tableId}`
      );

      // Get metric type from the first metric to determine schema
      const metricType = metrics[0]?.metricType;
      if (!metricType) {
        throw new Error('Metrics must have a metricType field');
      }

      // Ensure the table exists with the correct schema
      const schema = this.getSchemaForMetricType(metricType);
      await this.createTableIfNotExists(datasetId, tableId, schema, metricType);

      // Get all PR numbers from the metrics
      const prNumbers = metrics.map((metric) => metric.prNumber);

      // Check which PR numbers already exist in BigQuery
      const existingMetrics = await this.checkExistingMetrics(
        datasetId,
        tableId,
        prNumbers
      );

      // Filter out metrics that already exist
      const newMetrics = metrics.filter(
        (metric) => !existingMetrics[metric.prNumber]
      );

      if (newMetrics.length === 0) {
        logger.info('All metrics already exist in BigQuery, nothing to upload');
        return;
      }

      logger.info(
        `Uploading ${newMetrics.length} new metrics to BigQuery (${
          metrics.length - newMetrics.length
        } already exist)`
      );

      // Transform metrics to BigQuery row format
      const rows = newMetrics.map((metric) =>
        this.transformMetricsToRow(metric)
      );

      // Get a reference to the table
      const table = this.bigquery.dataset(datasetId).table(tableId);

      // Upload the rows to BigQuery
      const [apiResponse] = await table.insert(rows);

      logger.info(
        `Successfully uploaded ${newMetrics.length} metrics to BigQuery`,
        {
          datasetId,
          tableId,
          insertedRows: newMetrics.length,
          skippedRows: metrics.length - newMetrics.length,
        }
      );

      return apiResponse;
    } catch (err) {
      logger.error(
        `Error uploading metrics to BigQuery ${datasetId}.${tableId}`,
        {},
        err
      );

      // Log more details about the error if it's an insertion error
      if (
        err.name === 'PartialFailureError' &&
        err.errors &&
        err.errors.length > 0
      ) {
        err.errors.forEach((error, index) => {
          logger.error(`Row ${index} error:`, { error });
        });
      }

      throw err;
    }
  }
}

export default BigQueryClient;
