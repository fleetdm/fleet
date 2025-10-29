/**
 * GitHub username validator
 * Validates that extracted usernames are real GitHub accounts
 */

import { Octokit } from 'octokit';
import logger from './logger.js';

/**
 * Validates a single GitHub username
 * @param {Octokit} octokit - GitHub API client
 * @param {string} username - GitHub username to validate
 * @returns {Promise<boolean>} True if username exists, false otherwise
 */
const validateUsername = async (octokit, username) => {
  try {
    await octokit.rest.users.getByUsername({ username });
    return true;
  } catch (error) {
    if (error.status === 404) {
      logger.warn(`GitHub username not found: ${username}`);
      return false;
    }

    // For other errors (rate limiting, network issues), log but assume valid
    logger.warn(`Error validating username ${username}: ${error.message}`);
    return true; // Assume valid to avoid false negatives
  }
};

/**
 * Validates multiple GitHub usernames
 * @param {string} githubToken - GitHub API token
 * @param {Array<string>} usernames - Array of usernames to validate
 * @returns {Promise<Array<string>>} Array of valid usernames
 */
export const validateUsernames = async (githubToken, usernames) => {
  if (!githubToken) {
    throw new Error('GitHub token is required for username validation');
  }

  const octokit = new Octokit({ auth: githubToken });
  const validUsernames = [];
  const invalidUsernames = [];

  logger.info(`Validating ${usernames.length} GitHub usernames...`);

  // Process usernames with a small delay to respect rate limits
  for (const username of usernames) {
    const isValid = await validateUsername(octokit, username);

    if (isValid) {
      validUsernames.push(username);
    } else {
      invalidUsernames.push(username);
    }

    // Small delay to avoid hitting rate limits too aggressively.
    // 2025/07/03: GitHub's authenticated rate limit is 5000 requests/hour (~1.4 requests/second). This could lead to rate limit errors with larger username lists.
    await new Promise((resolve) => setTimeout(resolve, 100));
  }

  if (invalidUsernames.length > 0) {
    logger.warn(
      `Found ${
        invalidUsernames.length
      } invalid GitHub usernames: ${invalidUsernames.join(', ')}`
    );
  }

  logger.info(
    `Validated ${validUsernames.length} out of ${usernames.length} GitHub usernames`
  );
  return validUsernames;
};

/**
 * Filters user groups to only include valid usernames
 * @param {string} githubToken - GitHub API token
 * @param {Array<{group: string, username: string}>} userGroups - Array of user group mappings
 * @returns {Promise<Array<{group: string, username: string}>>} Array of user group mappings with valid usernames only
 */
export const filterValidUserGroups = async (githubToken, userGroups) => {
  // Get unique usernames for validation
  const uniqueUsernames = [...new Set(userGroups.map((ug) => ug.username))];

  // Validate usernames
  const validUsernames = await validateUsernames(githubToken, uniqueUsernames);
  const validUsernameSet = new Set(validUsernames);

  // Filter user groups to only include valid usernames
  const validUserGroups = userGroups.filter((ug) =>
    validUsernameSet.has(ug.username)
  );

  const removedCount = userGroups.length - validUserGroups.length;
  if (removedCount > 0) {
    logger.info(
      `Removed ${removedCount} user group mappings due to invalid usernames`
    );
  }

  return validUserGroups;
};

export default {
  validateUsernames,
  filterValidUserGroups,
};
