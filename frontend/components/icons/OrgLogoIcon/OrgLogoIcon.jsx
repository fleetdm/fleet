import React, { Component, PropTypes } from 'react';

import kolideLogo from '../../../../assets/images/kolide-logo.svg';

class OrgLogoIcon extends Component {
  static propTypes = {
    src: PropTypes.string.isRequired,
  };

  static defaultProps = {
    src: kolideLogo,
  };

  constructor (props) {
    super(props);

    this.state = { imageSrc: kolideLogo };
  }

  componentWillMount () {
    const { src } = this.props;

    this.setState({ imageSrc: src });

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { src } = nextProps;

    this.setState({ imageSrc: src });

    return false;
  }

  onError = () => {
    this.setState({ imageSrc: kolideLogo });

    return false;
  }

  render () {
    const { imageSrc } = this.state;
    const { onError } = this;

    return (
      <img
        alt="Organization Logo"
        onError={onError}
        src={imageSrc}
      />
    );
  }
}

export default OrgLogoIcon;
