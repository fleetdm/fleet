/**
 * Tests for metrics collector module
 */

import { jest } from '@jest/globals';
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

describe('MetricsCollector', () => {
  let metricsCollector;
  let mockConfig;
  let mockGitHubClient;
  let mockBigQueryClient;

  beforeEach(() => {
    mockConfig = {
      githubToken: 'fake-token',
      serviceAccountKeyPath: '/fake/path',
      repositories: ['owner/repo'],
      lookbackDays: 7,
      targetBranch: 'main',
      bigQueryDatasetId: 'test_dataset',
      printOnly: false,
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

    metricsCollector = new MetricsCollector(mockConfig);

    // Mock the clients
    mockGitHubClient = {
      fetchPullRequests: jest.fn(),
      fetchPRTimelineEvents: jest.fn(),
      fetchPRReviewEvents: jest.fn(),
      calculatePickupTime: jest.fn(),
      calculateTimeToMerge: jest.fn()
    };

    mockBigQueryClient = {
      uploadMetrics: jest.fn()
    };

    metricsCollector.githubClient = mockGitHubClient;
    metricsCollector.bigqueryClient = mockBigQueryClient;
  });

  describe('collectPRMetrics', () => {
    const mockPR = {
      number: 123,
      html_url: 'https://github.com/owner/repo/pull/123',
      user: { login: 'testuser' },
      base: { ref: 'main' },
      head: { repo: { full_name: 'owner/repo' } }
    };

    const mockTimelineEvents = [
      { event: 'ready_for_review', created_at: '2023-06-15T10:00:00Z' }
    ];

    const mockReviewEvents = [
      { submitted_at: '2023-06-15T12:00:00Z', state: 'approved', user: { login: 'reviewer1' } }
    ];

    test('should collect both metrics when both are enabled', async () => {
      const firstReviewMetric = {
        metricType: 'time_to_first_review',
        prNumber: 123,
        pickupTimeSeconds: 7200
      };

      const mergeMetric = {
        metricType: 'time_to_merge',
        prNumber: 123,
        mergeTimeSeconds: 16200
      };

      mockGitHubClient.calculatePickupTime.mockReturnValue(firstReviewMetric);
      mockGitHubClient.calculateTimeToMerge.mockReturnValue(mergeMetric);

      const result = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

      expect(result).toEqual([firstReviewMetric, mergeMetric]);
      expect(mockGitHubClient.calculatePickupTime).toHaveBeenCalledWith(mockPR, mockTimelineEvents, mockReviewEvents);
      expect(mockGitHubClient.calculateTimeToMerge).toHaveBeenCalledWith(mockPR, mockTimelineEvents, mockReviewEvents);
    });

    test('should only collect first review metric when merge is disabled', async () => {
      metricsCollector.config.metrics.timeToMerge.enabled = false;

      const firstReviewMetric = {
        metricType: 'time_to_first_review',
        prNumber: 123,
        pickupTimeSeconds: 7200
      };

      mockGitHubClient.calculatePickupTime.mockReturnValue(firstReviewMetric);

      const result = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

      expect(result).toEqual([firstReviewMetric]);
      expect(mockGitHubClient.calculatePickupTime).toHaveBeenCalled();
      expect(mockGitHubClient.calculateTimeToMerge).not.toHaveBeenCalled();
    });

    test('should only collect merge metric when first review is disabled', async () => {
      metricsCollector.config.metrics.timeToFirstReview.enabled = false;

      const mergeMetric = {
        metricType: 'time_to_merge',
        prNumber: 123,
        mergeTimeSeconds: 16200
      };

      mockGitHubClient.calculateTimeToMerge.mockReturnValue(mergeMetric);

      const result = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

      expect(result).toEqual([mergeMetric]);
      expect(mockGitHubClient.calculatePickupTime).not.toHaveBeenCalled();
      expect(mockGitHubClient.calculateTimeToMerge).toHaveBeenCalled();
    });

    test('should handle null metrics gracefully', async () => {
      mockGitHubClient.calculatePickupTime.mockReturnValue(null);
      mockGitHubClient.calculateTimeToMerge.mockReturnValue(null);

      const result = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

      expect(result).toEqual([]);
    });

    test('should handle errors in metric calculation', async () => {
      mockGitHubClient.calculatePickupTime.mockImplementation(() => {
        throw new Error('Calculation error');
      });

      const mergeMetric = {
        metricType: 'time_to_merge',
        prNumber: 123,
        mergeTimeSeconds: 16200
      };

      mockGitHubClient.calculateTimeToMerge.mockReturnValue(mergeMetric);

      const result = await metricsCollector.collectPRMetrics(mockPR, mockTimelineEvents, mockReviewEvents);

      expect(result).toEqual([mergeMetric]);
    });
  });

  describe('groupMetricsByType', () => {
    test('should group metrics by type correctly', () => {
      const metrics = [
        { metricType: 'time_to_first_review', prNumber: 123 },
        { metricType: 'time_to_merge', prNumber: 123 },
        { metricType: 'time_to_first_review', prNumber: 124 },
        { metricType: 'time_to_merge', prNumber: 124 }
      ];

      const grouped = metricsCollector.groupMetricsByType(metrics);

      expect(grouped).toEqual({
        time_to_first_review: [
          { metricType: 'time_to_first_review', prNumber: 123 },
          { metricType: 'time_to_first_review', prNumber: 124 }
        ],
        time_to_merge: [
          { metricType: 'time_to_merge', prNumber: 123 },
          { metricType: 'time_to_merge', prNumber: 124 }
        ]
      });
    });

    test('should handle empty metrics array', () => {
      const grouped = metricsCollector.groupMetricsByType([]);
      expect(grouped).toEqual({});
    });
  });

  describe('getTableNameForMetricType', () => {
    test('should return correct table name for time_to_first_review', () => {
      const tableName = metricsCollector.getTableNameForMetricType('time_to_first_review');
      expect(tableName).toBe('pr_first_review');
    });

    test('should return correct table name for time_to_merge', () => {
      const tableName = metricsCollector.getTableNameForMetricType('time_to_merge');
      expect(tableName).toBe('pr_merge');
    });

    test('should throw error for unknown metric type', () => {
      expect(() => {
        metricsCollector.getTableNameForMetricType('unknown_type');
      }).toThrow('Unknown metric type: unknown_type');
    });
  });

  describe('uploadMetrics', () => {
    test('should upload metrics to correct tables', async () => {
      const metrics = [
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

      await metricsCollector.uploadMetrics(metrics);

      expect(mockBigQueryClient.uploadMetrics).toHaveBeenCalledTimes(2);

      // Check first call (time_to_first_review)
      expect(mockBigQueryClient.uploadMetrics).toHaveBeenNthCalledWith(
        1,
        'test_dataset',
        'pr_first_review',
        [
          { metricType: 'time_to_first_review', prNumber: 123, pickupTimeSeconds: 7200 },
          { metricType: 'time_to_first_review', prNumber: 124, pickupTimeSeconds: 3600 }
        ]
      );

      // Check second call (time_to_merge)
      expect(mockBigQueryClient.uploadMetrics).toHaveBeenNthCalledWith(
        2,
        'test_dataset',
        'pr_merge',
        [
          { metricType: 'time_to_merge', prNumber: 123, mergeTimeSeconds: 16200 }
        ]
      );
    });

    test('should handle empty metrics array', async () => {
      await metricsCollector.uploadMetrics([]);

      expect(mockBigQueryClient.uploadMetrics).not.toHaveBeenCalled();
    });

    test('should skip empty metric groups', async () => {
      const metrics = [
        {
          metricType: 'time_to_first_review',
          prNumber: 123,
          pickupTimeSeconds: 7200
        }
      ];

      await metricsCollector.uploadMetrics(metrics);

      expect(mockBigQueryClient.uploadMetrics).toHaveBeenCalledTimes(1);
      expect(mockBigQueryClient.uploadMetrics).toHaveBeenCalledWith(
        'test_dataset',
        'pr_first_review',
        [{ metricType: 'time_to_first_review', prNumber: 123, pickupTimeSeconds: 7200 }]
      );
    });
  });

  describe('getMetricTypeDisplayName', () => {
    test('should return correct display names', () => {
      expect(metricsCollector.getMetricTypeDisplayName('time_to_first_review')).toBe('Time to First Review');
      expect(metricsCollector.getMetricTypeDisplayName('time_to_merge')).toBe('Time to Merge');
      expect(metricsCollector.getMetricTypeDisplayName('unknown_type')).toBe('unknown_type');
    });
  });

  describe('getTimeFieldForMetricType', () => {
    test('should return correct time fields', () => {
      const firstReviewMetric = { pickupTimeSeconds: 7200, mergeTimeSeconds: 16200 };
      const mergeMetric = { pickupTimeSeconds: 7200, mergeTimeSeconds: 16200 };

      expect(metricsCollector.getTimeFieldForMetricType('time_to_first_review', firstReviewMetric)).toBe(7200);
      expect(metricsCollector.getTimeFieldForMetricType('time_to_merge', mergeMetric)).toBe(16200);
      expect(metricsCollector.getTimeFieldForMetricType('unknown_type', {})).toBe(0);
    });
  });
});
