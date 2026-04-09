import yaml from "js-yaml";

interface IYAMLError {
  name: string;
  reason: string;
  line: string;
}

export const constructErrorString = (yamlError: IYAMLError) => {
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export const agentOptionsToYaml = (agentOpts: unknown) => {
  const opts: Record<string, unknown> = (agentOpts as
    | Record<string, unknown>
    | null
    | undefined) ?? { config: {} };

  // hide the "overrides" key if it is empty
  const overrides = opts.overrides as Record<string, unknown> | undefined;
  if (!overrides || Object.keys(overrides).length === 0) {
    delete opts.overrides;
  }

  // add a comment besides the "command_line_flags" if it is empty
  let addFlagsComment = false;
  const commandLineFlags = opts.command_line_flags as
    | Record<string, unknown>
    | undefined;
  if (!commandLineFlags || Object.keys(commandLineFlags).length === 0) {
    // delete it so it does not render, and will add it explicitly after (along with the comment)
    delete opts.command_line_flags;
    addFlagsComment = true;
  }

  let yamlString = yaml.dump(opts);
  if (addFlagsComment) {
    yamlString +=
      "# Requires Fleet's osquery installer\n# command_line_flags: {}\n";
  }

  return yamlString;
};

export default constructErrorString;
