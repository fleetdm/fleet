import React from 'react';
import classNames from 'classnames';
import { includes, isEqual, noop } from 'lodash';

import { shouldShowModal } from './helpers';
import TargetOption from '../TargetOption';

const SelectTargetsMenuWrapper = (onMoreInfoClick, onRemoveMoreInfoTarget, moreInfoTarget) => {
  const SelectTargetsMenu = ({
    focusedOption,
    instancePrefix,
    onFocus,
    onSelect,
    optionClassName,
    optionComponent,
    options,
    valueArray = [],
    valueKey,
    onOptionRef,
  }) => {
    const Option = optionComponent;

    return options.map((option, index) => {
      const { disabled: isDisabled } = option;
      const isSelected = includes(valueArray, option);
      const isFocused = isEqual(focusedOption, option);
      const className = classNames(optionClassName, {
        'Select-option': true,
        'is-selected': isSelected,
        'is-focused': isFocused,
        'is-disabled': true,
      });
      const setRef = (ref) => { onOptionRef(ref, isFocused); };
      const isShowModal = shouldShowModal(moreInfoTarget, option);

      return (
        <Option
          className={className}
          instancePrefix={instancePrefix}
          isDisabled={isDisabled}
          isFocused={isFocused}
          isSelected={isSelected}
          key={`option-${index}-${option[valueKey]}`}
          onFocus={onFocus}
          onSelect={noop}
          option={option}
          optionIndex={index}
          ref={setRef}
        >
          <TargetOption
            target={moreInfoTarget && isShowModal ? moreInfoTarget : option}
            onSelect={onSelect}
            onRemoveMoreInfoTarget={onRemoveMoreInfoTarget}
            onMoreInfoClick={onMoreInfoClick}
            shouldShowModal={isShowModal}
          />
        </Option>
      );
    });
  };

  return SelectTargetsMenu;
};

export default SelectTargetsMenuWrapper;
