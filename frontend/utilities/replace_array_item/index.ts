const replaceArrayItem = (
  array: unknown[],
  item: unknown,
  replacement: unknown
) => {
  const index = array.indexOf(item);

  if (index === -1) {
    return array;
  }

  array[index] = replacement;

  return array;
};

export default replaceArrayItem;
