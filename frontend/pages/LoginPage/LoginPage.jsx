import React, { Component } from 'react';
import componentStyles from './styles';
import Icon from '../../components/icons/Icon';
import { loadBackground, resizeBackground } from '../../utilities/backgroundImage';
import LoginForm from '../../components/forms/LoginForm';

export class LoginPage extends Component {
  componentWillMount () {
    const { window } = global;

    loadBackground();
    window.onresize = resizeBackground;
  }

  onSubmit = (formData) => {
    console.log('formData', formData);
  }

  render () {
    const { containerStyles, formWrapperStyles, whiteTabStyles } = componentStyles;
    const { onSubmit } = this;

    return (
      <div style={containerStyles}>
        <div style={formWrapperStyles}>
          <Icon name="kolideText" />
          <div style={whiteTabStyles} />
          <LoginForm onSubmit={onSubmit} />
        </div>
      </div>
    );
  }
}

export default LoginPage;
