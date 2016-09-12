import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import Footer from './Footer';

describe('Footer - component', () => {
  it('renders the Kolide logo', () => {
    const footer = mount(<Footer />);
    const kolideLogo = footer.find('KolideLogo');

    expect(kolideLogo.length).toEqual(1);
  });

  it('renders the Kolide text logo', () => {
    const footer = mount(<Footer />);
    const kolideTextLogo = footer.find('KolideText');

    expect(kolideTextLogo.length).toEqual(1);
  });
});
