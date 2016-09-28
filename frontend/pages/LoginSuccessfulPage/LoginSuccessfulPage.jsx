import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import radium from 'radium';
import componentStyles from './styles';
import Icon from '../../components/icons/Icon';


class LoginSuccessfulPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
  };

  render () {
    const { loginSuccessStyles, subtextStyles, whiteBoxStyles } = componentStyles;
    return (
      <div style={whiteBoxStyles}>
        <Icon name="check" />
        <p style={loginSuccessStyles}>Login successful</p>
        <p style={subtextStyles}>hold on to your butts...</p>
      </div>
    );
  }
}

const ConnectedComponent = connect()(LoginSuccessfulPage);
export default radium(ConnectedComponent);
