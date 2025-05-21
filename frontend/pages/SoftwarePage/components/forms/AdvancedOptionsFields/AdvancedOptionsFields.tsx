import React, { ReactNode } from "react";
import classnames from "classnames";

import Editor from "components/Editor";
import SQLEditor from "components/SQLEditor";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "advanced-options-fields";

interface IAdvancedOptionsFieldsProps {
  showSchemaButton: boolean;
  installScriptTooltip?: string;
  installScriptHelpText: ReactNode;
  postInstallScriptHelpText: ReactNode;
  uninstallScriptTooltip?: string;
  uninstallScriptHelpText: ReactNode;
  errors: { preInstallQuery?: string; postInstallScript?: string };
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  uninstallScript?: string;
  className?: string;
  onClickShowSchema: () => void;
  onChangePreInstallQuery: (value?: string) => void;
  onChangeInstallScript: (value: string) => void;
  onChangePostInstallScript: (value?: string) => void;
  onChangeUninstallScript: (value?: string) => void;
}

const AdvancedOptionsFields = ({
  showSchemaButton,
  installScriptTooltip,
  installScriptHelpText,
  postInstallScriptHelpText,
  uninstallScriptTooltip,
  uninstallScriptHelpText,
  errors,
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
  className,
  onClickShowSchema,
  onChangePreInstallQuery,
  onChangeInstallScript,
  onChangePostInstallScript,
  onChangeUninstallScript,
}: IAdvancedOptionsFieldsProps) => {
  const classNames = classnames(baseClass, className);

  const renderLabelComponent = (): JSX.Element | null => {
    if (!showSchemaButton) {
      return null;
    }

    return (
      <Button variant="text-icon" onClick={onClickShowSchema}>
        Schema
        <Icon name="info" size="small" />
      </Button>
    );
  };

  return (
    <div className={classNames}>
      <SQLEditor
        className="form-field"
        focus
        error={errors.preInstallQuery}
        value={preInstallQuery}
        placeholder="SELECT * FROM osquery_info WHERE start_time > 1"
        label="Pre-install query"
        name="preInstallQuery"
        maxLines={10}
        onChange={onChangePreInstallQuery}
        labelActionComponent={renderLabelComponent()}
        helpText="Software will be installed only if the query returns results."
      />
      <Editor
        wrapEnabled
        maxLines={10}
        name="install-script"
        onChange={onChangeInstallScript}
        value={installScript}
        helpText={installScriptHelpText}
        label="Install script"
        labelTooltip={installScriptTooltip}
      />
      <Editor
        label="Post-install script"
        focus
        error={errors.postInstallScript}
        wrapEnabled
        name="post-install-script-editor"
        maxLines={10}
        onChange={onChangePostInstallScript}
        value={postInstallScript}
        helpText={postInstallScriptHelpText}
      />
      <Editor
        label="Uninstall script"
        labelTooltip={uninstallScriptTooltip}
        focus
        wrapEnabled
        name="uninstall-script-editor"
        maxLines={20}
        onChange={onChangeUninstallScript}
        value={uninstallScript}
        helpText={uninstallScriptHelpText}
      />
    </div>
  );
};

export default AdvancedOptionsFields;
