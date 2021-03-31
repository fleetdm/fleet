const DEFAULT_RESULTS_NAME = 'results';

const generateResultsCountText = (name: string = DEFAULT_RESULTS_NAME, pageIndex: number, pageSize: number, resultsCount: number) => {
  if (resultsCount === 0) return `No ${name}`;

  if (pageSize === resultsCount) return `${pageSize}+ ${name}`;
  if (pageIndex !== 0 && (resultsCount <= pageSize)) return `${pageSize}+ ${name}`;
  return `${resultsCount} ${name}`;
};

export default { generateResultsCountText };
