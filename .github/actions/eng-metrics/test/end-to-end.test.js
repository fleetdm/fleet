/**
 * Tests the complete workflow from configuration to metric collection
 */

import { jest } from '@jest/globals';
import { loadConfig } from '../src/config.js';
import { MetricsCollector } from '../src/metrics-collector.js';

// Mock the logger
jest.mock('../src/logger.js', () => ({
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn()
}));

// Mock GitHubClient
jest.mock('../src/github-client.js', () => {
  return jest.fn().mockImplementation(() => ({
    fetchPullRequests: jest.fn(),
    fetchPRTimelineEvents: jest.fn(),
    fetchPRReviewEvents: jest.fn(),
    filterBotReviews: jest.fn((reviews, excludeBots) => excludeBots ? [] : reviews),
    calculatePickupTime: jest.fn(),
    calculateTimeToMerge: jest.fn()
  }));
});

// Mock BigQueryClient
jest.mock('../src/bigquery-client.js', () => ({
  BigQueryClient: jest.fn().mockImplementation(() => ({
    uploadMetrics: jest.fn()
  }))
}));

describe('End-to-End Time to Merge Workflow', () => {
  beforeEach(() => {
    // Set environment variables for testing
    process.env.GITHUB_TOKEN = 'test-token';
    process.env.SERVICE_ACCOUNT_KEY_PATH = '/fake/path';
    process.env.BIGQUERY_DATASET_ID = 'test_dataset';
    process.env.REPOSITORIES = 'owner/repo';
    process.env.PRINT_ONLY = 'true';
  });

  test('should collect both Time to First Review and Time to Merge metrics', async () => {
    // Load configuration
    const config = loadConfig();

    // Verify both metrics are enabled
    expect(config.metrics.timeToFirstReview.enabled).toBe(true);
    expect(config.metrics.timeToMerge.enabled).toBe(true);

    // Create metrics collector
    const metricsCollector = new MetricsCollector(config);

    // Mock the GitHub client methods
    metricsCollector.githubClient = {
      fetchPullRequests: jest.fn().mockResolvedValue([
        {
          number: 123,
          html_url: 'https://github.com/owner/repo/pull/123',
          user: { login: 'testuser' },
          base: {
            ref: 'main',
            repo: {
              owner: { login: 'owner' },
              name: 'repo'
            }
          },
          head: { repo: { full_name: 'owner/repo' } },
          state: 'closed',
          merged_at: '2023-06-15T14:30:00Z'
        }
      ]),
      fetchPRTimelineEvents: jest.fn().mockResolvedValue([
        {
          event: 'ready_for_review',
          created_at: '2023-06-15T10:00:00Z'
        }
      ]),
      fetchPRReviewEvents: jest.fn().mockResolvedValue([
        {
          submitted_at: '2023-06-15T12:00:00Z',
          state: 'approved',
          user: { login: 'reviewer1' }
        }
      ]),
      calculatePickupTime: jest.fn().mockReturnValue({
        metricType: 'time_to_first_review',
        reviewDate: '2023-06-15',
        prCreator: 'testuser',
        prUrl: 'https://github.com/owner/repo/pull/123',
        pickupTimeSeconds: 7200,
        repository: 'owner/repo',
        prNumber: 123,
        targetBranch: 'main',
        readyTime: new Date('2023-06-15T10:00:00Z'),
        firstReviewTime: new Date('2023-06-15T12:00:00Z')
      }),
      calculateTimeToMerge: jest.fn().mockReturnValue({
        metricType: 'time_to_merge',
        mergeDate: '2023-06-15',
        prCreator: 'testuser',
        prUrl: 'https://github.com/owner/repo/pull/123',
        mergeTimeSeconds: 16200,
        repository: 'owner/repo',
        prNumber: 123,
        targetBranch: 'main',
        readyTime: new Date('2023-06-15T10:00:00Z'),
        mergeTime: new Date('2023-06-15T14:30:00Z')
      }),
      filterBotReviews: jest.fn((reviews, excludeBots) => excludeBots ? [] : reviews)
    };

    // Collect metrics for a single repository
    const metrics = await metricsCollector.collectRepositoryMetrics('owner/repo');

    // Verify that both metrics were collected
    expect(metrics).toHaveLength(2);

    const firstReviewMetric = metrics.find(m => m.metricType === 'time_to_first_review');
    const mergeMetric = metrics.find(m => m.metricType === 'time_to_merge');

    expect(firstReviewMetric).toBeDefined();
    expect(firstReviewMetric.pickupTimeSeconds).toBe(7200);
    expect(firstReviewMetric.prNumber).toBe(123);

    expect(mergeMetric).toBeDefined();
    expect(mergeMetric.mergeTimeSeconds).toBe(16200);
    expect(mergeMetric.prNumber).toBe(123);

    // Verify that both calculation methods were called
    expect(metricsCollector.githubClient.calculatePickupTime).toHaveBeenCalled();
    expect(metricsCollector.githubClient.calculateTimeToMerge).toHaveBeenCalled();
  });

  test('should handle configuration with only Time to Merge enabled', async () => {
    // Override environment to enable only Time to Merge
    process.env.ENABLED_METRICS = 'time_to_merge';

    const config = loadConfig();

    expect(config.metrics.timeToFirstReview.enabled).toBe(false);
    expect(config.metrics.timeToMerge.enabled).toBe(true);

    const metricsCollector = new MetricsCollector(config);

    // Mock the GitHub client
    metricsCollector.githubClient = {
      calculateTimeToMerge: jest.fn().mockReturnValue({
        metricType: 'time_to_merge',
        mergeDate: '2023-06-15',
        prCreator: 'testuser',
        prUrl: 'https://github.com/owner/repo/pull/123',
        mergeTimeSeconds: 16200,
        repository: 'owner/repo',
        prNumber: 123,
        targetBranch: 'main',
        readyTime: new Date('2023-06-15T10:00:00Z'),
        mergeTime: new Date('2023-06-15T14:30:00Z')
      }),
      calculatePickupTime: jest.fn().mockReturnValue(null) // Should not be called
    };

    // Mock PR data
    const mockPR = {
      number: 123,
      html_url: 'https://github.com/owner/repo/pull/123',
      user: { login: 'testuser' },
      base: {
        ref: 'main',
        repo: {
          owner: { login: 'owner' },
          name: 'repo'
        }
      },
      head: { repo: { full_name: 'owner/repo' } },
      state: 'closed',
      merged_at: '2023-06-15T14:30:00Z'
    };

    const mockTimelineEvents = [
      {
        event: 'ready_for_review',
        created_at: '2023-06-15T10:00:00Z'
      }
    ];

    const mockReviewEvents = [];

    // Test collectPRMetrics with only merge enabled
    const metrics = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

    // Should only collect Time to Merge metric
    expect(metrics).toHaveLength(1);
    expect(metrics[0].metricType).toBe('time_to_merge');
  });

  test('should group and route metrics to correct tables', () => {
    const config = loadConfig();
    const metricsCollector = new MetricsCollector(config);

    const mixedMetrics = [
      {
        metricType: 'time_to_first_review',
        prNumber: 123,
        pickupTimeSeconds: 7200
      },
      {
        metricType: 'time_to_merge',
        prNumber: 123,
        mergeTimeSeconds: 16200
      },
      {
        metricType: 'time_to_first_review',
        prNumber: 124,
        pickupTimeSeconds: 3600
      }
    ];

    // Test grouping
    const grouped = metricsCollector.groupMetricsByType(mixedMetrics);

    expect(grouped.time_to_first_review).toHaveLength(2);
    expect(grouped.time_to_merge).toHaveLength(1);

    // Test table name mapping
    expect(metricsCollector.getTableNameForMetricType('time_to_first_review')).toBe('pr_first_review');
    expect(metricsCollector.getTableNameForMetricType('time_to_merge')).toBe('pr_merge');
  });
});
