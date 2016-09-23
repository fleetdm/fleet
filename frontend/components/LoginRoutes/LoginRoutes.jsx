import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import componentStyles from './styles';
import { hideBackgroundImage, showBackgroundImage } from '../../redux/nodes/app/actions';

export class LoginRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
  };

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(showBackgroundImage);
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(hideBackgroundImage);
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

export default connect()(LoginRoutes);
