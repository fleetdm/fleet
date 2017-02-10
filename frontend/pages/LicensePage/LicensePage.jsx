import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';

import { setupLicense } from 'redux/nodes/auth/actions';
import EnsureUnauthenticated from 'components/EnsureUnauthenticated';
import Footer from 'components/Footer';
import LicenseForm from 'components/forms/LicenseForm';
import licenseInterface from 'interfaces/license';
import LicenseSuccess from 'components/LicenseSuccess';
import { showBackgroundImage } from 'redux/nodes/app/actions';

import kolideLogo from '../../../assets/images/kolide-logo-condensed.svg';

const baseClass = 'license-page';

class LicensePage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
      license: PropTypes.string,
    }),
    license: licenseInterface.isRequired,
  };

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(showBackgroundImage);

    return false;
  }

  handleSubmit = ({ license }) => {
    const { dispatch } = this.props;

    dispatch(setupLicense({ license }))
      .catch(() => false);

    return false;
  }

  render () {
    const { handleSubmit } = this;
    const { errors, license } = this.props;

    if (license.token) {
      return (
        <div className={baseClass}>
          <img
            alt="Kolide"
            src={kolideLogo}
            className={`${baseClass}__logo`}
          />
          <LicenseSuccess license={license} />
          <Footer />
        </div>
      );
    }

    return (
      <div className={baseClass}>
        <img
          alt="Kolide"
          src={kolideLogo}
          className={`${baseClass}__logo`}
        />
        <LicenseForm handleSubmit={handleSubmit} serverErrors={errors} />
        <Footer />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { errors, license } = state.auth;

  return { errors, license };
};

const ConnectedComponent = connect(mapStateToProps)(LicensePage);
export default EnsureUnauthenticated(ConnectedComponent);
