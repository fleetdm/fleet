import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";
import React, { ReactNode } from "react";
import AceEditor from "react-ace";
import { IAceEditor } from "react-ace/lib/types";

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
  /** Enabled wrapping lines.
   * @default false
   */
  wrapEnabled?: boolean;
  /** A unique name for the editor.
   * @default "editor"
   */
  name?: string;
  /** Include correct styles as a form field.
   * @default false
   */
  isFormField?: boolean;
  maxLines?: number;
  className?: string;
  onChange?: (value: string, event?: any) => void;
}

/**
 * This component is a generic editor that uses the AceEditor component.
 * TODO: We should move FleetAce and YamlAce into here and deprecate importing
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
  wrapEnabled = false,
  name = "editor",
  isFormField = false,
  maxLines = 20,
  className,
  onChange,
}: IEditorProps) => {
  const classNames = classnames(baseClass, className, {
    "form-field": isFormField,
    [`${baseClass}__error`]: !!error,
  });

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
      <AceEditor
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
        onLoad={onLoadHandler}
      />
      {renderHelpText()}
    </div>
  );
};

export default Editor;
