/**
 * General tests for GitHub client functionality
 */

import { jest } from '@jest/globals';
import GitHubClient from '../src/github-client.js';

// Mock the logger
jest.mock('../src/logger.js', () => ({
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn()
}));

describe('GitHubClient - General Functionality', () => {
  let githubClient;
  let mockOctokit;

  beforeEach(() => {
    // Mock Octokit
    mockOctokit = {
      rest: {
        pulls: {
          list: jest.fn(),
          listReviews: jest.fn()
        },
        issues: {
          listEventsForTimeline: jest.fn()
        }
      }
    };

    githubClient = new GitHubClient('fake-token');
    githubClient.octokit = mockOctokit;
  });

  describe('constructor', () => {
    test('should initialize with token', () => {
      const client = new GitHubClient('test-token');
      expect(client).toBeInstanceOf(GitHubClient);
    });

    test('should throw error without token', () => {
      expect(() => new GitHubClient()).toThrow('GitHub token is required');
    });
  });

  describe('fetchPullRequests', () => {
    test('should fetch pull requests successfully', async () => {
      const mockResponse = {
        data: [
          {
            number: 1,
            title: 'Test PR',
            state: 'closed',
            merged_at: '2023-06-15T14:30:00Z',
            updated_at: '2023-06-15T14:30:00Z',
            base: { ref: 'main' }
          }
        ]
      };

      mockOctokit.rest.pulls.list.mockResolvedValue(mockResponse);

      const result = await githubClient.fetchPullRequests('owner', 'repo', 'all', new Date('2023-01-01'));
      expect(result).toHaveLength(1);
      expect(result[0].number).toBe(1);
    });

    test('should handle API errors gracefully', async () => {
      const mockError = new Error('API Error');
      mockOctokit.rest.pulls.list.mockRejectedValue(mockError);

      await expect(githubClient.fetchPullRequests('owner', 'repo', 'all', new Date('2023-01-01')))
        .rejects.toThrow('API Error');
    });
  });

  describe('fetchPRTimelineEvents', () => {
    test('should fetch timeline events successfully', async () => {
      const mockResponse = {
        data: [
          {
            event: 'ready_for_review',
            created_at: '2023-06-15T10:00:00Z'
          }
        ]
      };

      mockOctokit.rest.issues.listEventsForTimeline.mockResolvedValue(mockResponse);

      const result = await githubClient.fetchPRTimelineEvents('owner', 'repo', 123);
      expect(result).toHaveLength(1);
      expect(result[0].event).toBe('ready_for_review');
    });

    test('should handle API errors gracefully', async () => {
      const mockError = new Error('Timeline API Error');
      mockOctokit.rest.issues.listEventsForTimeline.mockRejectedValue(mockError);

      await expect(githubClient.fetchPRTimelineEvents('owner', 'repo', 123))
        .rejects.toThrow('Timeline API Error');
    });
  });

  describe('fetchPRReviewEvents', () => {
    test('should fetch reviews successfully', async () => {
      const mockResponse = {
        data: [
          {
            id: 1,
            state: 'APPROVED',
            submitted_at: '2023-06-15T12:00:00Z',
            user: { login: 'reviewer1' }
          }
        ]
      };

      mockOctokit.rest.pulls.listReviews.mockResolvedValue(mockResponse);

      const result = await githubClient.fetchPRReviewEvents('owner', 'repo', 123);
      expect(result).toHaveLength(1);
      expect(result[0].state).toBe('APPROVED');
    });

    test('should handle API errors gracefully', async () => {
      const mockError = new Error('Reviews API Error');
      mockOctokit.rest.pulls.listReviews.mockRejectedValue(mockError);

      await expect(githubClient.fetchPRReviewEvents('owner', 'repo', 123))
        .rejects.toThrow('Reviews API Error');
    });
  });

});
