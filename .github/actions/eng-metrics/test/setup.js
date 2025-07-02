/**
 * Jest setup file for engineering metrics tests
 */

// Set test environment variables
process.env.NODE_ENV = 'test';

// Mock Date.now for consistent testing
const mockDate = new Date('2023-06-15T12:00:00Z');
const originalDateNow = Date.now;

beforeAll(() => {
  Date.now = () => mockDate.getTime();
});

afterAll(() => {
  Date.now = originalDateNow;
});