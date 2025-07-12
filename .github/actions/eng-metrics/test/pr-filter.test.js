/**
 * Tests for bot detection functionality
 */

import { jest } from '@jest/globals';

// Mock the logger
const mockLogger = {
  info: jest.fn(),
  debug: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
};

jest.unstable_mockModule('../src/logger.js', () => ({
  default: mockLogger
}));

const { GitHubClient } = await import('../src/github-client.js');

describe('Bot Detection', () => {
  let githubClient;

  beforeEach(() => {
    githubClient = new GitHubClient('fake-token');
    jest.clearAllMocks();
  });

  describe('identifyBotUser', () => {
    test('should identify GitHub API Bot type', () => {
      const botUser = {
        login: 'coderabbitai[bot]',
        type: 'Bot',
        name: 'CodeRabbit AI',
        bio: 'AI-powered code review assistant'
      };

      // Access the function through the module (it's not exported as a method)
      // We'll test it indirectly through filterBotReviews
      const reviews = [
        {
          user: botUser,
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, true);
      expect(filtered).toHaveLength(0);
    });

    test('should identify bot by username patterns', () => {
      const testCases = [
        { login: 'dependabot[bot]', type: 'User' },
        { login: 'renovate[bot]', type: 'User' },
        { login: 'github-actions[bot]', type: 'User' },
        { login: 'codecov-commenter', type: 'User' },
        { login: 'coderabbitai', type: 'User' },
        { login: 'sonarcloud[bot]', type: 'User' },
        { login: 'snyk-bot', type: 'User' },
        { login: 'greenkeeper[bot]', type: 'User' },
        { login: 'semantic-release-bot', type: 'User' },
        { login: 'stale[bot]', type: 'User' },
        { login: 'imgbot[bot]', type: 'User' },
        { login: 'allcontributors[bot]', type: 'User' },
        { login: 'whitesource-bolt', type: 'User' },
        { login: 'deepsource-autofix[bot]', type: 'User' }
      ];

      testCases.forEach(({ login, type }) => {
        const reviews = [
          {
            user: { login, type },
            state: 'COMMENTED',
            submitted_at: '2023-06-15T12:00:00Z'
          }
        ];

        const filtered = githubClient.filterBotReviews(reviews, true);
        expect(filtered).toHaveLength(0);
      });
    });

    test('should not filter human users', () => {
      const humanUser = {
        login: 'johndoe',
        type: 'User',
        name: 'John Doe',
        bio: 'Software engineer'
      };

      const reviews = [
        {
          user: humanUser,
          state: 'APPROVED',
          submitted_at: '2023-06-15T12:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, true);
      expect(filtered).toHaveLength(1);
      expect(filtered[0].user.login).toBe('johndoe');
    });

    test('should not filter when excludeBots is false', () => {
      const botUser = {
        login: 'coderabbitai[bot]',
        type: 'Bot'
      };

      const humanUser = {
        login: 'johndoe',
        type: 'User'
      };

      const reviews = [
        {
          user: botUser,
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: humanUser,
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, false);
      expect(filtered).toHaveLength(2);
    });

    test('should handle mixed bot and human reviews', () => {
      const reviews = [
        {
          user: { login: 'coderabbitai[bot]', type: 'Bot' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'johndoe', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        },
        {
          user: { login: 'dependabot[bot]', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T14:00:00Z'
        },
        {
          user: { login: 'janedoe', type: 'User' },
          state: 'CHANGES_REQUESTED',
          submitted_at: '2023-06-15T15:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, true);
      expect(filtered).toHaveLength(2);
      expect(filtered[0].user.login).toBe('johndoe');
      expect(filtered[1].user.login).toBe('janedoe');
    });

    test('should log filtering activity', () => {
      const reviews = [
        {
          user: { login: 'coderabbitai[bot]', type: 'Bot' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'johndoe', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        }
      ];

      githubClient.filterBotReviews(reviews, true);
      
      expect(mockLogger.info).toHaveBeenCalledWith(
        'Filtered out 1 reviews (1 bot reviews) from 2 total reviews'
      );
    });

    test('should handle empty review array', () => {
      const filtered = githubClient.filterBotReviews([], true);
      expect(filtered).toHaveLength(0);
    });

    test('should handle reviews with no bots', () => {
      const reviews = [
        {
          user: { login: 'johndoe', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        },
        {
          user: { login: 'janedoe', type: 'User' },
          state: 'CHANGES_REQUESTED',
          submitted_at: '2023-06-15T15:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, true);
      expect(filtered).toHaveLength(2);
      expect(mockLogger.info).not.toHaveBeenCalledWith(
        expect.stringContaining('Filtered out')
      );
    });
  });

  describe('PR Creator Filtering', () => {
    test('should filter out PR creator reviews', () => {
      const prCreator = { login: 'prauthor', type: 'User' };
      const reviews = [
        {
          user: { login: 'prauthor', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'reviewer1', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        },
        {
          user: { login: 'reviewer2', type: 'User' },
          state: 'CHANGES_REQUESTED',
          submitted_at: '2023-06-15T14:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, false, prCreator);
      expect(filtered).toHaveLength(2);
      expect(filtered[0].user.login).toBe('reviewer1');
      expect(filtered[1].user.login).toBe('reviewer2');
      
      expect(mockLogger.info).toHaveBeenCalledWith(
        'Filtered out 1 reviews (1 PR creator reviews) from 3 total reviews'
      );
    });

    test('should filter out multiple PR creator reviews', () => {
      const prCreator = { login: 'prauthor', type: 'User' };
      const reviews = [
        {
          user: { login: 'prauthor', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'reviewer1', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        },
        {
          user: { login: 'prauthor', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T14:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, false, prCreator);
      expect(filtered).toHaveLength(1);
      expect(filtered[0].user.login).toBe('reviewer1');
      
      expect(mockLogger.info).toHaveBeenCalledWith(
        'Filtered out 2 reviews (2 PR creator reviews) from 3 total reviews'
      );
    });

    test('should not filter when prCreator is null', () => {
      const reviews = [
        {
          user: { login: 'prauthor', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'reviewer1', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T13:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, false, null);
      expect(filtered).toHaveLength(2);
      
      expect(mockLogger.info).not.toHaveBeenCalledWith(
        expect.stringContaining('Filtered out')
      );
    });

    test('should filter both bots and PR creator', () => {
      const prCreator = { login: 'prauthor', type: 'User' };
      const reviews = [
        {
          user: { login: 'prauthor', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T12:00:00Z'
        },
        {
          user: { login: 'coderabbitai[bot]', type: 'Bot' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T13:00:00Z'
        },
        {
          user: { login: 'reviewer1', type: 'User' },
          state: 'APPROVED',
          submitted_at: '2023-06-15T14:00:00Z'
        },
        {
          user: { login: 'dependabot[bot]', type: 'User' },
          state: 'COMMENTED',
          submitted_at: '2023-06-15T15:00:00Z'
        }
      ];

      const filtered = githubClient.filterBotReviews(reviews, true, prCreator);
      expect(filtered).toHaveLength(1);
      expect(filtered[0].user.login).toBe('reviewer1');
      
      expect(mockLogger.info).toHaveBeenCalledWith(
        'Filtered out 3 reviews (2 bot reviews, 1 PR creator reviews) from 4 total reviews'
      );
    });

  });
});