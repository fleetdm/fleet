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
    logger.error(`Error parsing product groups file: ${filePath}`, err);
    return [];
  }
};

/**
 * Extracts usernames from markdown content
 * @param {string} content - Markdown content
 * @returns {Array<{group: string, username: string}>} Array of user group mappings
 */
const extractUsernamesFromMarkdown = (content) => {
  const userGroups = [];

  // Define the groups we're looking for and their corresponding database group names
  const groupMappings = {
    'MDM group': 'mdm',
    'Orchestration group': 'orchestration',
    'Software group': 'software'
  };

  // For each group, find its section and extract usernames
  for (const [sectionName, groupName] of Object.entries(groupMappings)) {
    // Find the section for this group
    const sectionRegex = new RegExp(`### ${sectionName}([\\s\\S]*?)(?=### |$)`, 'i');
    const sectionMatch = content.match(sectionRegex);

    if (sectionMatch) {
      const sectionContent = sectionMatch[1];
      const usernames = extractUsernamesFromSection(sectionContent, groupName);
      userGroups.push(...usernames);
    } else {
      logger.warn(`Section not found: ${sectionName}`);
    }
  }

  logger.info(`Extracted ${userGroups.length} user-group mappings from markdown`);
  return userGroups;
};

/**
 * Extracts usernames from a specific section
 * @param {string} sectionContent - Content of the section
 * @param {string} groupName - Name of the group (mdm, orchestration, software)
 * @returns {Array<{group: string, username: string}>} Array of user group mappings
 */
const extractUsernamesFromSection = (sectionContent, groupName) => {
  const userGroups = [];

  // Look for the Developer row in the table
  // The pattern needs to handle multi-line content in the cell
  const developerRowMatch = sectionContent.match(/\|\s*Developer\s*\|\s*([\s\S]*?)(?=\n\||\n\n|$)/);

  if (!developerRowMatch) {
    logger.warn(`No Developer row found in ${groupName} group section`);
    return userGroups;
  }

  const developerCell = developerRowMatch[1];

  // Extract GitHub usernames from the developer cell
  // Look for patterns like _([@username](https://github.com/username))_
  const usernameMatches = developerCell.match(/_\(\[@([a-zA-Z0-9-]+)]\([^)]+\)\)_/g);

  if (!usernameMatches) {
    logger.warn(`No GitHub usernames found in ${groupName} group Developer row`);
    return userGroups;
  }

  const usernames = usernameMatches.map(match => {
    // Extract username from _([@username](url))_ format
    const usernameMatch = match.match(/_\(\[@([a-zA-Z0-9-]+)]/);
    return usernameMatch ? usernameMatch[1] : null;
  }).filter(Boolean);

  logger.info(`Found ${usernames.length} developers in ${groupName} group: ${usernames.join(', ')}`);

  // Create user group mappings for both the specific group and engineering
  for (const username of usernames) {
    // Add to specific group (mdm, orchestration, software)
    userGroups.push({ group: groupName, username });

    // Add to engineering group (all developers are in engineering)
    userGroups.push({ group: 'engineering', username });
  }

  return userGroups;
};

/**
 * Validates the structure of the markdown content
 * @param {string} content - Markdown content to validate
 * @returns {boolean} True if structure is valid, false otherwise
 */
export const validateMarkdownStructure = (content) => {
  const requiredSections = ['MDM group', 'Orchestration group', 'Software group'];

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
  validateMarkdownStructure
};
