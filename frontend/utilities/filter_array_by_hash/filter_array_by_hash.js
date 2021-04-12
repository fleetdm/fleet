import { every, filter, keys, pick } from "lodash";

const filterArrayByHash = (array, arrayFilter) => {
  return filter(array, (obj) => {
    const filterKeys = keys(arrayFilter);

    return every(pick(obj, filterKeys), (val, key) => {
      const arrayFilterValue = arrayFilter[key];

      if (!arrayFilterValue) {
        return true;
      }

      const lowerVal = val.toLowerCase();
      const lowerArrayFilterValue = arrayFilterValue.toLowerCase();

      return lowerVal.includes(lowerArrayFilterValue);
    });
  });
};

export default filterArrayByHash;
