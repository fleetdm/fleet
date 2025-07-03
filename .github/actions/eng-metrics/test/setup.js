/**
 * Jest setup file for engineering metrics tests
 */
import { jest } from "@jest/globals";

// Set test environment variables
process.env.NODE_ENV = "test";

// Mock Date.now for consistent testing
beforeAll(() => {
  jest.useFakeTimers().setSystemTime(new Date("2023-06-15T12:00:00Z"));
});

afterAll(() => {
  jest.useRealTimers();
});
