import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Button from '../../buttons/Button';
import targetInterface from '../../../interfaces/target';
import TargetInfoModal from '../../modals/TargetInfoModal';

const classBlock = 'target-option';

class TargetOption extends Component {
  static propTypes = {
    onMoreInfoClick: PropTypes.func,
    onRemoveMoreInfoTarget: PropTypes.func,
    onSelect: PropTypes.func,
    shouldShowModal: PropTypes.bool,
    target: targetInterface.isRequired,
  };

  handleSelect = (evt) => {
    const { onSelect, target } = this.props;
    return onSelect(target, evt);
  }

  handleSelectFromModal = (evt) => {
    const { handleSelect } = this;
    const { onRemoveMoreInfoTarget } = this.props;

    handleSelect(evt);
    onRemoveMoreInfoTarget();
  }

  hostPlatformIconClass = () => {
    const { platform } = this.props.target;

    return platform === 'darwin' ? 'kolidecon-apple' : `kolidecon-${platform}`;
  }

  targetIconClass = () => {
    const { label, target_type: targetType } = this.props.target;

    if (label.toLowerCase() === 'all hosts') {
      return 'kolidecon-all-hosts';
    }

    if (targetType === 'hosts') {
      return 'kolidecon-single-host';
    }

    return 'kolidecon-label';
  }

  renderTargetDetail = () => {
    const { target } = this.props;
    const { count, ip, target_type: targetType } = target;

    if (targetType === 'hosts') {
      return <span className={`${classBlock}__ip`}>{ip}</span>;
    }

    return <span className={`${classBlock}__count`}>{count} hosts</span>;
  }

  renderTargetInfoModal = () => {
    const { onRemoveMoreInfoTarget, shouldShowModal, target } = this.props;

    if (!shouldShowModal) return false;

    const { handleSelectFromModal } = this;

    return (
      <TargetInfoModal
        className={`${classBlock}__modal-wrapper`}
        onAdd={handleSelectFromModal}
        onExit={onRemoveMoreInfoTarget}
        target={target}
      />
    );
  }

  render () {
    const { onMoreInfoClick, target } = this.props;
    const { label, target_type: targetType } = target;
    const {
      handleSelect,
      hostPlatformIconClass,
      targetIconClass,
      renderTargetDetail,
      renderTargetInfoModal,
    } = this;
    const wrapperClassName = classnames(`${classBlock}__wrapper`, {
      '--is-label': targetType === 'labels',
      '--is-host': targetType === 'hosts',
    });

    return (
      <div className={wrapperClassName}>
        <i className={`${targetIconClass()} ${classBlock}__target-icon`} />
        {targetType === 'hosts' && <i className={`${classBlock}__icon ${hostPlatformIconClass()}`} />}
        <span className={`${classBlock}__label-label`}>{label}</span>
        <span className={`${classBlock}__delimeter`}>&bull;</span>
        {renderTargetDetail()}
        <Button className={`${classBlock}__btn`} text="ADD" onClick={handleSelect} />
        <Button className={`${classBlock}__more-info`} onClick={onMoreInfoClick(target)} text="more info" variant="unstyled" />
        {renderTargetInfoModal()}
      </div>
    );
  }
}

export default TargetOption;
