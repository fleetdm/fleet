/**
 * Configuration module for engineering metrics collector
 * Loads and validates configuration from files and environment variables
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import dotenv from 'dotenv';
import logger from './logger.js';

// Load environment variables from .env file
dotenv.config();

// Get the directory name of the current module
path.dirname(fileURLToPath(import.meta.url));
/**
 * Default configuration values
 */
const DEFAULT_CONFIG = {
  // Default target branch to track PRs for
  targetBranch: 'main',

  // Default BigQuery dataset ID
  bigQueryDatasetId: 'github_metrics',

  // Default time window for fetching PRs (in days)
  lookbackDays: 5,

  // Default print-only mode (false = upload to BigQuery, true = print to console)
  printOnly: false,

  // User group management configuration
  userGroupEnabled: true,
  userGroupFilepath: '../../../handbook/company/product-groups.md',

  // Bot filtering configuration
  excludeBotReviews: true,

  // Multi-table configuration
  metrics: {
    timeToFirstReview: {
      enabled: true,
      tableName: 'pr_first_review'
    },
    timeToMerge: {
      enabled: true,
      tableName: 'pr_merge'
    }
  }
};

/**
 * Loads configuration from a JSON file
 * @param {string} configPath - Path to the configuration file
 * @returns {Object} Configuration object
 */
const loadConfigFromFile = (configPath) => {
  try {
    const resolvedPath = path.resolve(process.cwd(), configPath);
    logger.info(`Loading configuration from ${resolvedPath}`);

    if (!fs.existsSync(resolvedPath)) {
      logger.warn(`Configuration file not found at ${resolvedPath}`);
      return {};
    }

    const configData = fs.readFileSync(resolvedPath, 'utf8');
    return JSON.parse(configData);
  } catch (err) {
    logger.error(`Error loading configuration from file: ${configPath}`, err);
    return {};
  }
};

/**
 * Loads configuration from environment variables
 * @returns {Object} Configuration object
 */
const loadConfigFromEnv = () => {
  // Create a config object with only defined values
  const config = {};

  // Parse repositories from environment variable if provided
  if (process.env.REPOSITORIES) {
    config.repositories = process.env.REPOSITORIES.split(',').map(repo => repo.trim());
  }

  // Add other environment variables if they are defined
  if (process.env.GITHUB_TOKEN) config.githubToken = process.env.GITHUB_TOKEN;
  if (process.env.BIGQUERY_DATASET_ID) config.bigQueryDatasetId = process.env.BIGQUERY_DATASET_ID;
  if (process.env.SERVICE_ACCOUNT_KEY_PATH) config.serviceAccountKeyPath = process.env.SERVICE_ACCOUNT_KEY_PATH;
  if (process.env.TARGET_BRANCH) config.targetBranch = process.env.TARGET_BRANCH;
  if (process.env.PRINT_ONLY) config.printOnly = process.env.PRINT_ONLY === 'true';
  if (process.env.USER_GROUP_ENABLED) config.userGroupEnabled = process.env.USER_GROUP_ENABLED === 'true';
  if (process.env.USER_GROUP_FILEPATH) config.userGroupFilepath = process.env.USER_GROUP_FILEPATH;

  // Handle metrics configuration from environment variables
  if (process.env.ENABLED_METRICS) {
    const enabledMetrics = process.env.ENABLED_METRICS.split(',').map(metric => metric.trim());
    config.metrics = {
      timeToFirstReview: {
        enabled: enabledMetrics.includes('time_to_first_review'),
        tableName: process.env.TIME_TO_FIRST_REVIEW_TABLE || 'pr_first_review'
      },
      timeToMerge: {
        enabled: enabledMetrics.includes('time_to_merge'),
        tableName: process.env.TIME_TO_MERGE_TABLE || 'pr_merge'
      }
    };
  }

  return config;
};

/**
 * Validates the configuration
 * @param {Object} config - Configuration object
 * @returns {boolean} True if configuration is valid, false otherwise
 */
const validateConfig = (config) => {
  // Always required fields
  const requiredFields = [
    'repositories',
    'githubToken'
  ];

  // Fields required only when not in print-only mode
  if (!config.printOnly) {
    requiredFields.push('serviceAccountKeyPath');
  }

  const missingFields = requiredFields.filter(field => !config[field]);

  if (missingFields.length > 0) {
    logger.error(`Missing required configuration fields: ${missingFields.join(', ')}`);
    return false;
  }

  // Validate repositories array
  if (!Array.isArray(config.repositories) || config.repositories.length === 0) {
    logger.error('Configuration must include at least one repository');
    return false;
  }

  // Validate repository format (owner/repo)
  const invalidRepos = config.repositories.filter(repo => {
    return typeof repo !== 'string' || !repo.includes('/');
  });

  if (invalidRepos.length > 0) {
    logger.error(`Invalid repository format: ${invalidRepos.join(', ')}`);
    return false;
  }

  // Validate metrics configuration
  if (!config.metrics || typeof config.metrics !== 'object') {
    logger.error('Configuration must include metrics configuration');
    return false;
  }

  // Validate that at least one metric is enabled
  const enabledMetrics = Object.values(config.metrics).filter(metric => metric.enabled);
  if (enabledMetrics.length === 0) {
    logger.error('At least one metric must be enabled');
    return false;
  }

  // Validate metric configurations
  for (const [metricName, metricConfig] of Object.entries(config.metrics)) {
    if (metricConfig.enabled) {
      if (!metricConfig.tableName || typeof metricConfig.tableName !== 'string') {
        logger.error(`Metric ${metricName} must have a valid tableName`);
        return false;
      }
    }
  }

  // Validate userGroupFilepath when userGroupEnabled is true
  if (config.userGroupEnabled) {
    if (!config.userGroupFilepath) {
      logger.error('userGroupFilepath must be specified when userGroupEnabled is true');
      return false;
    }

    const resolvedUserGroupPath = path.resolve(process.cwd(), config.userGroupFilepath);
    if (!fs.existsSync(resolvedUserGroupPath)) {
      logger.error(`User group file not found at ${resolvedUserGroupPath}`);
      return false;
    }
  }

  return true;
};

/**
 * Loads and validates configuration
 * @param {string} [configPath='config.json'] - Path to the configuration file
 * @returns {Object} Configuration object
 */
export const loadConfig = (configPath = 'config.json') => {
  // Load configuration from file
  const fileConfig = loadConfigFromFile(configPath);

  // Load configuration from environment variables
  const envConfig = loadConfigFromEnv();

  // Merge configurations with precedence: env > file > default
  const config = {
    ...DEFAULT_CONFIG,
    ...fileConfig,
    ...envConfig
  };

  // Filter out undefined values
  Object.keys(config).forEach(key => {
    if (config[key] === undefined) {
      delete config[key];
    }
  });

  // Validate configuration
  const isValid = validateConfig(config);

  if (!isValid) {
    throw new Error('Invalid configuration');
  }

  logger.info('Configuration loaded successfully', {
    repositories: config.repositories,
    targetBranch: config.targetBranch,
    printOnly: config.printOnly,
    metrics: Object.fromEntries(
      Object.entries(config.metrics).map(([key, value]) => [key, { enabled: value.enabled, tableName: value.tableName }])
    ),
    ...(config.printOnly ? {} : {
      bigQueryDatasetId: config.bigQueryDatasetId
    })
  });

  return config;
};

export { validateConfig };

export default {
  loadConfig
};
