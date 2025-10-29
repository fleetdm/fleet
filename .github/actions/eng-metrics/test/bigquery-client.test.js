/**
 * Tests for BigQuery client module
 */

import { jest } from '@jest/globals';
import { BigQueryClient } from '../src/bigquery-client.js';

// Mock the logger
jest.mock('../src/logger.js', () => ({
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn()
}));

// Mock fs
jest.mock('fs', () => ({
  existsSync: jest.fn(() => true),
  readFileSync: jest.fn(() => JSON.stringify({
    project_id: 'test-project-id',
    type: 'service_account',
    private_key_id: 'key-id',
    private_key: '-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----\n',
    client_email: 'test@test-project-id.iam.gserviceaccount.com',
    client_id: '123456789',
    auth_uri: 'https://accounts.google.com/o/oauth2/auth',
    token_uri: 'https://oauth2.googleapis.com/token'
  }))
}));

// Mock @google-cloud/bigquery
jest.mock('@google-cloud/bigquery', () => ({
  BigQuery: jest.fn(() => ({
    dataset: jest.fn(),
    query: jest.fn()
  }))
}));

describe('BigQueryClient', () => {
  let bigqueryClient;
  let mockBigQuery;
  let mockDataset;
  let mockTable;

  beforeEach(() => {
    mockTable = {
      exists: jest.fn(() => [true]),
      create: jest.fn(),
      insert: jest.fn(() => [{}])
    };

    mockDataset = {
      exists: jest.fn(() => [true]),
      create: jest.fn(),
      table: jest.fn(() => mockTable)
    };

    mockBigQuery = {
      dataset: jest.fn(() => mockDataset),
      query: jest.fn(() => [[]])
    };

    // Create client without calling constructor to avoid file check
    bigqueryClient = Object.create(BigQueryClient.prototype);
    bigqueryClient.bigquery = mockBigQuery;
    bigqueryClient.projectId = 'test-project-id';
  });


  describe('getProjectId', () => {
    test('should return the extracted project ID', () => {
      expect(bigqueryClient.getProjectId()).toBe('test-project-id');
    });
  });

  describe('getSchemaForMetricType', () => {
    test('should return first_review table schema', () => {
      const schema = bigqueryClient.getSchemaForMetricType('time_to_first_review');
      expect(schema.fields).toBeTruthy();
    });

    test('should return pr_merge table schema', () => {
      const schema = bigqueryClient.getSchemaForMetricType('time_to_merge');
      expect(schema.fields).toBeTruthy();
    });

    test('should throw error for unknown metric type', () => {
      expect(() => {
        bigqueryClient.getSchemaForMetricType('unknown_metric');
      }).toThrow('Unknown metric type: unknown_metric');
    });
  });

  describe('getConfigurationForMetricType', () => {
    test('should return first_review table configuration', () => {
      const config = bigqueryClient.getConfigurationForMetricType('time_to_first_review');
      expect(config).toBeTruthy();
    });

    test('should return pr_merge table configuration', () => {
      const config = bigqueryClient.getConfigurationForMetricType('time_to_merge');
      expect(config).toBeTruthy();
    });

    test('should throw error for unknown metric type configuration', () => {
      expect(() => {
        bigqueryClient.getConfigurationForMetricType('unknown_metric');
      }).toThrow('Unknown metric type for table configuration: unknown_metric');
    });
  });

  describe('transformMetricsToRow', () => {
    test('should transform time_to_first_review metrics', () => {
      const metrics = {
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
      };

      const row = bigqueryClient.transformMetricsToRow(metrics);

      expect(row).toEqual({
        review_date: '2023-06-15',
        pr_creator: 'testuser',
        pr_url: 'https://github.com/owner/repo/pull/123',
        pickup_time_seconds: 7200,
        repository: 'owner/repo',
        pr_number: 123,
        target_branch: 'main',
        ready_time: '2023-06-15T10:00:00.000Z',
        first_review_time: '2023-06-15T12:00:00.000Z'
      });
    });

    test('should transform time_to_merge metrics', () => {
      const metrics = {
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
      };

      const row = bigqueryClient.transformMetricsToRow(metrics);

      expect(row).toEqual({
        merge_date: '2023-06-15',
        pr_creator: 'testuser',
        pr_url: 'https://github.com/owner/repo/pull/123',
        merge_time_seconds: 16200,
        repository: 'owner/repo',
        pr_number: 123,
        target_branch: 'main',
        ready_time: '2023-06-15T10:00:00.000Z',
        merge_time: '2023-06-15T14:30:00.000Z'
      });
    });

    test('should throw error for unknown metric type', () => {
      const metrics = {
        metricType: 'unknown_type'
      };

      expect(() => {
        bigqueryClient.transformMetricsToRow(metrics);
      }).toThrow('Unknown metric type: unknown_type');
    });
  });

  describe('createTableIfNotExists', () => {
    test('should create table with correct configuration for first_review', async () => {
      mockTable.exists.mockResolvedValue([false]);
      const schema = { fields: [] };

      await bigqueryClient.createTableIfNotExists('test_dataset', 'first_review', schema, 'time_to_first_review');

      expect(mockTable.create).toHaveBeenCalled();
    });

    test('should create table with correct configuration for pr_merge', async () => {
      mockTable.exists.mockResolvedValue([false]);
      const schema = { fields: [] };

      await bigqueryClient.createTableIfNotExists('test_dataset', 'pr_merge', schema, 'time_to_merge');

      expect(mockTable.create).toHaveBeenCalled();
    });

    test('should not create table if it already exists', async () => {
      mockTable.exists.mockResolvedValue([true]);
      const schema = { fields: [] };

      await bigqueryClient.createTableIfNotExists('test_dataset', 'first_review', schema, 'time_to_first_review');

      expect(mockTable.create).not.toHaveBeenCalled();
    });
  });

  describe('uploadMetrics', () => {
    test('should upload metrics with correct schema', async () => {
      const metrics = [
        {
          metricType: 'time_to_first_review',
          prNumber: 123,
          reviewDate: '2023-06-15',
          prCreator: 'testuser',
          prUrl: 'https://github.com/owner/repo/pull/123',
          pickupTimeSeconds: 7200,
          repository: 'owner/repo',
          targetBranch: 'main',
          readyTime: new Date('2023-06-15T10:00:00Z'),
          firstReviewTime: new Date('2023-06-15T12:00:00Z')
        }
      ];

      await bigqueryClient.uploadMetrics('test_dataset', 'pr_first_review', metrics);

      expect(mockTable.insert).toHaveBeenCalledWith([
        {
          review_date: '2023-06-15',
          pr_creator: 'testuser',
          pr_url: 'https://github.com/owner/repo/pull/123',
          pickup_time_seconds: 7200,
          repository: 'owner/repo',
          pr_number: 123,
          target_branch: 'main',
          ready_time: '2023-06-15T10:00:00.000Z',
          first_review_time: '2023-06-15T12:00:00.000Z'
        }
      ]);
    });

    test('should handle empty metrics array', async () => {
      await bigqueryClient.uploadMetrics('test_dataset', 'pr_first_review', []);

      expect(mockTable.insert).not.toHaveBeenCalled();
    });

    test('should filter out existing metrics', async () => {
      const metrics = [
        {
          metricType: 'time_to_first_review',
          prNumber: 123,
          reviewDate: '2023-06-15',
          prCreator: 'testuser',
          prUrl: 'https://github.com/owner/repo/pull/123',
          pickupTimeSeconds: 7200,
          repository: 'owner/repo',
          targetBranch: 'main',
          readyTime: new Date('2023-06-15T10:00:00Z'),
          firstReviewTime: new Date('2023-06-15T12:00:00Z')
        }
      ];

      // Mock existing metrics check to return that PR 123 already exists
      mockBigQuery.query.mockResolvedValue([[{ pr_number: 123 }]]);

      await bigqueryClient.uploadMetrics('test_dataset', 'pr_first_review', metrics);

      expect(mockTable.insert).not.toHaveBeenCalled();
    });
  });
});
