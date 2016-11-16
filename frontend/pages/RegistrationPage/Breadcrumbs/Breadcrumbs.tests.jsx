import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import Breadcrumbs from 'pages/RegistrationPage/Breadcrumbs';

describe('Breadcrumbs - component', () => {
  afterEach(restoreSpies);

  it('renders 3 Button components', () => {
    const component = mount(<Breadcrumbs page={1} />);
    expect(component.find('button').length).toEqual(3);
  });

  it('renders page 1 Button as active when the page prop is 1', () => {
    const component = mount(<Breadcrumbs page={1} />);
    const page1Btn = component.find('button.page-1-btn');
    const page2Btn = component.find('button.page-2-btn');
    const page3Btn = component.find('button.page-3-btn');

    expect(page1Btn.prop('className')).toInclude('is-active');
    expect(page2Btn.prop('className')).toNotInclude('is-active');
    expect(page3Btn.prop('className')).toNotInclude('is-active');
  });

  it('calls the onClick prop with the page # when clicked', () => {
    const onClickSpy = createSpy();
    const component = mount(<Breadcrumbs page={1} onClick={onClickSpy} />);
    const page1Btn = component.find('button.page-1-btn');
    const page2Btn = component.find('button.page-2-btn');
    const page3Btn = component.find('button.page-3-btn');

    page1Btn.simulate('click');

    expect(onClickSpy).toHaveBeenCalledWith(1);

    page2Btn.simulate('click');

    expect(onClickSpy).toHaveBeenCalledWith(2);

    page3Btn.simulate('click');

    expect(onClickSpy).toHaveBeenCalledWith(3);
  });
});
