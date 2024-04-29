import React, { ReactNode } from "react";
import AceEditor from "react-ace";

const baseClass = "editor";

interface IEditorProps {
  focus?: boolean;
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
  maxLines?: number;
  onChange: (value: string, event?: any) => void;
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
  focus,
  value,
  defaultValue,
  wrapEnabled = false,
  name = "editor",
  maxLines = 20,
  onChange,
}: IEditorProps) => {
  const renderHelpText = () => {
    if (helpText) {
      return <div className={`${baseClass}__help-text`}>{helpText}</div>;
    }
    return null;
  };

  return (
    <div className={baseClass}>
      <AceEditor
        wrapEnabled={wrapEnabled}
        name={name}
        className={baseClass}
        fontSize={14}
        theme="fleet"
        width="100%"
        minLines={2}
        maxLines={maxLines}
        editorProps={{ $blockScrolling: Infinity }}
        value={value}
        defaultValue={defaultValue}
        tabSize={2}
        focus={focus}
        onChange={onChange}
      />
      {renderHelpText()}
    </div>
  );
};

export default Editor;
