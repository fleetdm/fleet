import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import radium from 'radium';
import componentStyles from './styles';
import Icon from '../../components/icons/Icon';
import paths from '../../router/paths';

const COUNTDOWN_INTERVAL = 1000;
const REDIRECT_TIME = 3000;

class LoginSuccessfulPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
    user: PropTypes.object,
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

  componentWillUnmount () {
    const { interval } = this;

    if (interval) clearInterval(interval);
  }

  startRedirectCountdown = () => {
    const { dispatch } = this.props;
    const { HOME } = paths;

    this.interval = setInterval(() => {
      const { redirectTime } = this.state;

      if (redirectTime > 0) {
        this.setState({
          redirectTime: redirectTime - COUNTDOWN_INTERVAL,
        });

        return false;
      }

      return dispatch(push(HOME));
    }, COUNTDOWN_INTERVAL);
  }

  render () {
    const { loginSuccessStyles, subtextStyles, whiteBoxStyles } = componentStyles;
    const { redirectTime } = this.state;
    const secondsToRedirect = redirectTime / 1000;

    return (
      <div style={whiteBoxStyles}>
        <Icon name="check" />
        <p style={loginSuccessStyles}>Login successful</p>
        <p style={subtextStyles}>Hold on to your butts.</p>
        <p style={subtextStyles}>redirecting in {secondsToRedirect}</p>
      </div>
    );
  }
}

const ConnectedComponent = connect()(LoginSuccessfulPage);
export default radium(ConnectedComponent);
