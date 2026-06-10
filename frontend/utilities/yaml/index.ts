import yaml from "js-yaml";

interface IYAMLError {
  name: string;
  reason: string;
  line: string;
}

export const constructErrorString = (yamlError: IYAMLError) => {
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export const agentOptionsToYaml = (agentOpts: any) => {
  agentOpts ||= { config: {} };

  // hide the "overrides" key if it is empty
  if (!agentOpts.overrides || Object.keys(agentOpts.overrides).length === 0) {
    delete agentOpts.overrides;
  }

  // when the "command_line_flags" key is absent, suggest it with a comment.
  // If it is present — even set to {} or null — render it as-is, since those
  // empty values have special semantics.
  let yamlString = yaml.dump(agentOpts);
  if (!("command_line_flags" in agentOpts)) {
    yamlString +=
      "# Requires Fleet's osquery installer\n" +
      "# Setting this to null or {} will clear all local osquery flags on hosts\n" +
      "# command_line_flags: {}\n";
  }

  return yamlString;
};

export default constructErrorString;
