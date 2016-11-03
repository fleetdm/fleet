import React, { Component, PropTypes } from 'react';

import Check from './svg/Check';
import Clipboard from './svg/Clipboard';
import Envelope from './svg/Envelope';
import KolideLoginBackground from './svg/KolideLoginBackground';
import Lock from './svg/Lock';
import User from './svg/User';

class Icon extends Component {
  static propTypes = {
    name: PropTypes.string.isRequired,
    className: PropTypes.string,
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

    return <IconComponent {...this.props} className={this.props.className} />;
  }
}

export default Icon;
