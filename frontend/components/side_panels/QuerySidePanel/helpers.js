import { includes } from 'lodash';

const DEFAULT_NUM_COLUMNS_TO_DISPLAY = 5;
const ALL_PLATFORMS_AVAILABILITY = ['specs', 'utility'];

export const columnsToRender = (table, showAllColumns) => {
  if (showAllColumns) return table.columns;

  return table.columns.slice(0, DEFAULT_NUM_COLUMNS_TO_DISPLAY);
};

export const displayTypeForDataType = (dataType) => {
  switch (dataType) {
    case 'TEXT_TYPE':
      return 'text';
    case 'BIGINT_TYPE':
      return 'big int';
    case 'INTEGER_TYPE':
      return 'integer';
    default:
      return dataType;
  }
};

export const shouldShowAllColumns = (table) => {
  const { columns } = table;

  return columns.length <= DEFAULT_NUM_COLUMNS_TO_DISPLAY;
};

export const numAdditionalColumns = (table) => {
  const { columns } = table;

  return columns.length - DEFAULT_NUM_COLUMNS_TO_DISPLAY;
};

export const availability = (platform) => {
  if (!platform) {
    return [];
  }

  if (includes(ALL_PLATFORMS_AVAILABILITY, platform.toLowerCase())) {
    return [
      {
        type: 'all',
        display_text: 'All Platforms',
      },
    ];
  }

  if (platform === 'windows') {
    return [
      {
        display_text: 'Windows',
      },
    ];
  }

  if (platform === 'posix') {
    return [
      {
        display_text: 'macOS',
      },
      {
        display_text: 'Ubuntu',
      },
      {
        display_text: 'CentOS',
      },
    ];
  }

  if (platform === 'linux') {
    return [
      {
        display_text: 'Ubuntu',
      },
      {
        display_text: 'CentOS',
      },
    ];
  }

  if (platform === 'darwin') {
    return [
      {
        display_text: 'macOS',
      },
    ];
  }

  return [
    {
      display_text: 'Unknown',
    },
  ];
};
