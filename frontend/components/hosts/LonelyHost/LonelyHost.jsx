import React from 'react';
import PropTypes from 'prop-types';
import { noop } from 'lodash';

import Button from 'components/buttons/Button';
import laptop from '../../../../assets/images/laptop-plus.svg';
import swoop from '../../../../assets/images/swoop-arrow.svg';

const baseClass = 'lonely-host';

const LonelyHost = ({ onClick = noop }) => {
  return (
    <div className={baseClass}>
      <Button variant="unstyled" className={`${baseClass}__add-host-btn`} onClick={onClick}>
        <span className={`${baseClass}__title`}>Add New Host</span>
        <span className={`${baseClass}__icon`}>
          <img src={laptop} className={`${baseClass}__laptop`} role="presentation" />
        </span>
      </Button>

      <div className={`${baseClass}__content`}>
        <h1>It&#39;s Kinda Lonely In Here...</h1>
        <h2>Get started adding hosts to Fleet.</h2>
        <p>This can be done individually or across your entire fleet.</p>
        <img src={swoop} className={`${baseClass}__swoop`} role="presentation" />
      </div>
    </div>
  );
};

LonelyHost.propTypes = {
  onClick: PropTypes.func,
};

export default LonelyHost;
