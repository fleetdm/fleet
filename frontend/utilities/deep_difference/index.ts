import { differenceWith, isArray, isEqual, isObject, map } from "lodash";

/**
 * Compares two objects and returns the differences.
 * It returns the properties from obj1 that are different from obj2.
 */
const deepDifference = (obj1: any, obj2: any) => {
  const result: any = {};

  map(obj1, (value, key) => {
    const obj2Value = obj2[key];

    if (isEqual(value, obj2Value)) return;

    if (isArray(value) && isArray(obj2Value)) {
      if (!value.length && obj2Value.length) {
        result[key] = value;
      } else {
        const arrayDiff = differenceWith(value, obj2Value, isEqual);

        if (arrayDiff.length) {
          result[key] = arrayDiff;
        }
      }
    } else if (isObject(value) && isObject(obj2Value)) {
      result[key] = deepDifference(value, obj2Value);
    } else {
      result[key] = value;
    }
  });

  return result;
};

export default deepDifference;
