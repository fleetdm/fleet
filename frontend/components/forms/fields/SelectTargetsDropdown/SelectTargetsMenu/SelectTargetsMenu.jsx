import React from "react";
import PropTypes from "prop-types";
import classNames from "classnames";
import { filter, includes, isEqual, noop } from "lodash";

import targetInterface from "interfaces/target";
import TargetDetails from "../TargetDetails";
import { targetFilter } from "./helpers";
import TargetOption from "../TargetOption";

const baseClass = "target-list";

const SelectTargetsMenuWrapper = (
  onMoreInfoClick,
  moreInfoTarget,
  handleBackToResults
) => {
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
      const targets = filter(options, targetFilter(targetType));
      const targetsOutput = [];
      const targetTitle = targetType === "all" ? "all hosts" : targetType;

      targetsOutput.push(
        <p className={`${baseClass}__type`} key={`type-${targetType}-key`}>
          {targetTitle}
        </p>
      );

      if (targets.length === 0) {
        if (targetType === "all") {
          return false;
        }

        targetsOutput.push(
          <span
            className={`${baseClass}__not-found`}
            key={`${targetType}-notfound`}
          >
            Unable to find any matching {targetType}.
          </span>
        );

        return targetsOutput;
      }

      targetsOutput.push(
        targets.map((target, index) => {
          const { disabled: isDisabled } = target;
          const isSelected = includes(valueArray, target);
          const isFocused = isEqual(focusedOption, target);
          const className = classNames(optionClassName, {
            "Select-option": true,
            "is-selected": isSelected,
            "is-focused": isFocused,
            "is-disabled": true,
          });
          const setRef = (ref) => {
            onOptionRef(ref, isFocused);
          };

          return (
            <Option
              className={className}
              instancePrefix={instancePrefix}
              isDisabled={isDisabled}
              isFocused={isFocused}
              isSelected={isSelected}
              key={`option-${target[valueKey]}-${target.id}`}
              onFocus={onFocus}
              onSelect={noop}
              option={target}
              optionIndex={index}
              ref={setRef}
            >
              <TargetOption
                target={target}
                onSelect={onSelect}
                onMoreInfoClick={onMoreInfoClick}
              />
            </Option>
          );
        })
      );

      return targetsOutput;
    };

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__options`}>
          {renderTargets("all")}
          {renderTargets("labels")}
          {renderTargets("hosts")}
        </div>
        <TargetDetails
          target={moreInfoTarget}
          className={`${baseClass}__spotlight`}
          handleBackToResults={handleBackToResults}
        />
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
