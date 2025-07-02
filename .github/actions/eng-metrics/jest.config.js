/**
 * Jest configuration for engineering metrics tests
 */

export default {
  // Test environment
  testEnvironment: 'node',
  
  // Transform configuration for ES modules
  transform: {},
  
  // File extensions to consider
  moduleFileExtensions: ['js', 'json'],
  
  // Test file patterns
  testMatch: [
    '**/test/**/*.test.js'
  ],
  
  // Coverage configuration
  collectCoverage: false, // Disable for now to focus on functionality
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
  
  // Files to collect coverage from
  collectCoverageFrom: [
    'src/**/*.js',
    '!src/index.js', // Exclude main entry point
    '!src/logger.js' // Exclude logger (simple utility)
  ],
  
  // Setup files
  setupFilesAfterEnv: ['<rootDir>/test/setup.js'],
  
  // Clear mocks between tests
  clearMocks: true,
  
  // Restore mocks after each test
  restoreMocks: true,
  
  // Verbose output
  verbose: true,
  
  // Global configuration for ES modules
  globals: {
    'ts-jest': {
      useESM: true
    }
  }
};