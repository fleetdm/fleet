export default (dataType) => {
  switch (dataType) {
    case "TEXT_TYPE":
      return "text";
    case "BIGINT_TYPE":
      return "big int";
    case "INTEGER_TYPE":
      return "integer";
    default:
      return dataType;
  }
};
