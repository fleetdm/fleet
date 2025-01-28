import React, { ReactNode, useCallback, useRef } from "react";
import AceEditor from "react-ace";
import ReactAce from "react-ace/lib/ace";
import { IAceEditor } from "react-ace/lib/types";
import classnames from "classnames";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/ext-linking";
import "ace-builds/src-noconflict/ext-language_tools";
import { noop } from "lodash";
import ace, { Ace } from "ace-builds";
import {
  osqueryTableNames,
  selectedTableColumns,
} from "utilities/osquery_tables";
import {
  checkTable,
  sqlBuiltinFunctions,
  sqlDataTypes,
  sqlKeyWords,
} from "utilities/sql_tools";

import "./mode";
import "./theme";

export interface IFleetAceProps {
  focus?: boolean;
  error?: string | null;
  fontSize?: number;
  label?: string;
  name?: string;
  value?: string;
  placeholder?: string;
  readOnly?: boolean;
  maxLines?: number;
  showGutter?: boolean;
  wrapEnabled?: boolean;
  /** @deprecated use the prop `className` instead */
  wrapperClassName?: string;
  className?: string;
  helpText?: ReactNode;
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
  placeholder,
  readOnly,
  maxLines = 20,
  showGutter = true,
  wrapEnabled = false,
  wrapperClassName,
  className,
  helpText,
  style,
  onBlur,
  onLoad,
  onChange,
  handleSubmit = noop,
}: IFleetAceProps): JSX.Element => {
  const editorRef = useRef<ReactAce>(null);
  const wrapperClass = classnames(className, wrapperClassName, baseClass, {
    [`${baseClass}__wrapper--error`]: !!error,
  });

  const fixHotkeys = (editor: IAceEditor) => {
    editor.commands.removeCommand("gotoline");
    editor.commands.removeCommand("find");
  };

  const langTools = ace.require("ace/ext/language_tools");

  // Error handling within checkTableValues

  if (!readOnly) {
    // Takes SQL and returns what table(s) are being used
    const checkTableValues = checkTable(value);

    // Update completers if no sql errors or the errors include syntax near table name
    const updateCompleters =
      !checkTableValues.error ||
      checkTableValues.error
        .toString()
        .includes("Syntax error found near Identifier (FROM Clause)");

    if (updateCompleters) {
      langTools.setCompleters([]); // Reset completers as modifications are additive

      // Autocomplete sql keywords, builtin functions, and datatypes
      const sqlKeyWordsCompleter = {
        getCompletions: (
          editor: Ace.Editor,
          session: Ace.EditSession,
          pos: Ace.Point,
          prefix: string,
          callback: Ace.CompleterCallback
        ): void => {
          callback(null, [
            ...sqlKeyWords.map(
              (keyWord: string) =>
                ({
                  caption: `${keyWord}`,
                  value: keyWord.toUpperCase(),
                  meta: "keyword",
                } as Ace.Completion)
            ),
            ...sqlBuiltinFunctions.map(
              (builtInFunction: string) =>
                ({
                  caption: builtInFunction,
                  value: builtInFunction.toUpperCase(),
                  meta: "built-in function",
                } as Ace.Completion)
            ),
            ...sqlDataTypes.map(
              (dataType: string) =>
                ({
                  caption: dataType,
                  value: dataType.toUpperCase(),
                  meta: "data type",
                } as Ace.Completion)
            ),
          ]);
        },
      };

      langTools.addCompleter(sqlKeyWordsCompleter); // Add selected table columns or all columns

      const sqlTableColumns = selectedTableColumns(
        checkTableValues.tables || []
      );

      // Autocomplete table columns
      const sqlTableColumnsCompleter = {
        getCompletions: (
          editor: Ace.Editor,
          session: Ace.EditSession,
          pos: Ace.Point,
          prefix: string,
          callback: Ace.CompleterCallback
        ): void => {
          callback(
            null,
            sqlTableColumns.map(
              (column: { name: string; description: string }) =>
                ({
                  caption: column.name, // Distinct values from tables,
                  value: column.name,
                  meta: `${column.description.slice(0, 15)}... Column`,
                } as Ace.Completion)
            )
          );
        },
      };
      langTools.addCompleter(sqlTableColumnsCompleter); // Add selected table columns or all columns

      // Add all table name completers if no table name found
      const updateTableNameCompleters =
        !checkTableValues.tables?.length || !sqlTableColumns.length;

      if (updateTableNameCompleters) {
        // Autocomplete table names
        const sqlTables = osqueryTableNames;
        const sqlTablesCompleter = {
          getCompletions: (
            editor: Ace.Editor,
            session: Ace.EditSession,
            pos: Ace.Point,
            prefix: string,
            callback: Ace.CompleterCallback
          ): void => {
            callback(
              null,
              sqlTables.map(
                (table: string) =>
                  ({
                    caption: `${table}`, // Distinct values from columns,
                    value: table,
                    meta: "Table",
                    score: 1,
                  } as Ace.Completion)
              )
            );
          },
        };
        langTools.addCompleter(sqlTablesCompleter); // Add table name completers
      }
    }
  }

  const onLoadHandler = (editor: IAceEditor) => {
    fixHotkeys(editor);

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

  const renderHelpText = () => {
    if (helpText) {
      return <span className={`${baseClass}__help-text`}>{helpText}</span>;
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
        maxLines={maxLines}
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
        placeholder={placeholder}
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
      {renderHelpText()}
    </div>
  );
};

export default FleetAce;
