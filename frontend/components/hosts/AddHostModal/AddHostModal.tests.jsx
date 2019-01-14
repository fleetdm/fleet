import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import AddHostModal from './AddHostModal';

describe('AddHostModal - component', () => {
  it('clicking Reveal Secret should change input type', () => {
    const component = mount(<AddHostModal dispatch={noop} onReturnToApp={noop} />);
    const revealSecretLink = component.find('.add-host-modal__reveal-secret');
    let secretInput = component.find('.add-host-modal__secret-input').find('input');

    expect(component.state().revealSecret).toEqual(false);
    expect(secretInput.prop('type')).toEqual('password');
    expect(revealSecretLink.text()).toEqual('Reveal Secret');

    revealSecretLink.simulate('click');

    secretInput = component.find('.add-host-modal__secret-input').find('input');
    expect(component.state().revealSecret).toEqual(true);
    expect(secretInput.prop('type')).toEqual('text');
    expect(revealSecretLink.text()).toEqual('Hide Secret');

    revealSecretLink.simulate('click');

    secretInput = component.find('.add-host-modal__secret-input').find('input');
    expect(component.state().revealSecret).toEqual(false);
    expect(secretInput.prop('type')).toEqual('password');
    expect(revealSecretLink.text()).toEqual('Reveal Secret');
  });
});
