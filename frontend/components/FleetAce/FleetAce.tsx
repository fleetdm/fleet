import React, { useCallback } from "react";
import AceEditor from "react-ace";
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
  onChange?: (value: string, event?: any) => void;
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

  const fixHotkeys = (editor: IAceEditor) => {
    editor.commands.removeCommand("gotoline");
    editor.commands.removeCommand("find");
    onLoad && onLoad(editor);
  };

  return (
    <div className={wrapperClass}>
      {renderLabel()}
      <AceEditor
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
        ]}
      />
      {renderHint()}
    </div>
  );
};

export default FleetAce;
