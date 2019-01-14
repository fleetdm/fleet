import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import nock from 'nock';
import { noop } from 'lodash';

import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import Test from 'test';

describe('SelectTargetsDropdown - component', () => {
  beforeEach(() => Test.Mocks.targetMock().persist());

  const defaultProps = {
    disabled: false,
    label: 'Select Targets',
    onFetchTargets: noop,
    onSelect: noop,
    selectedTargets: [],
    targetsCount: 0,
  };
  const DefaultComponent = mount(<SelectTargetsDropdown {...defaultProps} />);

  afterEach(() => nock.cleanAll());

  it('sets default state', () => {
    expect(DefaultComponent.state()).toEqual({
      isEmpty: false,
      isLoadingTargets: false,
      moreInfoTarget: null,
      query: '',
      targets: [],
    });
  });

  describe('rendering', () => {
    it('renders', () => {
      expect(DefaultComponent.length).toEqual(1, 'Expected component to render');
    });

    it('renders the SelectTargetsInput', () => {
      const SelectTargetsInput = DefaultComponent.find('SelectTargetsInput');

      expect(SelectTargetsInput.length).toEqual(1, 'Expected SelectTargetsInput to render');
    });

    it('renders a label when passed as a prop', () => {
      const noLabelProps = { ...defaultProps, label: undefined };
      const ComponentWithoutLabel = mount(<SelectTargetsDropdown {...noLabelProps} />);
      const Label = DefaultComponent.find('.target-select__label');
      const NoLabel = ComponentWithoutLabel.find('.target-select__label');

      expect(Label.length).toEqual(1, 'Expected label to render');
      expect(NoLabel.length).toEqual(0, 'Expected label to not render');
    });

    it('renders the error when passed as a prop', () => {
      const errorProps = { ...defaultProps, error: "You can't do this!" };
      const ErrorComponent = mount(<SelectTargetsDropdown {...errorProps} />);
      const Error = ErrorComponent.find('.target-select__label--error');
      const NoError = DefaultComponent.find('.target-select__label--error');

      expect(Error.length).toEqual(1, 'Expected error to render');
      expect(NoError.length).toEqual(0, 'Expected error to not render');
    });

    it('renders the target count', () => {
      const targetCountProps = { ...defaultProps, targetsCount: 10 };
      const TargetCountComponent = mount(<SelectTargetsDropdown {...targetCountProps} />);

      expect(DefaultComponent.text()).toInclude('0 unique hosts');
      expect(TargetCountComponent.text()).toInclude('10 unique hosts');
    });
  });

  describe('#fetchTargets', () => {
    const apiResponseWithTargets = {
      targets: {
        hosts: [],
        labels: [Test.Stubs.labelStub],
      },
    };
    const apiResponseWithoutTargets = {
      targets: {
        hosts: [],
        labels: [],
      },
    };
    const defaultSelectedTargets = { hosts: [], labels: [] };
    const defaultParams = {
      query: '',
      selected: defaultSelectedTargets,
    };
    const expectedApiClientResponseWithTargets = {
      targets: [{ ...Test.Stubs.labelStub, target_type: 'labels' }],
    };

    afterEach(() => restoreSpies());

    it('calls the api', () => {
      nock.cleanAll();
      Test.Mocks.targetMock(defaultParams, apiResponseWithTargets);
      const Component = mount(<SelectTargetsDropdown {...defaultProps} />);
      const node = Component.instance();

      const request = Test.Mocks.targetMock(defaultParams, apiResponseWithTargets);

      return node.fetchTargets()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it('calls the onFetchTargets prop', () => {
      nock.cleanAll();
      Test.Mocks.targetMock(defaultParams, apiResponseWithTargets).persist();
      const onFetchTargets = createSpy();
      const props = { ...defaultProps, onFetchTargets };
      const Component = mount(<SelectTargetsDropdown {...props} />);
      const node = Component.instance();


      return node.fetchTargets()
        .then(() => {
          expect(onFetchTargets).toHaveBeenCalledWith('', expectedApiClientResponseWithTargets);
        });
    });

    it('does not call the onFetchTargets prop when the component is not mounted', () => {
      const onFetchTargets = createSpy();
      const props = { ...defaultProps, onFetchTargets };
      const Component = mount(<SelectTargetsDropdown {...props} />);
      const node = Component.instance();

      node.mounted = false;

      expect(node.fetchTargets()).toEqual(false);
      expect(onFetchTargets).toNotHaveBeenCalled();
    });

    it('sets state correctly when no targets are returned', () => {
      const Component = mount(<SelectTargetsDropdown {...defaultProps} />);
      const node = Component.instance();

      Test.Mocks.targetMock(defaultParams, apiResponseWithoutTargets);

      return node.fetchTargets()
        .then(() => {
          expect(Component.state('isEmpty')).toEqual(true);
          expect(Component.state('targets')).toEqual([{}]);
          expect(Component.state('isLoadingTargets')).toEqual(false);
        });
    });

    it('returns the query', () => {
      const query = 'select * from users';
      const Component = mount(<SelectTargetsDropdown {...defaultProps} />);
      const node = Component.instance();

      Test.Mocks.targetMock({ ...defaultParams, query });

      return node.fetchTargets(query)
        .then((q) => {
          expect(q).toEqual(query);
        });
    });
  });
});
