/**
 * Logger module for engineering metrics collector
 * Provides structured JSON logging
 */

/**
 * Log levels
 */
const LOG_LEVELS = {
  DEBUG: 'debug',
  INFO: 'info',
  WARN: 'warn',
  ERROR: 'error'
};

/**
 * Creates a log entry with the specified level, message, and optional data
 * @param {string} level - Log level
 * @param {string} message - Log message
 * @param {Object} [data] - Optional data to include in the log
 * @param {Error} [error] - Optional error object
 * @returns {Object} Log entry object
 */
const createLogEntry = (level, message, data = {}, error = null) => {
  const logEntry = {
    timestamp: new Date().toISOString(),
    level,
    message,
    ...data
  };

  if (error) {
    logEntry.error = {
      name: error.name,
      message: error.message,
      stack: error.stack
    };
  }

  return logEntry;
};

/**
 * Logs a message at the specified level
 * @param {string} level - Log level
 * @param {string} message - Log message
 * @param {Object} [data] - Optional data to include in the log
 * @param {Error} [error] - Optional error object
 */
const log = (level, message, data = {}, error = null) => {
  const logEntry = createLogEntry(level, message, data, error);
  console.log(JSON.stringify(logEntry));
};

/**
 * Logs a debug message
 * @param {string} message - Log message
 * @param {Object} [data] - Optional data to include in the log
 */
export const debug = (message, data = {}) => {
  log(LOG_LEVELS.DEBUG, message, data);
};

/**
 * Logs an info message
 * @param {string} message - Log message
 * @param {Object} [data] - Optional data to include in the log
 */
export const info = (message, data = {}) => {
  log(LOG_LEVELS.INFO, message, data);
};

/**
 * Logs a warning message
 * @param {string} message - Log message
 * @param {Object} [data] - Optional data to include in the log
 */
export const warn = (message, data = {}) => {
  log(LOG_LEVELS.WARN, message, data);
};

/**
 * Logs an error message
 * @param {string} message - Log message
 * @param {Error} [error] - Optional error object
 * @param {Object} [data] - Optional data to include in the log
 */
export const error = (message, error = new Error(message), data = {}) => {
  log(LOG_LEVELS.ERROR, message, data, error);
};

export default {
  debug,
  info,
  warn,
  error
};
