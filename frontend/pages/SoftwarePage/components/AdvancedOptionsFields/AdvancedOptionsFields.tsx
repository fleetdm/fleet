import React, { ReactNode } from "react";
import classnames from "classnames";

import Editor from "components/Editor";
import FleetAce from "components/FleetAce";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "advanced-options-fields";

interface IAdvancedOptionsFieldsProps {
  showSchemaButton: boolean;
  installScriptHelpText: ReactNode;
  postInstallScriptHelpText: ReactNode;
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
  installScriptHelpText,
  postInstallScriptHelpText,
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
        <Icon name="info" size="small" />
        <span>Show schema</span>
      </Button>
    );
  };

  return (
    <div className={classNames}>
      <FleetAce
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
        isFormField
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
        isFormField
      />
      <Editor
        label="Uninstall script"
        focus
        wrapEnabled
        name="uninstall-script-editor"
        maxLines={20}
        onChange={onChangeUninstallScript}
        value={uninstallScript}
        helpText={uninstallScriptHelpText}
        isFormField
      />
    </div>
  );
};

export default AdvancedOptionsFields;
