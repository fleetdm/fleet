/**
 * Tests for markdown parser module
 */

import { jest } from '@jest/globals';

// Mock the logger
const mockLogger = {
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  debug: jest.fn()
};

// Mock fs
const mockFs = {
  existsSync: jest.fn(),
  readFileSync: jest.fn()
};

// Mock path
const mockPath = {
  resolve: jest.fn()
};

// Set up module mocks
jest.unstable_mockModule('../src/logger.js', () => ({
  default: mockLogger,
  ...mockLogger
}));

jest.unstable_mockModule('fs', () => ({
  default: mockFs,
  ...mockFs
}));

jest.unstable_mockModule('path', () => ({
  default: mockPath,
  ...mockPath
}));

// Import the module after mocking
const { parseProductGroups, validateMarkdownStructure } = await import('../src/markdown-parser.js');

describe('MarkdownParser', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPath.resolve.mockImplementation((cwd, filePath) => `/resolved/${filePath}`);
  });

  describe('parseProductGroups', () => {
    it('should parse valid markdown with all groups correctly', () => {
      const mockMarkdown = `
# Product Groups

## Some other content

### MDM group

| Role | Contributor |
|------|-------------|
| Product Manager | _([@testpm1](https://github.com/testpm1))_ |
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Developer | _([@testdev1](https://github.com/testdev1))_, _([@testdev2](https://github.com/testdev2))_, _([@testdev3](https://github.com/testdev3))_ |
| Quality Assurance | _([@testqa1](https://github.com/testqa1))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Product Manager | _([@testpm2](https://github.com/testpm2))_ |
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_, _([@orchdev2](https://github.com/orchdev2))_, _([@orchdev3](https://github.com/orchdev3))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_, _([@softdev2](https://github.com/softdev2))_ |
| Quality Assurance | _([@testqa2](https://github.com/testqa2))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_, _([@secdev2](https://github.com/secdev2))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(mockFs.existsSync).toHaveBeenCalledWith('/resolved/test-file.md');
      expect(mockFs.readFileSync).toHaveBeenCalledWith('/resolved/test-file.md', 'utf8');

      // Should extract usernames and create dual group membership
      expect(result).toEqual([
        // MDM group users (tech lead + developers)
        { group: 'mdm', username: 'techlead1' },
        { group: 'engineering', username: 'techlead1' },
        { group: 'mdm', username: 'testdev1' },
        { group: 'engineering', username: 'testdev1' },
        { group: 'mdm', username: 'testdev2' },
        { group: 'engineering', username: 'testdev2' },
        { group: 'mdm', username: 'testdev3' },
        { group: 'engineering', username: 'testdev3' },
        // Orchestration group users (tech lead + developers)
        { group: 'orchestration', username: 'techlead2' },
        { group: 'engineering', username: 'techlead2' },
        { group: 'orchestration', username: 'orchdev1' },
        { group: 'engineering', username: 'orchdev1' },
        { group: 'orchestration', username: 'orchdev2' },
        { group: 'engineering', username: 'orchdev2' },
        { group: 'orchestration', username: 'orchdev3' },
        { group: 'engineering', username: 'orchdev3' },
        // Software group users (tech lead + developers)
        { group: 'software', username: 'techlead3' },
        { group: 'engineering', username: 'techlead3' },
        { group: 'software', username: 'softdev1' },
        { group: 'engineering', username: 'softdev1' },
        { group: 'software', username: 'softdev2' },
        { group: 'engineering', username: 'softdev2' },
        // Security & compliance group users (tech lead + developers)
        { group: 'security-compliance', username: 'techlead4' },
        { group: 'engineering', username: 'techlead4' },
        { group: 'security-compliance', username: 'secdev1' },
        { group: 'engineering', username: 'secdev1' },
        { group: 'security-compliance', username: 'secdev2' },
        { group: 'engineering', username: 'secdev2' }
      ]);

      expect(mockLogger.info).toHaveBeenCalledWith('Parsing product groups from /resolved/test-file.md');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 1 tech lead(s) in mdm group: techlead1');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 3 developer(s) in mdm group: testdev1, testdev2, testdev3');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 1 tech lead(s) in orchestration group: techlead2');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 3 developer(s) in orchestration group: orchdev1, orchdev2, orchdev3');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 1 tech lead(s) in software group: techlead3');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 2 developer(s) in software group: softdev1, softdev2');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 1 tech lead(s) in security-compliance group: techlead4');
      expect(mockLogger.info).toHaveBeenCalledWith('Found 2 developer(s) in security-compliance group: secdev1, secdev2');
      expect(mockLogger.info).toHaveBeenCalledWith('Extracted 28 user-group mappings from markdown');
    });

    it('should throw error when sections are missing', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Developer | _([@testdev1](https://github.com/testdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Section not found: Orchestration group');
    });

    it('should throw error when Tech Lead row is missing', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Product Manager | _([@testpm1](https://github.com/testpm1))_ |
| Developer | _([@testdev1](https://github.com/testdev1))_ |
| Quality Assurance | _([@testqa1](https://github.com/testqa1))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Error extracting usernames from MDM group', {}, expect.any(Error));
    });

    it('should throw error when Developer row is missing', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Product Manager | _([@testpm1](https://github.com/testpm1))_ |
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Quality Assurance | _([@testqa1](https://github.com/testqa1))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Error extracting usernames from MDM group', {}, expect.any(Error));
    });

    it('should throw error when no valid developer usernames found', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Developer | Some text without proper format, _([@testvaliduser](https://github.com/testvaliduser))_, invalid format here |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | No valid usernames here at all |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Error extracting usernames from Orchestration group', {}, expect.any(Error));
    });

    it('should throw error when no Tech Lead usernames found', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Tech Lead | No valid usernames here |
| Developer | _([@testdev1](https://github.com/testdev1))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Error extracting usernames from MDM group', {}, expect.any(Error));
    });

    it('should handle multi-line developer cells', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead1](https://github.com/techlead1))_ |
| Developer | _([@testuser1](https://github.com/testuser1))_,
_([@testuser2](https://github.com/testuser2))_,
_([@testuser3](https://github.com/testuser3))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toContainEqual({ group: 'mdm', username: 'techlead1' });
      expect(result).toContainEqual({ group: 'engineering', username: 'techlead1' });
      expect(result).toContainEqual({ group: 'mdm', username: 'testuser1' });
      expect(result).toContainEqual({ group: 'engineering', username: 'testuser1' });
      expect(result).toContainEqual({ group: 'mdm', username: 'testuser2' });
      expect(result).toContainEqual({ group: 'engineering', username: 'testuser2' });
      expect(result).toContainEqual({ group: 'mdm', username: 'testuser3' });
      expect(result).toContainEqual({ group: 'engineering', username: 'testuser3' });
    });

    it('should handle file not found', () => {
      mockFs.existsSync.mockReturnValue(false);

      const result = parseProductGroups('nonexistent-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith('Product groups file not found at /resolved/nonexistent-file.md');
    });

    it('should handle file read errors', () => {
      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockImplementation(() => {
        throw new Error('Permission denied');
      });

      const result = parseProductGroups('error-file.md');

      expect(result).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalledWith(
        'Error parsing product groups file: error-file.md',
        {},
        expect.any(Error)
      );
    });

    it('should handle usernames with hyphens and numbers', () => {
      const mockMarkdown = `
# Product Groups

### MDM group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead-123](https://github.com/techlead-123))_ |
| Developer | _([@testuser-123](https://github.com/testuser-123))_, _([@fakeuser-456](https://github.com/fakeuser-456))_ |

### Orchestration group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead2](https://github.com/techlead2))_ |
| Developer | _([@orchdev1](https://github.com/orchdev1))_ |

### Software group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead3](https://github.com/techlead3))_ |
| Developer | _([@softdev1](https://github.com/softdev1))_ |

### Security & compliance group

| Role | Contributor |
|------|-------------|
| Tech Lead | _([@techlead4](https://github.com/techlead4))_ |
| Developer | _([@secdev1](https://github.com/secdev1))_ |
`;

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(mockMarkdown);

      const result = parseProductGroups('test-file.md');

      expect(result).toContainEqual({ group: 'mdm', username: 'techlead-123' });
      expect(result).toContainEqual({ group: 'engineering', username: 'techlead-123' });
      expect(result).toContainEqual({ group: 'mdm', username: 'testuser-123' });
      expect(result).toContainEqual({ group: 'engineering', username: 'testuser-123' });
      expect(result).toContainEqual({ group: 'mdm', username: 'fakeuser-456' });
      expect(result).toContainEqual({ group: 'engineering', username: 'fakeuser-456' });
    });
  });

  describe('validateMarkdownStructure', () => {
    it('should return true for valid markdown with all required sections', () => {
      const validMarkdown = `
# Product Groups

### MDM group
Some content

### Orchestration group
Some content

### Software group
Some content

### Security & compliance group
Some content
`;

      const result = validateMarkdownStructure(validMarkdown);
      expect(result).toBe(true);
    });

    it('should return false and warn for missing sections', () => {
      const invalidMarkdown = `
# Product Groups

### MDM group
Some content

### Software group
Some content
`;

      const result = validateMarkdownStructure(invalidMarkdown);
      expect(result).toBe(false);
      expect(mockLogger.warn).toHaveBeenCalledWith('Missing required section: Orchestration group');
    });

    it('should return false for completely empty content', () => {
      const result = validateMarkdownStructure('');
      expect(result).toBe(false);
      expect(mockLogger.warn).toHaveBeenCalledWith('Missing required section: MDM group');
      // The function returns false on first missing section, so other warnings may not be called
      expect(mockLogger.warn).toHaveBeenCalledTimes(1);
    });

    it('should handle case-sensitive section matching', () => {
      const invalidMarkdown = `
# Product Groups

### mdm group
Some content

### orchestration group
Some content

### software group
Some content

### security & compliance group
Some content
`;

      const result = validateMarkdownStructure(invalidMarkdown);
      expect(result).toBe(false);
      expect(mockLogger.warn).toHaveBeenCalledWith('Missing required section: MDM group');
      // The function returns false on first missing section, so other warnings may not be called
      expect(mockLogger.warn).toHaveBeenCalledTimes(1);
    });
  });
});