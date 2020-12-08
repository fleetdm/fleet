import React from 'react';
import { shallow } from 'enzyme';

import WarningBanner from 'components/WarningBanner/WarningBanner';

describe('WarningBanner - component', () => {
  it('renders default banner', () => {
    const props = { shouldShowWarning: true, message: 'message' };
    const component = shallow(<WarningBanner {...props} />);
    expect(component.length).toEqual(1);
    expect(component.find('Icon').props().name).toEqual('warning-filled');
    expect(component.find('.warning-banner__label').text()).toEqual('Warning:');
    expect(component.find('.warning-banner__text').text()).toEqual('message');
  });

  it('renders custom label', () => {
    const props = { shouldShowWarning: true, message: 'message', labelText: 'label' };
    const component = shallow(<WarningBanner {...props} />);
    expect(component.find('.warning-banner__label').text()).toEqual('label');
  });

  it('renders empty when disabled', () => {
    const props = { shouldShowWarning: false, message: 'message' };
    const component = shallow(<WarningBanner {...props} />);
    expect(component.html()).toBe(null);
  });

  it('handles dismiss action', () => {
    const spy = jest.fn();
    const props = { shouldShowWarning: true, message: 'message', onDismiss: spy };
    const component = shallow(<WarningBanner {...props} />);

    component.find('Button').simulate('click');
    expect(spy).toHaveBeenCalled();
  });

  it('handles resolve action', () => {
    const spy = jest.fn();
    const props = { shouldShowWarning: true, message: 'message', onResolve: spy };
    const component = shallow(<WarningBanner {...props} />);

    component.find('Button').simulate('click');
    expect(spy).toHaveBeenCalled();
  });
});
