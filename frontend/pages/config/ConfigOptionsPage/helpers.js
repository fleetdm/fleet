import { filter, find, flatMap, size } from 'lodash';
import replaceArrayItem from 'utilities/replace_array_item';

const configOptionDropdownOptions = (configOptions) => {
  return flatMap(configOptions, (option) => {
    if (option.value !== null) {
      return [];
    }

    return {
      disabled: option.read_only || false,
      label: option.name,
      value: option.name,
    };
  });
};

const configErrorsFor = (changedOptions, allOptions) => {
  const errors = {};

  changedOptions.forEach((option) => {
    const { id, name } = option;
    const optionErrors = {};

    if (!name) {
      optionErrors.name = 'Must be present';
    }

    if (name) {
      const configOptionsWithName = filter(allOptions, { name });

      if (configOptionsWithName.length > 1) {
        optionErrors.name = 'Must be unique';
      }
    }

    if (size(optionErrors)) {
      errors[id] = optionErrors;
    }
  });

  const valid = !size(errors);

  return { errors, valid };
};

const formatOptionsForServer = (options) => {
  return options.map((option) => {
    const { type, value } = option;

    if (value === null) {
      return option;
    }

    switch (type) {
      case 'int':
        return { ...option, value: Number(value) };
      case 'bool':
        return {
          ...option,
          value: (value === 'true') || (value === true),
        };
      case 'string':
        return { ...option, value: String(value) };
      default:
        return option;
    }
  });
};

const updatedConfigOptions = ({ oldOption, newOption, configOptions }) => {
  const existingConfigOption = find(configOptions, { name: newOption.name });
  const newValue = newOption.value || oldOption.value;
  const updatedConfigOption = { ...existingConfigOption, name: newOption.name, value: newValue };

  // we are making an update to the same option so only need to replace it
  if (updatedConfigOption.id === oldOption.id) {
    return replaceArrayItem(configOptions, oldOption, updatedConfigOption);
  }

  // we are changing the option name so we need to remove the other
  // option with the same name before replacing the current option
  const filteredConfigOptions = filter(configOptions, o => o.id !== updatedConfigOption.id);
  const option = { ...oldOption, value: null };

  return replaceArrayItem(filteredConfigOptions, oldOption, updatedConfigOption).concat(option);
};

export default { configErrorsFor, configOptionDropdownOptions, formatOptionsForServer, updatedConfigOptions };
