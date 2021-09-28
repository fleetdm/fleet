import React, { useCallback, useRef } from "react";
import AceEditor from "react-ace";
import ReactAce from "react-ace/lib/ace";
import classnames from "classnames";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/ext-linking";
import "ace-builds/src-noconflict/ext-language_tools";
import { noop } from "lodash";

import { IAceEditor } from "react-ace/lib/types";
import "./mode";
import "./theme";

interface IFleetAceProps {
  error?: string;
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
  onLoad?: (editor: IAceEditor) => void;
  onChange?: (value: string) => void;
  handleSubmit?: () => void;
}

const baseClass = "fleet-ace";

const FleetAce = ({
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
  onLoad,
  onChange,
  handleSubmit = noop,
}: IFleetAceProps) => {
  const editorRef = useRef<ReactAce>(null);

  const renderLabel = useCallback(() => {
    const labelText = error || label;
    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !!error,
      [`${baseClass}__label--with-action`]: !!labelActionComponent,
    });

    return (
      <div className={labelClassName}>
        <p>{labelText}</p>
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

  const wrapperClass = classnames(wrapperClassName, {
    [`${baseClass}__wrapper--error`]: !!error,
  });

  const handleDelete = (deleteCommand: string) => {
    const currentText = editorRef.current?.editor.getValue();
    const selectedText = editorRef.current?.editor.getSelectedText();
    const selectedStartPosition = editorRef.current?.editor
      .getSelection()
      .getCursor();

    if (selectedText) {
      const remainingText = currentText?.replace(selectedText, "");
      if (typeof remainingText !== "undefined") {
        onChange && onChange(remainingText);
        editorRef.current?.editor.clearSelection();
      }
    } else {
      editorRef.current?.editor.execCommand(deleteCommand);
    }

    // not sure why adding zero works smh
    if (selectedStartPosition && selectedStartPosition.column) {
      const newColumn = selectedStartPosition?.column + 0;
      selectedStartPosition &&
        editorRef.current?.editor.moveCursorTo(
          selectedStartPosition.row,
          newColumn
        );
    }
  };

  const fixHotkeys = (editor: IAceEditor) => {
    editor.commands.removeCommand("gotoline");
    editor.commands.removeCommand("find");
    onLoad && onLoad(editor);
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
        onLoad={fixHotkeys}
        readOnly={readOnly}
        setOptions={{ enableLinking: true }}
        showGutter={showGutter}
        showPrintMargin={false}
        theme="fleet"
        value={value}
        width="100%"
        wrapEnabled={wrapEnabled}
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
