#!/usr/bin/env node

/**
 * Main entry point for engineering metrics collector
 * Collects comprehensive GitHub engineering metrics including:
 * - Time to First Review (currently implemented)
 * - Time to Merge (planned)
 * - Time to QA Ready (planned)
 * - Time to Production Ready (planned)
 */

import { loadConfig } from './config.js';
import { MetricsCollector } from './metrics-collector.js';
import logger from './logger.js';
import { fileURLToPath } from 'url';
import { dirname, join, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * Main function
 */
async function main() {
  try {
    // Set the working directory to one level up from the current file location, which is the root of the action.
    process.chdir(resolve(__dirname, '..'));

    // Parse command line arguments
    const args = process.argv.slice(2);

    // Check for print-only flag in command line arguments
    const printOnlyFlag = args.includes('--print-only');

    // Get the configuration path from command line arguments
    // Filter out the --print-only flag if present
    const configPath =
      args.filter((arg) => arg !== '--print-only')[0] ||
      join(__dirname, '..', 'config.json');

    // Load configuration
    const config = loadConfig(configPath);

    // Override printOnly setting if flag is provided
    if (printOnlyFlag) {
      config.printOnly = true;
    }

    // Create and run metrics collector
    const metricsCollector = new MetricsCollector(config);
    const metrics = await metricsCollector.run();

    if (config.printOnly) {
      logger.info(
        `Successfully collected and printed ${metrics.length} engineering metrics`
      );
    } else {
      logger.info(
        `Successfully collected and uploaded ${metrics.length} engineering metrics to BigQuery`
      );
    }

    // Exit with success
    process.exit(0);
  } catch (err) {
    logger.error('Error running engineering metrics collector', {}, err);

    // Exit with error
    process.exit(1);
  }
}

// Run the main function
await main();
