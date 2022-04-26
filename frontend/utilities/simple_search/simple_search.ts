import { filter, includes } from "lodash";

interface IDictionary {
  [key: string]: any;
}

const simpleSearch = (
  searchQuery = "",
  dictionary: IDictionary | undefined
) => {
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
