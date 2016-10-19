import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import helpers from '../../../test/helpers';
import QuerySidePanel from './QuerySidePanel';

const { fillInFormInput } = helpers;

describe('QuerySidePanel - component', () => {
  afterEach(restoreSpies);

  const onOsqueryTableSelect = createSpy();
  const onTextEditorInputChange = createSpy();
  const selectedOsqueryTable = {
    attributes: {},
    blacklisted: false,
    columns: [
      { description: 'User ID', name: 'uid', options: { index: true }, type: 'BIGINT_TYPE' },
      { description: 'Group ID (unsigned)', name: 'gid', options: {}, type: 'BIGINT_TYPE' },
      { description: 'User ID as int64 signed (Apple)', name: 'uid_signed', options: {}, type: 'BIGINT_TYPE' },
      { description: 'Default group ID as int64 signed (Apple)', name: 'gid_signed', options: {}, type: 'BIGINT_TYPE' },
      { description: 'Username', name: 'username', options: {}, type: 'TEXT_TYPE' },
      { description: 'Optional user description', name: 'description', options: {}, type: 'TEXT_TYPE' },
      { description: "User's home directory", name: 'directory', options: {}, type: 'TEXT_TYPE' },
      { description: "User's configured default shell", name: 'shell', options: {}, type: 'TEXT_TYPE' },
      { description: "User's UUID (Apple)", name: 'uuid', options: {}, type: 'TEXT_TYPE' },
    ],
    description: 'Local system users.',
    examples: [
      'select * from users where uid = 1000',
      "select * from users where username = 'root'",
      'select count(*) from users u, user_groups ug where u.uid = ug.uid',
    ],
    foreign_keys: [],
    function: 'genUsers',
    name: 'users',
    profile: {},
  };
  const props = {
    onOsqueryTableSelect,
    onTextEditorInputChange,
    selectedOsqueryTable,
  };

  it('renders the selected table in the dropdown', () => {
    const component = mount(<QuerySidePanel {...props} />);
    const tableSelect = component.find('.kolide-dropdown');

    expect(tableSelect.prop('value')).toEqual('users');
  });

  it('calls the onOsqueryTableSelect prop when a new table is selected in the dropdown', () => {
    const component = mount(<QuerySidePanel {...props} />);
    const tableSelect = component.find('.kolide-dropdown');
    fillInFormInput(tableSelect, 'groups');

    expect(onOsqueryTableSelect).toHaveBeenCalledWith('groups');
  });

  it('renders a CTA to show more columns', () => {
    const component = mount(<QuerySidePanel {...props} />);
    expect(component.text()).toContain('4 MORE COLUMNSSHOW');
  });

  it('calls the onTextEditorInputChange props when loading a Suggested Query', () => {
    const component = mount(<QuerySidePanel {...props} />);
    const loadSuggestedQueryButton = component.find('Button').first();
    loadSuggestedQueryButton.simulate('click');

    expect(onTextEditorInputChange).toHaveBeenCalledWith('select * from users where uid = 1000');
  });
});
