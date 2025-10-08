/**
 * Markdown parser for extracting GitHub usernames from product groups
 * Parses the product-groups.md file to extract developer usernames by group
 */

import fs from 'fs';
import path from 'path';
import logger from './logger.js';

/**
 * Parses the product groups markdown file and extracts GitHub usernames
 * @param {string} filePath - Path to the product-groups.md file
 * @returns {Array<{group: string, username: string}>} Array of user group mappings
 */
export const parseProductGroups = (filePath) => {
  try {
    const resolvedPath = path.resolve(process.cwd(), filePath);
    logger.info(`Parsing product groups from ${resolvedPath}`);

    if (!fs.existsSync(resolvedPath)) {
      logger.error(`Product groups file not found at ${resolvedPath}`);
      return [];
    }

    const content = fs.readFileSync(resolvedPath, 'utf8');
    return extractUsernamesFromMarkdown(content);
  } catch (err) {
    logger.error(`Error parsing product groups file: ${filePath}`, {}, err);
    return [];
  }
};

/**
 * Extracts usernames from markdown content
 * @param {string} content - Markdown content
 * @returns {Array<{group: string, username: string}>} Array of user group mappings
 * @throws {Error} If required sections are missing or validation fails
 */
const extractUsernamesFromMarkdown = (content) => {
  const userGroups = [];

  // Define the groups we're looking for and their corresponding database group names
  const groupMappings = {
    'MDM group': 'mdm',
    'Orchestration group': 'orchestration',
    'Software group': 'software',
    'Security & compliance group': 'security-compliance',
  };

  // For each group, find its section and extract usernames
  for (const [sectionName, groupName] of Object.entries(groupMappings)) {
    // Find the section for this group
    const sectionRegex = new RegExp(
      `### ${sectionName}([\\s\\S]*?)(?=### |$)`,
      'i'
    );
    const sectionMatch = content.match(sectionRegex);

    if (sectionMatch) {
      const sectionContent = sectionMatch[1];
      try {
        const usernames = extractUsernamesFromSection(sectionContent, groupName);
        userGroups.push(...usernames);
      } catch (err) {
        logger.error(`Error extracting usernames from ${sectionName}`, {}, err);
        throw err;
      }
    } else {
      const error = new Error(`Section not found: ${sectionName}`);
      logger.error(error.message);
      throw error;
    }
  }

  logger.info(
    `Extracted ${userGroups.length} user-group mappings from markdown`
  );
  return userGroups;
};

/**
 * Extracts usernames from a specific section
 * @param {string} sectionContent - Content of the section
 * @param {string} groupName - Name of the group (mdm, orchestration, software, security-compliance)
 * @returns {Array<{group: string, username: string}>} Array of user group mappings
 * @throws {Error} If Tech Lead or Developer requirements are not met
 */
const extractUsernamesFromSection = (sectionContent, groupName) => {
  const userGroups = [];

  // Look for the Tech Lead row in the table
  const techLeadRowMatch = sectionContent.match(
    /\|\s*Tech Lead\s*\|\s*([\s\S]*?)(?=\n\||\n\n|$)/
  );

  if (!techLeadRowMatch) {
    throw new Error(`No Tech Lead row found in ${groupName} group section`);
  }

  const techLeadCell = techLeadRowMatch[1];
  const techLeadUsernames = extractUsernamesFromCell(techLeadCell);

  if (techLeadUsernames.length === 0) {
    throw new Error(`No Tech Lead found in ${groupName} group`);
  }

  logger.info(
    `Found ${techLeadUsernames.length} tech lead(s) in ${groupName} group: ${techLeadUsernames.join(', ')}`
  );

  for (const username of techLeadUsernames) {
    userGroups.push({ group: groupName, username });
    userGroups.push({ group: 'engineering', username });
  }

  // Look for the Developer row in the table
  // The pattern needs to handle multi-line content in the cell
  const developerRowMatch = sectionContent.match(
    /\|\s*Developer\s*\|\s*([\s\S]*?)(?=\n\||\n\n|$)/
  );

  if (!developerRowMatch) {
    throw new Error(`No Developer row found in ${groupName} group section`);
  }

  const developerCell = developerRowMatch[1];
  const developerUsernames = extractUsernamesFromCell(developerCell);

  if (developerUsernames.length === 0) {
    throw new Error(
      `No developers found in ${groupName} group Developer row`
    );
  }

  logger.info(
    `Found ${developerUsernames.length} developer(s) in ${groupName} group: ${developerUsernames.join(', ')}`
  );

  // Create user group mappings for both the specific group and engineering
  for (const username of developerUsernames) {
    // Add to specific group (mdm, orchestration, software, security-compliance)
    userGroups.push({ group: groupName, username });

    // Add to engineering group (all developers are in engineering)
    userGroups.push({ group: 'engineering', username });
  }

  return userGroups;
};

/**
 * Extracts GitHub usernames from a table cell
 * @param {string} cellContent - Content of the table cell
 * @returns {Array<string>} Array of GitHub usernames
 */
const extractUsernamesFromCell = (cellContent) => {
  // Extract GitHub usernames from the cell
  // Look for patterns like [@username](https://github.com/username)
  // Note: This match could fail with slight variations in formatting (extra spaces, different brackets, etc.).
  const usernameMatches = cellContent.match(/\[@([a-zA-Z0-9-]+)]\([^)]+\)/g);

  if (!usernameMatches) {
    return [];
  }

  return usernameMatches
    .map((match) => {
      // Extract username from _([@username](url))_ format
      const usernameMatch = match.match(/\[@([a-zA-Z0-9-]+)]/);
      return usernameMatch ? usernameMatch[1] : null;
    })
    .filter(Boolean);
};

/**
 * Validates the structure of the markdown content
 * @param {string} content - Markdown content to validate
 * @returns {boolean} True if structure is valid, false otherwise
 */
export const validateMarkdownStructure = (content) => {
  const requiredSections = [
    'MDM group',
    'Orchestration group',
    'Software group',
    'Security & compliance group',
  ];

  for (const section of requiredSections) {
    if (!content.includes(`### ${section}`)) {
      logger.warn(`Missing required section: ${section}`);
      return false;
    }
  }

  return true;
};

export default {
  parseProductGroups,
  validateMarkdownStructure,
};
