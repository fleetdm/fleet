/**
 * GitHub client module for engineering metrics collector
 * Handles interactions with the GitHub API using Octokit.js
 */

import { Octokit } from 'octokit';
import logger from './logger.js';

/**
 * Identifies if a GitHub user is likely a bot
 * @param {Object} user - GitHub user object
 * @returns {Object} Bot analysis result
 */
function identifyBotUser(user) {
  const botIndicators = {
    isBot: false,
    confidence: 'low',
    reasons: [],
  };

  // Check GitHub's bot flag (most reliable)
  if (user.type === 'Bot') {
    botIndicators.isBot = true;
    botIndicators.confidence = 'high';
    botIndicators.reasons.push('GitHub API type is "Bot"');
    return botIndicators;
  }

  // Check username patterns
  const username = user.login.toLowerCase();
  const botPatterns = [
    /\[bot]/, // contains '[bot]'
    /^dependabot/, // dependabot
    /^renovate/, // renovate bot
    /^github-actions/, // GitHub Actions
    /^codecov/, // codecov bot
    /^coderabbitai/, // coderabbit AI bot
    /^sonarcloud/, // sonarcloud bot
    /^snyk/, // snyk bot
    /^greenkeeper/, // greenkeeper bot
    /^semantic-release/, // semantic-release bot
    /^stale/, // stale bot
    /^imgbot/, // imgbot
    /^allcontributors/, // all-contributors bot
    /^whitesource/, // whitesource bot
    /^deepsource/, // deepsource bot
  ];

  for (const pattern of botPatterns) {
    if (pattern.test(username)) {
      botIndicators.isBot = true;
      botIndicators.confidence = 'high';
      botIndicators.reasons.push(`Username matches bot pattern: ${pattern}`);
      break;
    }
  }

  return botIndicators;
}

/**
 * GitHub client class
 */
export class GitHubClient {
  /**
   * Creates a new GitHub client
   * @param {string} token - GitHub API token
   */
  constructor(token) {
    if (!token) {
      throw new Error('GitHub token is required');
    }
    this.octokit = null;
    this.initialize(token);
  }

  /**
   * Initializes the GitHub client
   * @param {string} token - GitHub API token
   */
  initialize(token) {
    try {
      this.octokit = new Octokit({
        auth: token,
      });
      logger.info('GitHub client initialized');
    } catch (err) {
      logger.error('Failed to initialize GitHub client', {}, err);
      throw err;
    }
  }

  /**
   * Fetches pull requests for a repository
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {string} state - PR state (open, closed, all)
   * @param {Date} since - Fetch PRs updated since this date
   * @param {string} targetBranch - Target branch to filter PRs by
   * @returns {Array} Array of pull requests
   */
  async fetchPullRequests(
    owner,
    repo,
    state = 'all',
    since,
    targetBranch = 'main'
  ) {
    try {
      logger.info(
        `Fetching ${state} PRs for ${owner}/${repo} since ${since.toISOString()}`
      );

      // GitHub API returns paginated results, so we need to fetch all pages
      const pullRequests = [];
      let page = 1;
      let hasMorePages = true;

      while (hasMorePages) {
        const response = await this.octokit.rest.pulls.list({
          owner,
          repo,
          state,
          sort: 'updated',
          direction: 'desc',
          per_page: 100,
          page,
        });

        // Filter PRs by update date and target branch
        const filteredPRs = response.data.filter((pr) => {
          const prUpdatedAt = new Date(pr.updated_at);
          return prUpdatedAt >= since && pr.base.ref === targetBranch;
        });

        if (filteredPRs.length > 0) {
          pullRequests.push(...filteredPRs);
          page++;
        } else {
          hasMorePages = false;
        }

        // If we got fewer results than the page size, there are no more pages
        if (response.data.length < 100) {
          hasMorePages = false;
        }
      }

      logger.info(`Fetched ${pullRequests.length} PRs for ${owner}/${repo}`);
      return pullRequests;
    } catch (err) {
      logger.error(`Error fetching PRs for ${owner}/${repo}`, {}, err);

      // Implement basic retry for rate limiting
      if (
        err.status === 403 &&
        err.response?.headers?.['x-ratelimit-remaining'] === '0'
      ) {
        const resetTime =
          parseInt(err.response.headers['x-ratelimit-reset'], 10) * 1000;
        const waitTime = resetTime - Date.now();

        if (waitTime > 0 && waitTime < 3600000) {
          // Only retry if wait time is less than 1 hour
          logger.info(
            `Rate limit exceeded. Retrying in ${Math.ceil(
              waitTime / 1000
            )} seconds`
          );
          await new Promise((resolve) => setTimeout(resolve, waitTime + 1000));
          return this.fetchPullRequests(
            owner,
            repo,
            state,
            since,
            targetBranch
          );
        }
      }

      throw err;
    }
  }

  /**
   * Fetches PR review events
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {number} prNumber - PR number
   * @returns {Array} Array of review events
   */
  async fetchPRReviewEvents(owner, repo, prNumber) {
    try {
      logger.info(`Fetching review events for ${owner}/${repo}#${prNumber}`);

      const response = await this.octokit.rest.pulls.listReviews({
        owner,
        repo,
        pull_number: prNumber,
      });

      logger.info(
        `Fetched ${response.data.length} review events for ${owner}/${repo}#${prNumber}`
      );
      return response.data;
    } catch (err) {
      logger.error(
        `Error fetching review events for ${owner}/${repo}#${prNumber}`,
        {},
        err
      );

      // Implement basic retry for rate limiting
      if (
        err.status === 403 &&
        err.response?.headers?.['x-ratelimit-remaining'] === '0'
      ) {
        const resetTime =
          parseInt(err.response.headers['x-ratelimit-reset'], 10) * 1000;
        const waitTime = resetTime - Date.now();

        if (waitTime > 0 && waitTime < 3600000) {
          // Only retry if wait time is less than 1 hour
          logger.info(
            `Rate limit exceeded. Retrying in ${Math.ceil(
              waitTime / 1000
            )} seconds`
          );
          await new Promise((resolve) => setTimeout(resolve, waitTime + 1000));
          return this.fetchPRReviewEvents(owner, repo, prNumber);
        }
      }

      throw err;
    }
  }

  /**
   * Filters out bot reviews from review events
   * @param {Array} reviewEvents - Array of review events
   * @param {boolean} excludeBots - Whether to exclude bot reviews (default: false)
   * @returns {Array} Filtered review events
   */
  filterBotReviews(reviewEvents, excludeBots = false) {
    if (!excludeBots) {
      return reviewEvents;
    }

    const filteredReviews = reviewEvents.filter((review) => {
      const botAnalysis = identifyBotUser(review.user);
      if (botAnalysis.isBot) {
        logger.debug(`Filtering out bot review from ${review.user.login}`, {
          confidence: botAnalysis.confidence,
          reasons: botAnalysis.reasons,
        });
        return false;
      }
      return true;
    });

    const botCount = reviewEvents.length - filteredReviews.length;
    if (botCount > 0) {
      logger.info(
        `Filtered out ${botCount} bot reviews from ${reviewEvents.length} total reviews`
      );
    }

    return filteredReviews;
  }

  /**
   * Fetches PR timeline events
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {number} prNumber - PR number
   * @returns {Array} Array of timeline events
   */
  async fetchPRTimelineEvents(owner, repo, prNumber) {
    try {
      logger.info(`Fetching timeline events for ${owner}/${repo}#${prNumber}`);

      // GitHub API returns paginated results, so we need to fetch all pages
      const timelineEvents = [];
      let page = 1;
      let hasMorePages = true;

      while (hasMorePages) {
        const response = await this.octokit.rest.issues.listEventsForTimeline({
          owner,
          repo,
          issue_number: prNumber,
          per_page: 100,
          page,
        });

        if (response.data.length > 0) {
          timelineEvents.push(...response.data);
          page++;
        } else {
          hasMorePages = false;
        }

        // If we got fewer results than the page size, there are no more pages
        if (response.data.length < 100) {
          hasMorePages = false;
        }
      }

      logger.info(
        `Fetched ${timelineEvents.length} timeline events for ${owner}/${repo}#${prNumber}`
      );
      return timelineEvents;
    } catch (err) {
      logger.error(
        `Error fetching timeline events for ${owner}/${repo}#${prNumber}`,
        {},
        err
      );

      // Implement basic retry for rate limiting
      if (
        err.status === 403 &&
        err.response?.headers?.['x-ratelimit-remaining'] === '0'
      ) {
        const resetTime =
          parseInt(err.response.headers['x-ratelimit-reset'], 10) * 1000;
        const waitTime = resetTime - Date.now();

        if (waitTime > 0 && waitTime < 3600000) {
          // Only retry if wait time is less than 1 hour
          logger.info(
            `Rate limit exceeded. Retrying in ${Math.ceil(
              waitTime / 1000
            )} seconds`
          );
          await new Promise((resolve) => setTimeout(resolve, waitTime + 1000));
          return this.fetchPRTimelineEvents(owner, repo, prNumber);
        }
      }

      throw err;
    }
  }

  /**
   * Calculates pickup time for a PR
   * @param {Object} pr - Pull request object
   * @param {Array} timelineEvents - PR timeline events
   * @param {Array} reviewEvents - PR review events
   * @returns {Object} Pickup time metrics
   */
  calculatePickupTime(pr, timelineEvents, reviewEvents) {
    try {
      const result = this.getReadyAndFirstReview(
        pr,
        timelineEvents,
        reviewEvents
      );
      if (!result || !result.firstReviewTime) {
        return null;
      }
      const { relevantReadyEvent, firstReviewTime } = result;
      const readyTime = relevantReadyEvent.time;

      // Calculate pickup time excluding weekends
      const pickupTimeSeconds = this.calculatePickupTimeExcludingWeekends(
        readyTime,
        firstReviewTime
      );

      // If pickup time is negative, something went wrong
      if (pickupTimeSeconds < 0) {
        logger.warn(`Negative pickup time for ${pr.html_url}`, {
          readyTime,
          firstReviewTime,
          pickupTimeSeconds,
        });
        return null;
      }

      // Log which ready event was used
      const readyEventType =
        relevantReadyEvent.event.event === 'created_not_draft'
          ? 'PR creation (not draft)'
          : 'ready_for_review event';

      logger.info(`Calculated pickup time for ${pr.html_url}`, {
        pickupTimeSeconds,
        readyEventType,
        readyTime: readyTime.toISOString(),
        firstReviewTime: firstReviewTime.toISOString(),
      });

      // We already have readyEventType defined above, so we can use it here

      return {
        metricType: 'time_to_first_review',
        repository: `${pr.base.repo.owner.login}/${pr.base.repo.name}`,
        prNumber: pr.number,
        prUrl: pr.html_url,
        prCreator: pr.user.login,
        targetBranch: pr.base.ref,
        readyTime,
        firstReviewTime,
        reviewDate: firstReviewTime.toISOString().split('T')[0], // YYYY-MM-DD
        pickupTimeSeconds,
        readyEventType,
      };
    } catch (err) {
      logger.error(`Error calculating pickup time for ${pr.html_url}`, {}, err);
      return null;
    }
  }

  /**
   * Calculates pickup time for a PR
   * @param {Object} pr - Pull request object
   * @param {Array} timelineEvents - PR timeline events
   * @param {Array} reviewEvents - PR review events
   * @returns {Object} ready event and first review time
   */
  getReadyAndFirstReview(pr, timelineEvents, reviewEvents) {
    const mergeTime = pr.merged_at ? new Date(pr.merged_at) : null;

    // Find all ready_for_review events that occurred before merge time (if merged)
    const readyForReviewEvents = timelineEvents
      .filter((event) => event.event === 'ready_for_review')
      .map((event) => ({
        time: new Date(event.created_at),
        event,
      }))
      .filter((readyEvent) => !mergeTime || readyEvent.time <= mergeTime);

    // Add PR creation time as a ready event if PR was not created as draft
    if (!pr.draft) {
      readyForReviewEvents.push({
        time: new Date(pr.created_at),
        event: { event: 'created_not_draft', created_at: pr.created_at },
      });
    }

    // Sort ready events by time (ascending)
    readyForReviewEvents.sort((a, b) => a.time - b.time);

    // If we couldn't find any ready events, return null
    if (readyForReviewEvents.length === 0) {
      logger.warn(`No ready_for_review events found for ${pr.html_url}`);
      return null;
    }

    // If there is no review events, the PR may have been merged without a review.
    if (reviewEvents.length === 0) {
      const relevantReadyEvent =
        readyForReviewEvents[readyForReviewEvents.length - 1];
      return {
        relevantReadyEvent,
        firstReviewTime: null,
      };
    }

    // Sort review events by submitted_at (ascending)
    const sortedReviewEvents = [...reviewEvents].sort(
      (a, b) => new Date(a.submitted_at) - new Date(b.submitted_at)
    );

    const firstReview = sortedReviewEvents[0];
    const firstReviewTime = new Date(firstReview.submitted_at);

    // Find the most recent ready event that occurred before the first review
    const relevantReadyEvent = readyForReviewEvents
      .filter((readyEvent) => readyEvent.time < firstReviewTime)
      .pop();

    // If no ready event occurred before the first review, return null
    if (!relevantReadyEvent) {
      logger.warn(
        `No ready_for_review event found before first review for ${pr.html_url}`
      );
      return null;
    }

    return {
      relevantReadyEvent,
      firstReviewTime,
    };
  }

  /**
   * Calculates pickup time excluding weekends
   * @param {Date} readyTimeOrig - Time when PR was marked as ready for review
   * @param {Date} reviewTimeOrig - Time when the first review occurred
   * @returns {number} Pickup time in seconds, excluding weekends
   */
  calculatePickupTimeExcludingWeekends(readyTimeOrig, reviewTimeOrig) {
    const readyTime = new Date(readyTimeOrig);
    const reviewTime = new Date(reviewTimeOrig);

    // Get day of week (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
    const readyDay = readyTime.getUTCDay();
    const reviewDay = reviewTime.getUTCDay();

    // Case: Both ready time and review time are on the same weekend
    if (
      (readyDay === 0 || readyDay === 6) &&
      (reviewDay === 0 || reviewDay === 6) &&
      Math.floor(reviewTime / (24 * 60 * 60 * 1000)) -
        Math.floor(readyTime / (24 * 60 * 60 * 1000)) <=
        2
    ) {
      // Return 0 seconds pickup time
      return 0;
    }

    // Set to start of Monday if ready time is on weekend
    if (readyDay === 0) {
      // Sunday
      readyTime.setUTCDate(readyTime.getUTCDate() + 1);
      readyTime.setUTCHours(0, 0, 0, 0);
    } else if (readyDay === 6) {
      // Saturday
      readyTime.setUTCDate(readyTime.getUTCDate() + 2);
      readyTime.setUTCHours(0, 0, 0, 0);
    }
    // Set to start of Saturday if review time is on Sunday
    if (reviewDay === 0) {
      // Sunday
      reviewTime.setUTCDate(reviewTime.getUTCDate() - 1);
      reviewTime.setUTCHours(0, 0, 0, 0);
    } else if (reviewDay === 6) {
      // Saturday
      reviewTime.setUTCHours(0, 0, 0, 0);
    }

    // Calculate raw time difference in milliseconds
    const weekendDays = countWeekendDays(readyTime, reviewTime);
    const diffMs = reviewTime - readyTime - weekendDays * 24 * 60 * 60 * 1000;

    // Ensure we don't return negative values
    return Math.max(0, Math.floor(diffMs / 1000));
  }

  /**
   * Calculate time to merge metrics
   * @param {Object} pr - Pull request object
   * @param {Array} timelineEvents - Timeline events
   * @param {Array} reviewEvents - PR review events
   * @returns {Object|null} Time to merge metrics or null if not applicable
   */
  calculateTimeToMerge(pr, timelineEvents, reviewEvents) {
    try {
      // Only process merged PRs
      if (!pr.merged_at) {
        return null;
      }

      // Find the ready time using the same algorithm as we use for Time to First Review
      const result = this.getReadyAndFirstReview(
        pr,
        timelineEvents,
        reviewEvents
      );
      if (!result) {
        return null;
      }
      const relevantReadyEvent = result.relevantReadyEvent;
      const readyTime = relevantReadyEvent.time;
      const mergeTime = new Date(pr.merged_at);

      // Calculate merge time excluding weekends
      const mergeTimeSeconds = this.calculatePickupTimeExcludingWeekends(
        readyTime,
        mergeTime
      );

      // If merge time is negative, something went wrong
      if (mergeTimeSeconds < 0) {
        logger.warn(`Negative merge time for ${pr.html_url}`, {
          readyTime,
          mergeTime,
          mergeTimeSeconds,
        });
        return null;
      }

      // Log which ready event was used
      const readyEventType =
        relevantReadyEvent.event.event === 'created_not_draft'
          ? 'PR creation (not draft)'
          : 'ready_for_review event';

      logger.info(`Calculated merge time for ${pr.html_url}`, {
        mergeTimeSeconds,
        readyEventType,
        readyTime: readyTime.toISOString(),
        mergeTime: mergeTime.toISOString(),
      });

      return {
        metricType: 'time_to_merge',
        repository: `${pr.base.repo.owner.login}/${pr.base.repo.name}`,
        prNumber: pr.number,
        prUrl: pr.html_url,
        prCreator: pr.user.login,
        targetBranch: pr.base.ref,
        readyTime,
        mergeTime,
        mergeDate: mergeTime.toISOString().split('T')[0], // YYYY-MM-DD
        mergeTimeSeconds,
        readyEventType,
      };
    } catch (err) {
      logger.error(`Error calculating merge time for ${pr.html_url}`, {}, err);
      return null;
    }
  }
}

function countWeekendDays(startDate, endDate) {
  // Make local copies of dates
  startDate = new Date(startDate);
  endDate = new Date(endDate);

  // Ensure startDate is before endDate
  if (startDate > endDate) {
    [startDate, endDate] = [endDate, startDate];
  }

  // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
  if (startDate.getUTCDay() === 0) {
    startDate.setUTCDate(startDate.getUTCDate() + 1);
  } else if (startDate.getUTCDay() === 6) {
    startDate.setUTCDate(startDate.getUTCDate() + 2);
  }
  if (endDate.getUTCDay() === 0) {
    endDate.setUTCDate(endDate.getUTCDate() - 2);
  } else if (endDate.getUTCDay() === 6) {
    endDate.setUTCDate(endDate.getUTCDate() - 1);
  }

  let count = 0;
  const current = new Date(startDate);

  while (current <= endDate) {
    const day = current.getUTCDay();
    if (day === 0 || day === 6) {
      // Sunday (0) or Saturday (6)
      count++;
    }
    current.setUTCDate(current.getUTCDate() + 1);
  }

  return count;
}

export default GitHubClient;
