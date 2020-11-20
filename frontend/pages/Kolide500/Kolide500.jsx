import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { resetErrors } from 'redux/nodes/errors500/actions';
import errorsInterface from 'interfaces/errors500';

import fleetLogoText from '../../../assets/images/fleet-logo-text-white.svg';
import backgroundImg from '../../../assets/images/500.svg';

const baseClass = 'kolide-500';

class Kolide500 extends Component {
  static propTypes = {
    errors: errorsInterface,
    dispatch: PropTypes.func,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = {
      showErrorMessage: false,
    };
  }

  componentWillUnmount() {
    const { dispatch } = this.props;
    dispatch(resetErrors());
  }

  onShowErrorMessage = () => {
    this.setState({ showErrorMessage: true });
  }

  renderError = () => {
    const { errors } = this.props;
    const errorMessage = errors ? errors.base : null;
    const { showErrorMessage } = this.state;
    const { onShowErrorMessage } = this;

    if (errorMessage && !showErrorMessage) {
      // We only show the button when errorMessage exists
      // and showErrorMessage is set to false
      return (
        <button className="button button--muted" onClick={onShowErrorMessage}>SHOW ERROR</button>
      );
    }

    if (errorMessage && showErrorMessage) {
      // We only show the error message when errorMessage exists
      // and showErrorMessage is set to true
      return (
        <div className="error-message-container">
          <p>{errorMessage}</p>
        </div>
      );
    }

    return false;
  }

  render () {
    const { renderError } = this;

    return (
      <div className={baseClass}>
        <header className="primary-header">
          <a href="/">
            <img className="primary-header__logo" src={fleetLogoText} alt="Kolide" />
          </a>
        </header>
        <img className="background-image" src={backgroundImg}></img>
        <main>
          <h1>500: Oh, something went wrong.</h1>
          <p>Please file an issue if you believe this is a bug.</p>
          {renderError()}
          <a
            href="https://github.com/fleetdm/fleet/issues"
            target="_blank"
            rel="noopener noreferrer"
          >
            File an issue
          </a>
        </main>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { errors } = state.errors500;
  return {
    errors,
  };
};

export default connect(mapStateToProps)(Kolide500);
