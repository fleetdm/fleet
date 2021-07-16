import { filter, includes } from "lodash";

const simpleSearch = (searchQuery = "", dictionary) => {
  const lowerSearchQuery = searchQuery.toLowerCase();

  const filterResults = filter(dictionary, (item) => {
    if (!item.name) {
      return false;
    }

    const lowerItemName = item.name.toLowerCase();

    return includes(lowerItemName, lowerSearchQuery);
  });

  return filterResults;
};

export default simpleSearch;
