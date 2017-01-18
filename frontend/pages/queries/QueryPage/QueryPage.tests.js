import expect, { restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { defaultSelectedOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import helpers from 'test/helpers';
import kolide from 'kolide';
import QueryPage from 'pages/queries/QueryPage';
import { validUpdateQueryRequest } from 'test/mocks';

const { connectedComponent, createAceSpy, fillInFormInput, reduxMockStore } = helpers;
const locationProp = { params: {} };

describe('QueryPage - component', () => {
  beforeEach(createAceSpy);
  afterEach(restoreSpies);

  const mockStore = reduxMockStore({
    components: {
      QueryPages: {
        queryText: 'SELECT * FROM users',
        selectedOsqueryTable: defaultSelectedOsqueryTable,
        selectedTargets: [],
      },
    },
    entities: {
      queries: {},
      targets: {},
    },
  });

  it('renders the QueryForm component', () => {
    const page = mount(connectedComponent(QueryPage, { mockStore, props: locationProp }));

    expect(page.find('QueryForm').length).toEqual(1);
  });

  it('renders the QuerySidePanel component', () => {
    const page = mount(connectedComponent(QueryPage, { mockStore, props: locationProp }));

    expect(page.find('QuerySidePanel').length).toEqual(1);
  });

  it('sets targetError in state when the query is run and there are no selected targets', () => {
    const page = mount(connectedComponent(QueryPage, { mockStore, props: locationProp }));
    const form = page.find('QueryForm');
    const runQueryBtn = form.find('.query-form__run-query-btn');

    expect(form.prop('targetsError')).toNotExist();

    runQueryBtn.simulate('click');

    expect(form.prop('targetsError')).toEqual('You must select at least one target to run a query');
  });

  it('calls the onUpdateQuery prop when the query is updated', () => {
    const bearerToken = 'abc123';
    const locationWithQueryProp = { params: { id: 1 } };
    const query = { id: 1, name: 'My query', description: 'My query description', query: 'select * from users' };
    const mockStoreWithQuery = reduxMockStore({
      components: {
        QueryPages: {
          queryText: 'SELECT * FROM users',
          selectedOsqueryTable: defaultSelectedOsqueryTable,
          selectedTargets: [],
        },
      },
      entities: {
        queries: {
          data: {
            1: query,
          },
        },
      },
    });
    const page = mount(connectedComponent(QueryPage, {
      mockStore: mockStoreWithQuery,
      props: locationWithQueryProp,
    }));
    const form = page.find('QueryForm');
    const nameInput = form.find({ name: 'name' }).find('input');
    const saveChangesBtn = form.find('li.dropdown-button__option').first().find('Button');

    kolide.setBearerToken(bearerToken);
    validUpdateQueryRequest(bearerToken, query, {
      description: query.description,
      name: 'new name',
      queryText: 'SELECT * FROM users',
    });
    fillInFormInput(nameInput, 'new name');

    form.simulate('submit');
    saveChangesBtn.simulate('click');

    expect(mockStoreWithQuery.getActions()).toInclude({
      type: 'queries_UPDATE_REQUEST',
    });
  });
});
