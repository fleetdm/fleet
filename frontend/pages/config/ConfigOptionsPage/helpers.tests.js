import expect from 'expect';

import { configOptionStub } from 'test/stubs';
import helpers from 'pages/config/ConfigOptionsPage/helpers';

describe('ConfigOptionsPage - helpers', () => {
  describe('#configOptionDropdownOptions', () => {
    const configOptions = [
      configOptionStub,
      { ...configOptionStub, id: 2, name: 'another_config_option' },
      { ...configOptionStub, id: 3, name: 'third_config_option', read_only: true },
      { id: 4, name: 'fourth_config_option', value: null, read_only: true },
      { id: 5, name: 'fifth_config_option', value: '' },
      { id: 6, name: 'sixth_config_option', value: null, read_only: false },
    ];

    it('returns the available dropdown options', () => {
      expect(helpers.configOptionDropdownOptions(configOptions)).toEqual([
        { label: 'fourth_config_option', value: 'fourth_config_option', disabled: true },
        { label: 'sixth_config_option', value: 'sixth_config_option', disabled: false },
      ]);
    });
  });

  describe('#configErrorsFor', () => {
    it('validates presence of the config option name', () => {
      const configOptionWithoutName = { id: 10, name: '', value: 'something' };
      const configOptionWithoutValue = { id: 11, name: 'something', value: '' };
      const configOptions = [configOptionWithoutName, configOptionWithoutValue];

      expect(helpers.configErrorsFor(configOptions, configOptions)).toEqual({
        valid: false,
        errors: {
          10: { name: 'Must be present' },
        },
      });
    });

    it('validates uniqueness of config option names', () => {
      const configOption1 = { id: 10, name: 'something', value: 'something' };
      const configOption2 = { id: 11, name: 'something', value: 'something' };
      const configOptions = [configOption1, configOption2];

      expect(helpers.configErrorsFor([configOption1], configOptions)).toEqual({
        valid: false,
        errors: {
          10: { name: 'Must be unique' },
        },
      });
    });

    it('returns an empty object when the options are valid', () => {
      const configOption1 = { id: 10, name: 'something', value: 'something' };
      const configOption2 = { id: 11, name: 'something else', value: 'something' };
      const configOptions = [configOption1, configOption2];

      expect(helpers.configErrorsFor([configOption1], configOptions)).toEqual({
        valid: true,
        errors: {},
      });
    });
  });

  describe('#formatOptionsForServer', () => {
    it('sets boolean type options correctly', () => {
      const stringOption = { id: 1, type: 'bool', name: 'utc', value: 'true' };
      const boolOption = { id: 1, type: 'bool', name: 'utc', value: true };
      const nullOption = { id: 1, type: 'bool', name: 'utc', value: null };

      expect(helpers.formatOptionsForServer([stringOption])).toEqual([
        { ...stringOption, value: true },
      ]);
      expect(helpers.formatOptionsForServer([boolOption])).toEqual([boolOption]);
      expect(helpers.formatOptionsForServer([nullOption])).toEqual([nullOption]);
    });

    it('sets int type options correctly', () => {
      const stringOption = { id: 1, type: 'int', name: 'utc', value: '100' };
      const intOption = { id: 1, type: 'int', name: 'utc', value: 100 };
      const nullOption = { id: 1, type: 'bool', name: 'utc', value: null };

      expect(helpers.formatOptionsForServer([stringOption])).toEqual([
        { ...stringOption, value: 100 },
      ]);
      expect(helpers.formatOptionsForServer([intOption])).toEqual([intOption]);
      expect(helpers.formatOptionsForServer([nullOption])).toEqual([nullOption]);
    });

    it('sets string type options correctly', () => {
      const stringOption = { id: 1, type: 'string', name: 'utc', value: 'something' };
      const intOption = { id: 1, type: 'string', name: 'utc', value: 100 };
      const boolOption = { id: 1, type: 'string', name: 'utc', value: false };
      const nullOption = { id: 1, type: 'bool', name: 'utc', value: null };

      expect(helpers.formatOptionsForServer([stringOption])).toEqual([stringOption]);
      expect(helpers.formatOptionsForServer([nullOption])).toEqual([nullOption]);
      expect(helpers.formatOptionsForServer([intOption])).toEqual([
        { ...intOption, value: '100' },
      ]);
      expect(helpers.formatOptionsForServer([boolOption])).toEqual([
        { ...boolOption, value: 'false' },
      ]);
    });
  });

  describe('#updatedConfigOptions', () => {
    it('sets the old options value to null when changing the option name', () => {
      const oldOption = { id: 2, name: 'old_option', value: 100 };
      const newOption = { id: 3, name: 'new_option' };
      const configOptions = [oldOption, newOption];

      expect(helpers.updatedConfigOptions({ oldOption, newOption: { name: 'new_option' }, configOptions })).toEqual([
        { ...newOption, value: 100 },
        { ...oldOption, value: null },
      ]);
    });

    it('updates the option value when the value changes', () => {
      const option1 = { id: 2, name: 'old_option', value: 100 };
      const option2 = { id: 3, name: 'new_option', value: null };
      const configOptions = [option1, option2];

      const updatedOptions = helpers.updatedConfigOptions({
        oldOption: option2,
        newOption: { ...option2, value: 200 },
        configOptions,
      });

      expect(updatedOptions).toEqual([
        option1,
        { ...option2, value: 200 },
      ]);
    });
  });
});
