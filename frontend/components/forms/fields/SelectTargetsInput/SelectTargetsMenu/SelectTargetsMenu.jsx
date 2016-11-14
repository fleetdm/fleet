import React, { PropTypes } from 'react';
import classNames from 'classnames';
import { filter, includes, isEqual, noop } from 'lodash';

import targetInterface from 'interfaces/target';
import SelectTargets from 'components/forms/fields/SelectTargetsInput';


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
    const renderTargets = (targetType) => {
      const targets = filter(options, { target_type: targetType });

      return targets.map((target, index) => {
        const { disabled: isDisabled } = target;
        const isSelected = includes(valueArray, target);
        const isFocused = isEqual(focusedOption, target);
        const className = classNames(optionClassName, {
          'Select-option': true,
          'is-selected': isSelected,
          'is-focused': isFocused,
          'is-disabled': true,
        });
        const setRef = (ref) => { onOptionRef(ref, isFocused); };

        return (
          <Option
            className={className}
            instancePrefix={instancePrefix}
            isDisabled={isDisabled}
            isFocused={isFocused}
            isSelected={isSelected}
            key={`option-${index}-${target[valueKey]}`}
            onFocus={onFocus}
            onSelect={noop}
            option={target}
            optionIndex={index}
            ref={setRef}
          >
            <SelectTargets.Option
              target={target}
              onSelect={onSelect}
              onRemoveMoreInfoTarget={onRemoveMoreInfoTarget}
              onMoreInfoClick={onMoreInfoClick}
            />
          </Option>
        );
      });
    };

    return (
      <div>
        <div>
          <div>hosts</div>
          {renderTargets('hosts')}
          <div>labels</div>
          {renderTargets('labels')}
        </div>
        <SelectTargets.Details target={moreInfoTarget} />
      </div>
    );
  };

  SelectTargetsMenu.propTypes = {
    focusedOption: targetInterface,
    instancePrefix: PropTypes.string,
    onFocus: PropTypes.func,
    onSelect: PropTypes.func,
    optionClassName: PropTypes.string,
    optionComponent: PropTypes.node,
    options: PropTypes.arrayOf(targetInterface),
    valueArray: PropTypes.arrayOf(targetInterface),
    valueKey: PropTypes.string,
    onOptionRef: PropTypes.func,
  };

  return SelectTargetsMenu;
};

export default SelectTargetsMenuWrapper;
