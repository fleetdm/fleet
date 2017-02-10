import React from 'react';
import moment from 'moment';

import Icon from 'components/icons/Icon';
import licenseInterface from 'interfaces/license';

import key from '../../../assets/images/key.svg';

const baseClass = 'license-success';

const LicenseSuccess = ({ license }) => {
  const { allowed_hosts: allowedHosts, expiry } = license;
  const expiryMoment = moment(expiry);
  const timeToExpiration = expiryMoment.toNow(true);
  const hostText = allowedHosts.count === 1 ? 'Host' : 'Hosts';

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__container`}>
        <h2><Icon name="success-check" />License Upload Successful!</h2>
        <h3>
          <img alt="Kolide License" className={`${baseClass}__key-img`} src={key} />
          Kolide License Details:
        </h3>
        <h4>Current License Level:</h4>
        <ul>
          <li><Icon name="single-host" />{allowedHosts}&nbsp;{hostText}</li>
          {timeToExpiration && <li><Icon name="clock" />Expires in {timeToExpiration}</li>}
        </ul>
        <a href="/setup" className="button button--success">
          SETUP KOLIDE
        </a>
      </div>
    </div>
  );
};

LicenseSuccess.propTypes = {
  license: licenseInterface.isRequired,
};

export default LicenseSuccess;

