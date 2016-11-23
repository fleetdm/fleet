import React, { Component, PropTypes } from 'react';
import { isEqual, noop } from 'lodash';
import classnames from 'classnames';

import Kolide from 'kolide';
import targetInterface from 'interfaces/target';
import { formatSelectedTargetsForApi } from './helpers';
import Input from './SelectTargetsInput';
import Menu from './SelectTargetsMenu';

class SelectTargetsDropdown extends Component {
  static propTypes = {
    onFetchTargets: PropTypes.func,
    onSelect: PropTypes.func.isRequired,
    selectedTargets: PropTypes.arrayOf(targetInterface),
  };

  static defaultProps = {
    onFetchTargets: noop,
  };

  constructor (props) {
    super(props);

    this.state = {
      isEmpty: false,
      isLoadingTargets: false,
      moreInfoTarget: null,
      query: '',
      targets: [],
    };
  }

  componentDidMount () {
    this.fetchTargets();

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { selectedTargets } = nextProps;
    const { query } = this.state;

    if (!isEqual(selectedTargets, this.props.selectedTargets)) {
      this.fetchTargets(query, selectedTargets);
    }
  }

  onInputClose = () => {
    this.setState({ moreInfoTarget: null, query: '' });

    return false;
  }

  onTargetSelectMoreInfo = (moreInfoTarget) => {
    return (evt) => {
      evt.preventDefault();

      const currentMoreInfoTarget = this.state.moreInfoTarget || {};

      if (isEqual(moreInfoTarget.display_text, currentMoreInfoTarget.display_text)) {
        return false;
      }

      const { target_type: targetType } = moreInfoTarget;

      if (targetType.toLowerCase() === 'labels') {
        return Kolide.getLabelHosts(moreInfoTarget.id)
          .then((hosts) => {
            this.setState({
              moreInfoTarget: { ...moreInfoTarget, hosts },
            });

            return false;
          });
      }

      this.setState({ moreInfoTarget });

      return false;
    };
  }

  onBackToResults = () => {
    this.setState({
      moreInfoTarget: null,
    });
  }

  fetchTargets = (query, selectedTargets = this.props.selectedTargets) => {
    const { onFetchTargets } = this.props;

    this.setState({ isLoadingTargets: true, query });

    return Kolide.getTargets(query, formatSelectedTargetsForApi(selectedTargets))
      .then((response) => {
        const {
          targets,
        } = response;

        if (targets.length === 0) {
          // We don't want the lib's default "No Results" so we fake it
          targets.push({});
          this.setState({ isEmpty: true });
        } else {
          this.setState({ isEmpty: false });
        }

        onFetchTargets(query, response);

        this.setState({ isLoadingTargets: false, targets });

        return query;
      })
      .catch((error) => {
        this.setState({ isLoadingTargets: false });

        throw error;
      });
  }

  render () {
    const { isEmpty, isLoadingTargets, moreInfoTarget, targets } = this.state;
    const { fetchTargets, onBackToResults, onInputClose, onTargetSelectMoreInfo } = this;
    const { onSelect, selectedTargets } = this.props;
    const menuRenderer = Menu(onTargetSelectMoreInfo, moreInfoTarget, onBackToResults);

    const inputClasses = classnames({
      'show-preview': moreInfoTarget,
      'is-empty': isEmpty,
    });

    return (
      <Input
        className={inputClasses}
        isLoading={isLoadingTargets}
        menuRenderer={menuRenderer}
        onClose={onInputClose}
        onTargetSelect={onSelect}
        onTargetSelectInputChange={fetchTargets}
        onInputChange={fetchTargets}
        selectedTargets={selectedTargets}
        targets={targets}
      />
    );
  }
}

export default SelectTargetsDropdown;
