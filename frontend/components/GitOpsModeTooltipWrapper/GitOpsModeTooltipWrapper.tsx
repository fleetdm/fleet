import classnames from "classnames";
import { uniqueId } from "lodash";
import { ITooltipWrapper } from "components/TooltipWrapper/TooltipWrapper";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IGitOpsExceptions } from "interfaces/config";
import React, { useLayoutEffect, useMemo, useRef, useState } from "react";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";
import { getGitOpsModeTipContent } from "utilities/helpers";

interface IGitOpsModeTooltipWrapper {
  renderChildren: (disableChildren?: boolean) => React.ReactNode;
  /** Position left is used for forms partially disabled by gitops mode (e.g. settings/organization/advanced) */
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
  fixedPositionStrategy?: ITooltipWrapper["fixedPositionStrategy"];
  // When specified, the wrapper checks the exception for this entity type.
  // If the entity is excepted, children remain enabled even in GitOps mode.
  entityType?: keyof IGitOpsExceptions;
  /** Set to true when wrapping an input/dropdown or a group of them that must stretch to the
   *  full form width. By default the wrapper hugs its content so the tooltip centers on the
   *  control; inputs opt back into full width. */
  isInputField?: boolean;
}

const baseClass = "gitops-mode-tooltip-wrapper";

// The label/control row of a wrapped FormField. FormField renders the label as a
// direct-child <label> for inputs/dropdowns, and a checkbox's control row is also a
// direct-child <label>. The help text is a <span class="form-field__help-text"> and is
// intentionally never matched, so the tooltip anchors at the label rather than the
// geometric center of label + input + help text.
const FIRST_ROW_PARTS = [".form-field__label", ".form-field > label"];

const GitOpsModeTooltipWrapper = ({
  position = "top",
  tipOffset,
  renderChildren,
  fixedPositionStrategy,
  entityType,
  isInputField = false,
}: IGitOpsModeTooltipWrapper) => {
  const { gitOpsModeEnabled, repoURL } = useGitOpsMode(entityType);

  const wrapperRef = useRef<HTMLSpanElement>(null);
  const [hasSingleFieldRow, setHasSingleFieldRow] = useState(false);
  // Prefix makes this a valid CSS id selector (lodash uniqueId returns a bare number).
  const wrapperId = useMemo(() => uniqueId(`${baseClass}-`), []);

  useLayoutEffect(() => {
    // Only re-anchor when the wrapped content is a single FormField (its root element is the
    // `.form-field`, as rendered by Checkbox/InputField/Dropdown) with exactly one label
    // row. Groups (e.g. a fieldset of radios + a checkbox wrapped in a div) and non-field
    // content (buttons, icon rows) keep whole-wrapper anchoring so hovering anywhere still
    // shows the tooltip.
    const root = wrapperRef.current?.firstElementChild;
    const rows = wrapperRef.current?.querySelectorAll(
      FIRST_ROW_PARTS.join(", ")
    );
    setHasSingleFieldRow(!!root?.matches(".form-field") && rows?.length === 1);
  }, []);

  if (!gitOpsModeEnabled) {
    return <>{renderChildren()}</>;
  }

  const tipContent = (
    <div className={`${baseClass}__tooltip-content`}>
      {repoURL && getGitOpsModeTipContent(repoURL)}
    </div>
  );

  // Hug the wrapped content (buttons, checkboxes, groups) so the tooltip centers on the visible
  // control instead of the full-width form row. Inputs/dropdowns opt back into full width.
  const wrapperClass = classnames(baseClass, {
    [`${baseClass}--inputfield`]: isInputField,
    [`${baseClass}--hug`]: !isInputField,
  });

  // Anchor to the field's first row so the arrow points at the label regardless of input,
  // help-text, or tooltip-content height. Falls back to the whole wrapper otherwise,
  // preserving the previous centered, hover-anywhere behavior.
  const anchorSelect = hasSingleFieldRow
    ? FIRST_ROW_PARTS.map((part) => `#${wrapperId} ${part}`).join(", ")
    : `#${wrapperId}`;

  return (
    <span ref={wrapperRef} id={wrapperId} className={wrapperClass}>
      {renderChildren(true)}
      <ReactTooltip5
        className={`${baseClass}__tip-text`}
        anchorSelect={anchorSelect}
        place={position}
        // This wrapper always renders an arrow. A 5px offset leaves no gap between the
        // arrow and a button (e.g. "Add certificate authority"), so non-field content
        // gets extra clearance. Field tooltips anchor to the label row and read fine at 5.
        offset={tipOffset ?? (hasSingleFieldRow ? 5 : 8)}
        opacity={1}
        disableStyleInjection
        clickable
        delayShow={250}
        delayHide={250}
        positionStrategy={fixedPositionStrategy ? "fixed" : "absolute"}
      >
        {tipContent}
      </ReactTooltip5>
    </span>
  );
};

export default GitOpsModeTooltipWrapper;
