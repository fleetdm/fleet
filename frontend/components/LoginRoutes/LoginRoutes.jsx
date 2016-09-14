import React, { Component, PropTypes } from 'react';
import componentStyles from './styles';
import { loadBackground, removeBackground, resizeBackground } from '../../utilities/backgroundImage';

export class LoginRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
  };

  componentWillMount () {
    const { window } = global;

    loadBackground();
    window.onresize = resizeBackground;
  }

  componentWillUnmount () {
    removeBackground();
  }

  render () {
    const { containerStyles } = componentStyles;
    const { children } = this.props;

    return (
      <div style={containerStyles}>
        <img alt="Kolide text logo" src="/assets/images/kolide-logo-text.svg" />
        {children}
      </div>
    );
  }
}

export default LoginRoutes;

