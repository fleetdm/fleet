interface IYAMLError {
  name: string;
  reason: string;
  line: string;
}

const constructErrorString = (yamlError: IYAMLError) => {
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export default constructErrorString;
