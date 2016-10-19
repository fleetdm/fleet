import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import Check from './svg/Check';
import Clipboard from './svg/Clipboard';
import Envelope from './svg/Envelope';
import KolideLoginBackground from './svg/KolideLoginBackground';
import Lock from './svg/Lock';
import User from './svg/User';

class Icon extends Component {
  static propTypes = {
    alt: PropTypes.string,
    name: PropTypes.string.isRequired,
    onClick: PropTypes.func,
    style: PropTypes.object,
    variant: PropTypes.string,
  };

  static iconNames = {
    check: Check,
    clipboard: Clipboard,
    envelope: Envelope,
    kolideLoginBackground: KolideLoginBackground,
    lock: Lock,
    user: User,
  };

  render () {
    const IconComponent = Icon.iconNames[this.props.name];

    return <IconComponent {...this.props} />;
  }
}

export default radium(Icon);
