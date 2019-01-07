import React, { Component } from 'react';
import PropTypes from 'prop-types';

import kolideLogo from '../../../../assets/images/kolide-logo.svg';

class OrgLogoIcon extends Component {
  static propTypes = {
    className: PropTypes.string,
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
    const { unchangedSourceProp } = this;

    if (unchangedSourceProp(nextProps)) {
      return false;
    }

    this.setState({ imageSrc: src });

    return false;
  }

  shouldComponentUpdate (nextProps) {
    const { imageSrc } = this.state;
    const { unchangedSourceProp } = this;

    if (unchangedSourceProp(nextProps) && (imageSrc === kolideLogo)) {
      return false;
    }

    return true;
  }

  onError = () => {
    this.setState({ imageSrc: kolideLogo });

    return false;
  }

  unchangedSourceProp = (nextProps) => {
    const { src: nextSrcProp } = nextProps;
    const { src } = this.props;

    return src === nextSrcProp;
  }

  render () {
    const { className } = this.props;
    const { imageSrc } = this.state;
    const { onError } = this;

    return (
      <img
        alt="Organization Logo"
        className={className}
        onError={onError}
        src={imageSrc}
      />
    );
  }
}

export default OrgLogoIcon;
