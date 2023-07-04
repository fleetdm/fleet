import React, { useCallback, useRef } from "react";
import AceEditor from "react-ace";
import ReactAce from "react-ace/lib/ace";
import { IAceEditor } from "react-ace/lib/types";
import classnames from "classnames";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/ext-linking";
import "ace-builds/src-noconflict/ext-language_tools";
import { noop } from "lodash";

import "./mode";
import "./theme";

export interface IFleetAceProps {
  focus?: boolean;
  error?: string | null;
  fontSize?: number;
  label?: string;
  name?: string;
  value?: string;
  readOnly?: boolean;
  showGutter?: boolean;
  wrapEnabled?: boolean;
  wrapperClassName?: string;
  hint?: string;
  labelActionComponent?: React.ReactNode;
  style?: React.CSSProperties;
  onBlur?: (editor?: IAceEditor) => void;
  onLoad?: (editor: IAceEditor) => void;
  onChange?: (value: string) => void;
  handleSubmit?: () => void;
}

const baseClass = "fleet-ace";

const FleetAce = ({
  focus,
  error,
  fontSize = 14,
  label,
  labelActionComponent,
  name = "query-editor",
  value,
  readOnly,
  showGutter = true,
  wrapEnabled = false,
  wrapperClassName,
  hint,
  style,
  onBlur,
  onLoad,
  onChange,
  handleSubmit = noop,
}: IFleetAceProps): JSX.Element => {
  const editorRef = useRef<ReactAce>(null);
  const wrapperClass = classnames(wrapperClassName, baseClass, {
    [`${baseClass}__wrapper--error`]: !!error,
  });

  const fixHotkeys = (editor: IAceEditor) => {
    editor.commands.removeCommand("gotoline");
    editor.commands.removeCommand("find");
  };

  const onLoadHandler = (editor: IAceEditor) => {
    fixHotkeys(editor);
    onLoad && onLoad(editor);
  };

  const onBlurHandler = (event: any, editor?: IAceEditor): void => {
    onBlur && onBlur(editor);
  };

  const handleDelete = (deleteCommand: string) => {
    const selectedText = editorRef.current?.editor.getSelectedText();

    if (selectedText) {
      editorRef.current?.editor.removeWordLeft();
    } else {
      editorRef.current?.editor.execCommand(deleteCommand);
    }
  };

  const renderLabel = useCallback(() => {
    const labelText = error || label;
    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !!error,
      [`${baseClass}__label--with-action`]: !!labelActionComponent,
    });

    if (!label) {
      return <></>;
    }

    return (
      <div className={labelClassName}>
        {labelText}
        {labelActionComponent}
      </div>
    );
  }, [error, label, labelActionComponent]);

  const renderHint = () => {
    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  return (
    <div className={wrapperClass}>
      {renderLabel()}
      <AceEditor
        ref={editorRef}
        enableBasicAutocompletion
        enableLiveAutocompletion
        editorProps={{ $blockScrolling: Infinity }}
        fontSize={fontSize}
        mode="fleet"
        minLines={2}
        maxLines={20}
        name={name}
        onChange={onChange}
        onBlur={onBlurHandler}
        onLoad={onLoadHandler}
        readOnly={readOnly}
        setOptions={{ enableLinking: true }}
        showGutter={showGutter}
        showPrintMargin={false}
        theme="fleet"
        value={value}
        width="100%"
        wrapEnabled={wrapEnabled}
        style={style}
        focus={focus}
        commands={[
          {
            name: "commandName",
            bindKey: { win: "Ctrl-Enter", mac: "Ctrl-Enter" },
            exec: handleSubmit,
          },
          {
            name: "deleteSelection",
            bindKey: { win: "Delete", mac: "Delete" },
            exec: () => handleDelete("del"),
          },
          {
            name: "backspaceSelection",
            bindKey: { win: "Backspace", mac: "Backspace" },
            exec: () => handleDelete("backspace"),
          },
        ]}
      />
      {renderHint()}
    </div>
  );
};

export default FleetAce;
