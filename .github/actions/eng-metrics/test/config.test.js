/**
 * Tests for configuration module
 */

import { jest } from "@jest/globals";

// Mock the logger
const mockLogger = {
  default: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn(),
};

// Mock fs
const mockFs = {
  default: {
    existsSync: jest.fn(),
    readFileSync: jest.fn(),
  },
  existsSync: jest.fn(),
  readFileSync: jest.fn(),
};

// Mock dotenv
const mockDotenv = {
  default: {
    config: jest.fn(),
  },
  config: jest.fn(),
};

// Mock modules before importing
jest.unstable_mockModule("../src/logger.js", () => mockLogger);
jest.unstable_mockModule("fs", () => mockFs);
jest.unstable_mockModule("dotenv", () => mockDotenv);

// Now import the module under test
const { loadConfig, validateConfig } = await import("../src/config.js");

describe("Config", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Clear environment variables. This is the recommended approach to prevent performance hit.
    Reflect.deleteProperty(process.env, "GITHUB_TOKEN");
    Reflect.deleteProperty(process.env, "SERVICE_ACCOUNT_KEY_PATH");
    Reflect.deleteProperty(process.env, "BIGQUERY_DATASET_ID");
    Reflect.deleteProperty(process.env, "REPOSITORIES");
    Reflect.deleteProperty(process.env, "LOOKBACK_DAYS");
    Reflect.deleteProperty(process.env, "TARGET_BRANCH");
    Reflect.deleteProperty(process.env, "PRINT_ONLY");
    Reflect.deleteProperty(process.env, "ENABLED_METRICS");
    Reflect.deleteProperty(process.env, "TIME_TO_FIRST_REVIEW_TABLE");
    Reflect.deleteProperty(process.env, "TIME_TO_MERGE_TABLE");

    // Reset all mocks
    jest.clearAllMocks();

    // Mock fs.existsSync to return true for config.json
    mockFs.existsSync.mockReturnValue(true);

    // Mock fs.readFileSync to return a default config
    mockFs.readFileSync.mockReturnValue(
      JSON.stringify({
        repositories: ["owner/repo1", "owner/repo2"],
        targetBranch: "main",
        bigQueryDatasetId: "test_dataset",
        lookbackDays: 5,
        serviceAccountKeyPath: "./service-account-key.json",
        printOnly: false,
      })
    );
  });

  afterEach(() => {
    // Restore original environment
    process.env = originalEnv;
  });

  describe("loadConfig", () => {
    test("should load default configuration", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = "owner/repo1,owner/repo2";

      const config = loadConfig();

      expect(config).toEqual({
        githubToken: "test-token",
        serviceAccountKeyPath: "/path/to/key.json",
        bigQueryDatasetId: "test_dataset",
        repositories: ["owner/repo1", "owner/repo2"],
        lookbackDays: 5,
        targetBranch: "main",
        printOnly: false,
        userGroupEnabled: true,
        userGroupFilepath: "../../../handbook/company/product-groups.md",
        excludeBotReviews: true,
        metrics: {
          timeToFirstReview: {
            enabled: true,
            tableName: "pr_first_review",
          },
          timeToMerge: {
            enabled: true,
            tableName: "pr_merge",
          },
        },
      });
    });

    test("should override defaults with environment variables", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = "owner/repo";
      process.env.LOOKBACK_DAYS = "14";
      process.env.TARGET_BRANCH = "develop";
      process.env.PRINT_ONLY = "true";
      process.env.ENABLED_METRICS = "time_to_first_review";
      process.env.TIME_TO_FIRST_REVIEW_TABLE = "custom_first_review";
      process.env.TIME_TO_MERGE_TABLE = "custom_pr_merge";

      const config = loadConfig();

      expect(config).toEqual({
        githubToken: "test-token",
        serviceAccountKeyPath: "/path/to/key.json",
        bigQueryDatasetId: "test_dataset",
        repositories: ["owner/repo"],
        lookbackDays: 5,
        targetBranch: "develop",
        printOnly: true,
        userGroupEnabled: true,
        userGroupFilepath: "../../../handbook/company/product-groups.md",
        excludeBotReviews: true,
        metrics: {
          timeToFirstReview: {
            enabled: true,
            tableName: "custom_first_review",
          },
          timeToMerge: {
            enabled: false,
            tableName: "custom_pr_merge",
          },
        },
      });
    });

    test("should handle multiple enabled metrics", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = "owner/repo";
      process.env.ENABLED_METRICS = "time_to_first_review,time_to_merge";

      const config = loadConfig();

      expect(config.metrics).toEqual({
        timeToFirstReview: {
          enabled: true,
          tableName: "pr_first_review",
        },
        timeToMerge: {
          enabled: true,
          tableName: "pr_merge",
        },
      });
    });

    test("should handle only time_to_merge enabled", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = "owner/repo";
      process.env.ENABLED_METRICS = "time_to_merge";

      const config = loadConfig();

      expect(config.metrics).toEqual({
        timeToFirstReview: {
          enabled: false,
          tableName: "pr_first_review",
        },
        timeToMerge: {
          enabled: true,
          tableName: "pr_merge",
        },
      });
    });

    test("should trim whitespace from repositories", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = " owner/repo1 , owner/repo2 , owner/repo3 ";

      const config = loadConfig();

      expect(config.repositories).toEqual([
        "owner/repo1",
        "owner/repo2",
        "owner/repo3",
      ]);
    });

    test("should trim whitespace from enabled metrics", () => {
      process.env.GITHUB_TOKEN = "test-token";
      process.env.SERVICE_ACCOUNT_KEY_PATH = "/path/to/key.json";
      process.env.BIGQUERY_DATASET_ID = "test_dataset";
      process.env.REPOSITORIES = "owner/repo";
      process.env.ENABLED_METRICS = " time_to_first_review , time_to_merge ";

      const config = loadConfig();

      expect(config.metrics).toEqual({
        timeToFirstReview: {
          enabled: true,
          tableName: "pr_first_review",
        },
        timeToMerge: {
          enabled: true,
          tableName: "pr_merge",
        },
      });
    });
  });

  describe("validateConfig", () => {
    const baseValidConfig = {
      githubToken: "test-token",
      serviceAccountKeyPath: "/path/to/key.json",
      bigQueryDatasetId: "test_dataset",
      repositories: ["owner/repo"],
      lookbackDays: 30,
      targetBranch: "main",
      printOnly: false,
      metrics: {
        timeToFirstReview: {
          enabled: true,
          tableName: "pr_first_review",
        },
        timeToMerge: {
          enabled: true,
          tableName: "pr_merge",
        },
      },
    };

    test("should validate correct configuration", () => {
      expect(validateConfig(baseValidConfig)).toBe(true);
    });

    test("should return false for missing GitHub token", () => {
      const config = { ...baseValidConfig, githubToken: "" };
      expect(validateConfig(config)).toBe(false);
    });

    test("should return false for missing service account key path in non-print mode", () => {
      const config = { ...baseValidConfig, serviceAccountKeyPath: "" };
      expect(validateConfig(config)).toBe(false);
    });

    test("should not require service account key path in print-only mode", () => {
      const config = {
        ...baseValidConfig,
        serviceAccountKeyPath: "",
        printOnly: true,
      };
      expect(validateConfig(config)).toBe(true);
    });

    test("should not require BigQuery dataset ID in print-only mode", () => {
      const config = {
        ...baseValidConfig,
        bigQueryDatasetId: "",
        printOnly: true,
      };
      expect(validateConfig(config)).toBe(true);
    });

    test("should return false for empty repositories array", () => {
      const config = { ...baseValidConfig, repositories: [] };
      expect(validateConfig(config)).toBe(false);
    });

    test("should return false for invalid repository format", () => {
      const config = { ...baseValidConfig, repositories: ["invalid-repo"] };
      expect(validateConfig(config)).toBe(false);
    });

    test("should validate lookback days correctly", () => {
      const config = { ...baseValidConfig, lookbackDays: 5 };
      expect(validateConfig(config)).toBe(true);
    });

    test("should return false when no metrics are enabled", () => {
      const config = {
        ...baseValidConfig,
        metrics: {
          timeToFirstReview: { enabled: false, tableName: "pr_first_review" },
          timeToMerge: { enabled: false, tableName: "pr_merge" },
        },
      };
      expect(validateConfig(config)).toBe(false);
    });

    test("should return false for missing table name when metric is enabled", () => {
      const config = {
        ...baseValidConfig,
        metrics: {
          timeToFirstReview: { enabled: true, tableName: "" },
          timeToMerge: { enabled: false, tableName: "pr_merge" },
        },
      };
      expect(validateConfig(config)).toBe(false);
    });

    test("should allow missing table name when metric is disabled", () => {
      const config = {
        ...baseValidConfig,
        metrics: {
          timeToFirstReview: { enabled: false, tableName: "" },
          timeToMerge: { enabled: true, tableName: "pr_merge" },
        },
      };
      expect(validateConfig(config)).toBe(true);
    });

    test("should validate multiple valid repositories", () => {
      const config = {
        ...baseValidConfig,
        repositories: ["owner1/repo1", "owner2/repo2", "owner3/repo3"],
      };
      expect(validateConfig(config)).toBe(true);
    });

    test("should throw error for mixed valid and invalid repositories", () => {
      const config = {
        ...baseValidConfig,
        repositories: ["owner1/repo1", "invalid-repo", "owner3/repo3"],
      };
      expect(validateConfig(config)).toBe(false);
    });
  });
});
