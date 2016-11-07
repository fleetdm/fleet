import expect, { restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { defaultSelectedOsqueryTable } from '../../../redux/nodes/components/QueryPages/actions';
import helpers from '../../../test/helpers';
import QueryPage from './QueryPage';

const { connectedComponent, createAceSpy, reduxMockStore } = helpers;
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
      targets: {},
    },
  });

  it('renders the QueryComposer component', () => {
    const page = mount(connectedComponent(QueryPage, { mockStore, props: locationProp }));

    expect(page.find('QueryComposer').length).toEqual(1);
  });

  it('renders the QuerySidePanel component', () => {
    const page = mount(connectedComponent(QueryPage, { mockStore, props: locationProp }));

    expect(page.find('QuerySidePanel').length).toEqual(1);
  });
});
