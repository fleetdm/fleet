import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { campaignStub } from 'test/stubs';
import QueryPageSelectTargets from 'components/queries/QueryPageSelectTargets';

describe('QueryPageSelectTargets - component', () => {
  const DEFAULT_CAMPAIGN = {
    hosts_count: {
      total: 0,
    },
  };

  const defaultProps = {
    campaign: DEFAULT_CAMPAIGN,
    onFetchTargets: noop,
    onRunQuery: noop,
    onStopQuery: noop,
    onTargetSelect: noop,
    query: 'select * from users',
    queryIsRunning: false,
    selectedTargets: [],
    targetsCount: 0,
  };

  afterEach(restoreSpies);

  describe('rendering', () => {
    const DefaultComponent = mount(<QueryPageSelectTargets {...defaultProps} />);

    it('renders', () => {
      expect(DefaultComponent.length).toEqual(1, 'QueryPageSelectTargets did not render');
    });

    it('renders a SelectTargetsDropdown component', () => {
      const SelectTargetsDropdown = DefaultComponent.find('SelectTargetsDropdown');

      expect(SelectTargetsDropdown.length).toEqual(1, 'SelectTargetsDropdown did not render');
    });

    it('renders a Run Query Button', () => {
      const RunQueryButton = DefaultComponent.find('.query-page-select-targets__run-query-btn');

      expect(RunQueryButton.length).toEqual(1, 'RunQueryButton did not render');
    });

    it('does not render a Stop Query Button', () => {
      const StopQueryButton = DefaultComponent.find('.query-page-select-targets__stop-query-btn');

      expect(StopQueryButton.length).toEqual(0, 'StopQueryButton is not expected to render');
    });

    it('does not render a Timer component', () => {
      const Timer = DefaultComponent.find('Timer');

      expect(Timer.length).toEqual(0, 'Timer is not expected to render');
    });

    it('does not render a ProgressBar component', () => {
      const ProgressBar = DefaultComponent.find('ProgressBar');

      expect(ProgressBar.length).toEqual(0, 'ProgressBar is not expected to render');
    });

    describe('when the campaign has results', () => {
      describe('and the query is running', () => {
        const props = {
          ...defaultProps,
          campaign: campaignStub,
          queryIsRunning: true,
        };

        const Component = mount(<QueryPageSelectTargets {...props} />);

        it('renders a Timer component', () => {
          const Timer = Component.find('Timer');

          expect(Timer.length).toEqual(1, 'Timer is expected to render');
        });

        it('renders a Stop Query Button', () => {
          const StopQueryButton = Component.find('.query-page-select-targets__stop-query-btn');

          expect(StopQueryButton.length).toEqual(1, 'StopQueryButton is expected to render');
        });

        it('does not render a Run Query Button', () => {
          const RunQueryButton = Component.find('.query-page-select-targets__run-query-btn');

          expect(RunQueryButton.length).toEqual(0, 'RunQueryButton is not expected render');
        });

        it('renders a ProgressBar component', () => {
          const ProgressBar = Component.find('ProgressBar');

          expect(ProgressBar.length).toEqual(1, 'ProgressBar is expected to render');
        });
      });

      describe('and the query is not running', () => {
        const props = {
          ...defaultProps,
          campaign: campaignStub,
          queryIsRunning: false,
        };
        const Component = mount(<QueryPageSelectTargets {...props} />);

        it('does not render a Timer component', () => {
          const Timer = Component.find('Timer');

          expect(Timer.length).toEqual(0, 'Timer is not expected to render');
        });

        it('does not render a Stop Query Button', () => {
          const StopQueryButton = Component.find('.query-page-select-targets__stop-query-btn');

          expect(StopQueryButton.length).toEqual(0, 'StopQueryButton is not expected to render');
        });

        it('renders a Run Query Button', () => {
          const RunQueryButton = Component.find('.query-page-select-targets__run-query-btn');

          expect(RunQueryButton.length).toEqual(1, 'RunQueryButton did not render');
        });

        it('renders a ProgressBar component', () => {
          const ProgressBar = Component.find('ProgressBar');

          expect(ProgressBar.length).toEqual(1, 'ProgressBar is expected to render');
        });
      });
    });
  });

  describe('running a query', () => {
    it('calls the onRunQuery prop with the query text', () => {
      const spy = createSpy();
      const query = 'select * from groups';
      const props = {
        ...defaultProps,
        campaign: campaignStub,
        onRunQuery: spy,
        query,
      };
      const Component = mount(<QueryPageSelectTargets {...props} />);
      const RunQueryButton = Component.find('.query-page-select-targets__run-query-btn');

      RunQueryButton.simulate('click');

      expect(spy).toHaveBeenCalledWith(query);
    });
  });

  describe('stopping a query', () => {
    it('calls the onStopQuery prop', () => {
      const spy = createSpy();
      const props = {
        ...defaultProps,
        campaign: campaignStub,
        onStopQuery: spy,
        queryIsRunning: true,
      };
      const Component = mount(<QueryPageSelectTargets {...props} />);
      const StopQueryButton = Component.find('.query-page-select-targets__stop-query-btn');

      StopQueryButton.simulate('click');

      expect(spy).toHaveBeenCalled();
    });
  });
});
