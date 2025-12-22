import React, { MouseEvent, ReactNode, useState, useCallback } from "react";

import classnames from "classnames";
import AceEditor from "react-ace";
import "ace-builds/src-noconflict/mode-sh";
import "ace-builds/src-noconflict/mode-powershell";
import { IAceEditor } from "react-ace/lib/types";

import { stringToClipboard } from "utilities/copy_text";

import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "editor";

interface IEditorProps {
  focus?: boolean;
  label?: string;
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
  mode?: string;
  /** Include correct styles as a form field.
   * @default true
   */
  isFormField?: boolean;
  maxLines?: number;
  className?: string;
  onChange?: (value: string, event?: any) => void;
  onBlur?: () => void;
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
  labelTooltip,
  error,
  focus,
  value,
  defaultValue,
  readOnly = false,
  enableCopy = false,
  wrapEnabled = false,
  name = "editor",
  mode,
  isFormField = true,
  maxLines = 20,
  className,
  onChange,
  onBlur,
}: IEditorProps) => {
  const classNames = classnames(baseClass, className, {
    "form-field": isFormField,
    [`${baseClass}__error`]: !!error,
  });

  const [showCopiedMessage, setShowCopiedMessage] = useState(false);

  const onClickCopy = useCallback(
    (e: MouseEvent) => {
      e.preventDefault();
      stringToClipboard(value).then(() => {
        setShowCopiedMessage(true);
        setTimeout(() => {
          setShowCopiedMessage(false);
        }, 2000);
      });
    },
    [value]
  );

  const renderCopyButton = () => {
    const copyButtonValue = <Icon name="copy" />;
    const wrapperClasses = classnames(`${baseClass}__copy-wrapper`);

    const copiedConfirmationClasses = classnames(
      `${baseClass}__copied-confirmation`
    );

    return (
      <div className={wrapperClasses}>
        {showCopiedMessage && (
          <span className={copiedConfirmationClasses}>Copied!</span>
        )}
        <Button variant={"icon"} onClick={onClickCopy} iconStroke>
          {copyButtonValue}
        </Button>
      </div>
    );
  };

  const onLoadHandler = (editor: IAceEditor) => {
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
  };

  const renderLabel = () => {
    const labelText = error || label;
    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !!error,
    });

    if (!labelText) {
      return null;
    }

    if (labelTooltip) {
      return (
        <TooltipWrapper
          className={labelClassName}
          tipContent={labelTooltip}
          position="top-start"
        >
          {labelText}
        </TooltipWrapper>
      );
    }

    return <div className={labelClassName}>{labelText}</div>;
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
      {enableCopy && renderCopyButton()}
      <AceEditor
        mode={mode}
        wrapEnabled={wrapEnabled}
        name={name}
        className={baseClass}
        fontSize={14}
        theme="fleet"
        width="100%"
        readOnly={readOnly}
        minLines={2}
        maxLines={maxLines}
        editorProps={{ $blockScrolling: Infinity }}
        value={value}
        defaultValue={defaultValue}
        tabSize={2}
        focus={focus}
        onChange={onChange}
        onBlur={onBlur}
        onLoad={onLoadHandler}
      />
      {renderHelpText()}
    </div>
  );
};

export default Editor;
