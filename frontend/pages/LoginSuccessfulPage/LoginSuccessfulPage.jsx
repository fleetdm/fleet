import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import radium from 'radium';
import componentStyles from './styles';
import Icon from '../../components/icons/Icon';
import paths from '../../router/paths';

const REDIRECT_TIME = 1200;

class LoginSuccessfulPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
  };

  constructor (props) {
    super(props);

    this.state = {
      redirectTime: REDIRECT_TIME,
    };
  }

  componentDidMount () {
    this.startRedirectCountdown();
  }

  startRedirectCountdown = () => {
    const { dispatch } = this.props;
    const { HOME } = paths;
    const { redirectTime } = this.state;

    setTimeout(() => {
      return dispatch(push(HOME));
    }, redirectTime);
  }

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
