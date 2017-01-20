import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { defaultSelectedOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import helpers from 'test/helpers';
import kolide from 'kolide';
import queryActions from 'redux/nodes/entities/queries/actions';
import QueryPage from 'pages/queries/QueryPage';
import { validUpdateQueryRequest } from 'test/mocks';
import { hostStub } from 'test/stubs';

const { connectedComponent, createAceSpy, fillInFormInput, reduxMockStore } = helpers;
const locationProp = { params: {}, location: { query: {} } };

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
      hosts: {
        data: {
          [hostStub.id]: hostStub,
          99: { ...hostStub, id: 99 },
        },
      },
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

  it('sets selectedTargets based on host_ids', () => {
    const singleHostProps = { params: {}, location: { query: { host_ids: String(hostStub.id) } } };
    const multipleHostsProps = { params: {}, location: { query: { host_ids: [String(hostStub.id), '99'] } } };
    const singleHostPage = mount(connectedComponent(QueryPage, { mockStore, props: singleHostProps }));
    const multipleHostsPage = mount(connectedComponent(QueryPage, { mockStore, props: multipleHostsProps }));

    expect(singleHostPage.find('QueryPage').prop('selectedTargets')).toEqual([hostStub]);
    expect(multipleHostsPage.find('QueryPage').prop('selectedTargets')).toEqual([hostStub, { ...hostStub, id: 99 }]);
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
    spyOn(queryActions, 'update').andCallThrough();
    const bearerToken = 'abc123';
    const locationWithQueryProp = { params: { id: 1 }, location: { query: {} } };
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

    expect(queryActions.update).toHaveBeenCalledWith(query, { name: 'new name' });
    expect(mockStoreWithQuery.getActions()).toInclude({
      type: 'queries_UPDATE_REQUEST',
    });
  });
});
