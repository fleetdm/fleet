import { filter } from "lodash";

const filterQueries = (queries, filterText) => {
  const regex = new RegExp(filterText.toLowerCase());

  return filter(queries, (query) => {
    const lowerQueryName = query.name.toLowerCase();

    return regex.test(lowerQueryName);
  });
};

export default { filterQueries };
