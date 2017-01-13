const replaceArrayItem = (array, item, replacement) => {
  const index = array.indexOf(item);

  if (index === -1) {
    return array;
  }

  array[index] = replacement;

  return array;
};

export default replaceArrayItem;
