import expect from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import { packStub } from 'test/stubs';
import EditPackPage from './EditPackPage';

describe('EditPackPage - component', () => {
  const store = {
    entities: {
      packs: {
        data: {
          [packStub.id]: packStub,
        },
      },
      scheduled_queries: {},
    },
  };
  const page = mount(connectedComponent(EditPackPage, {
    props: { params: { id: String(packStub.id) }, route: {} },
    mockStore: reduxMockStore(store),
  }));

  it('renders', () => {
    expect(page.length).toEqual(1);
  });

  it('renders a EditPackFormWrapper component', () => {
    expect(page.find('EditPackFormWrapper').length).toEqual(1);
  });

  it('renders a ScheduleQuerySidePanel component', () => {
    expect(page.find('ScheduleQuerySidePanel').length).toEqual(1);
  });
});
