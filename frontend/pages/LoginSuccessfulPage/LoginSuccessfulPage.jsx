import React from 'react';
import componentStyles from './styles';
import Icon from '../../components/icons/Icon';

const LoginSuccessfulPage = () => {
  const { loginSuccessStyles, subtextStyles, whiteBoxStyles } = componentStyles;

  return (
    <div style={whiteBoxStyles}>
      <Icon name="check" />
      <p style={loginSuccessStyles}>Login successful</p>
      <p style={subtextStyles}>Hold on to your butts.</p>
    </div>
  );
};

export default LoginSuccessfulPage;
