import classnames from "classnames";
import React, { ReactNode, useEffect, useState } from "react";
import AceEditor from "react-ace";

import "ace-builds/src-noconflict/mode-sh";
import "ace-builds/src-noconflict/mode-powershell";
import "ace-builds/src-noconflict/mode-python";
import "ace-builds/src-noconflict/mode-xml";
import "ace-builds/src-noconflict/mode-json";

import { Ace } from "ace-builds";

import CopyButton from "components/buttons/CopyButton";
import TooltipWrapper from "components/TooltipWrapper";
import { releaseStuckSelectionOnScroll } from "utilities/ace_editor";

import "utilities/ace_theme";

const baseClass = "editor";

export type EditorMode =
  | "sh"
  | "powershell"
  | "python"
  | "xml"
  | "json"
  | "text";

export interface IEditorProps {
  focus?: boolean;
  label?: string;
  /** Accessible name for the text input. Defaults to `label`. */
  ariaLabel?: string;
  labelTooltip?: string | JSX.Element;
  error?: string | null;
  readOnly?: boolean;
  /**
   * Help text to display below the editor.
   */
  helpText?: ReactNode;
  /** Sets the value of the input. Use this if you'd like the editor
   * to be a controlled component */
  value?: string;
  /** Sets the default value of the input. Use this if you'd like the editor
   * to be an uncontrolled component */
  defaultValue?: string;
  /** Enable copying the value of the editor.
   * @default false
   */
  enableCopy?: boolean;
  /** Where the copy button sits when `enableCopy` is on.
   * - `"overlay"` (default): subdued copy icon absolute-positioned over the
   *   top-right of the editor, matching InputField's `enableCopy`.
   * - `"label"`: outlined copy button in the label row, aligned opposite the
   *   label text. For readonly display-and-copy blocks where overlaying the
   *   content isn't desired.
   * @default "overlay"
   */
  copyButtonLocation?: "overlay" | "label";
  /** Enabled wrapping lines.
   * @default false
   */
  wrapEnabled?: boolean;
  /** A unique name for the editor.
   * @default "editor"
   */
  name?: string;
  /** The syntax highlighting mode to use.
   */
  mode?: EditorMode;
  /** Include correct styles as a form field.
   * @default true
   */
  isFormField?: boolean;
  maxLines?: number;
  /** @default 2 */
  minLines?: number;
  /** @default true */
  showPrintMargin?: boolean;
  /** Shown while the editor is empty. */
  placeholder?: string;
  className?: string;
  onChange?: (value: string, event?: Ace.Delta) => void;
  onBlur?: () => void;
  onFocus?: () => void;
  /** Called after the Ace editor mounts with the editor instance. */
  onLoad?: (editor: Ace.Editor) => void;
}

/**
 * This component is a generic editor that uses the AceEditor component.
 * TODO: We should move SQLEditor and YamlAce into here and deprecate importing
 * them directly. This component should be used for all editor components and
 * be configurable from the props. We should look into dynmaic imports for
 * this.
 */
const Editor = ({
  helpText,
  label,
  ariaLabel,
  labelTooltip,
  error,
  focus,
  value,
  defaultValue,
  readOnly = false,
  enableCopy = false,
  copyButtonLocation = "overlay",
  wrapEnabled = false,
  name = "editor",
  mode = "text",
  isFormField = true,
  maxLines = 20,
  minLines = 2,
  showPrintMargin = true,
  placeholder,
  className,
  onChange,
  onBlur,
  onFocus,
  onLoad: onLoadProp,
}: IEditorProps) => {
  const classNames = classnames(baseClass, className, {
    "form-field": isFormField,
    [`${baseClass}__error`]: !!error,
  });

  // Track the editor's displayed content so copy works for uncontrolled
  // editors (defaultValue + edits) as well as controlled ones.
  const [displayedValue, setDisplayedValue] = useState(
    value ?? defaultValue ?? ""
  );
  useEffect(() => {
    if (value !== undefined) {
      setDisplayedValue(value);
    }
  }, [value]);

  const handleChange = (newValue: string, event?: Ace.Delta) => {
    setDisplayedValue(newValue);
    onChange?.(newValue, event);
  };

  const renderCopyButton = () => {
    return (
      <div className={`${baseClass}__copy-wrapper`}>
        <CopyButton copyText={displayedValue} variant="subdued" />
      </div>
    );
  };

  const renderLabelCopyButton = () => {
    return (
      <CopyButton
        copyText={displayedValue}
        variant="secondary"
        ariaLabel={ariaLabel ? `Copy ${ariaLabel}` : "Copy"}
        disabled={!displayedValue}
      />
    );
  };

  const onLoadHandler = (editor: Ace.Editor) => {
    // Lose focus using the Escape key so you can Tab forward (or Shift+Tab backwards) through app
    editor.commands.addCommand({
      name: "escapeToBlur",
      bindKey: { win: "Esc", mac: "Esc" },
      exec: (aceEditor) => {
        aceEditor.blur(); // Lose focus from the editor
        return true;
      },
      readOnly: true,
    });

    // Prevent scrolling from selecting text after a stationary click (#48490).
    releaseStuckSelectionOnScroll(editor);

    onLoadProp?.(editor);
  };

  const renderLabel = () => {
    const labelText = error || label;
    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !!error,
    });
    const showInlineCopyButton =
      enableCopy && copyButtonLocation === "label" && !error;

    if (!labelText && !showInlineCopyButton) {
      return null;
    }

    const labelNode = labelTooltip ? (
      <TooltipWrapper
        className={labelClassName}
        tipContent={labelTooltip}
        position="top-start"
      >
        {labelText as string}
      </TooltipWrapper>
    ) : (
      <div className={labelClassName}>{labelText}</div>
    );

    if (!showInlineCopyButton) {
      return labelNode;
    }

    return (
      <div className={`${baseClass}__label-row`}>
        {labelNode}
        {renderLabelCopyButton()}
      </div>
    );
  };

  const renderHelpText = () => {
    if (helpText) {
      return <div className={`${baseClass}__help-text`}>{helpText}</div>;
    }
    return null;
  };

  return (
    <div className={classNames}>
      {renderLabel()}
      {enableCopy && copyButtonLocation === "overlay" && renderCopyButton()}
      <AceEditor
        mode={mode}
        wrapEnabled={wrapEnabled}
        name={name}
        className={baseClass}
        fontSize={14}
        theme="fleet"
        width="100%"
        readOnly={readOnly}
        minLines={minLines}
        maxLines={maxLines}
        showPrintMargin={showPrintMargin}
        placeholder={placeholder}
        editorProps={{ $blockScrolling: Infinity }}
        setOptions={{
          textInputAriaLabel: ariaLabel ?? label,
          // Fleet doesn't serve Ace's worker-*.js scripts and validates content
          // itself, so the JSON/XML syntax workers would only 404.
          useWorker: false,
        }}
        value={value}
        defaultValue={defaultValue}
        tabSize={2}
        focus={focus}
        onChange={handleChange}
        onBlur={onBlur}
        onFocus={onFocus}
        onLoad={onLoadHandler}
      />
      {renderHelpText()}
    </div>
  );
};

export default Editor;
