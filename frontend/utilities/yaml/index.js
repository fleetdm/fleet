const constructErrorString = (yamlError) => {
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export default constructErrorString;
