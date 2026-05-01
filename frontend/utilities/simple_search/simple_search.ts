import { filter, includes } from "lodash";

const simpleSearch = <T extends { name?: string }>(
  searchQuery = "",
  dictionary: T[] | undefined
): T[] => {
  const lowerSearchQuery = searchQuery.toLowerCase();

  const filterResults = filter(dictionary, (item: T) => {
    if (!item.name) {
      return false;
    }

    const lowerItemName = item.name.toLowerCase();

    return includes(lowerItemName, lowerSearchQuery);
  });

  return filterResults;
};

export default simpleSearch;
