/* eslint-disable jsx-a11y/no-noninteractive-element-to-interactive-role */
/* eslint-disable jsx-a11y/interactive-supports-focus */
import React, { useState, useContext, useEffect, KeyboardEvent } from "react";
import { IAceEditor } from "react-ace/lib/types";
import { useDebouncedCallback } from "use-debounce/lib";
import { isUndefined, size } from "lodash";
import classnames from "classnames";

import { addGravatarUrlToResource } from "fleet/helpers";
// @ts-ignore

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { IQueryPlatform } from "interfaces/query";
import { DEFAULT_POLICIES } from "utilities/constants";

import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner";
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import TooltipWrapper from "components/TooltipWrapper";
import NewPolicyModal from "../NewPolicyModal";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";
import PencilIcon from "../../../../../../assets/images/icon-pencil-14x14@2x.png";

const baseClass = "policy-form";

interface IPolicyFormProps {
  policyIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedPolicy: IPolicy | undefined;
  isStoredPolicyLoading: boolean;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IPolicyFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
  backendValidators: { [key: string]: string };
}

const validateQuerySQL = (query: string) => {
  const errors: { [key: string]: any } = {};
  const { error: queryError, valid: queryValid } = validateQuery(query);

  if (!queryValid) {
    errors.query = queryError;
  }

  const valid = !size(errors);
  return { valid, errors };
};

const PolicyForm = ({
  policyIdForEdit,
  showOpenSchemaActionText,
  storedPolicy,
  isStoredPolicyLoading,
  onCreatePolicy,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
  backendValidators,
}: IPolicyFormProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isNewPolicyModalOpen, setIsNewPolicyModalOpen] = useState<boolean>(
    false
  );
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);
  const [isEditingName, setIsEditingName] = useState<boolean>(false);
  const [isEditingDescription, setIsEditingDescription] = useState<boolean>(
    false
  );
  const [isEditingResolution, setIsEditingResolution] = useState<boolean>(
    false
  );
  const [isDarwinCompatible, setIsDarwinCompatible] = useState<boolean>(false);
  const [isWindowsCompatible, setIsWindowsCompatible] = useState<boolean>(
    false
  );
  const [isLinuxCompatible, setIsLinuxCompatible] = useState<boolean>(false);

  // Note: The PolicyContext values should always be used for any mutable policy data such as query name
  // The storedPolicy prop should only be used to access immutable metadata such as author id
  const {
    policyTeamId,
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryResolution,
    lastEditedQueryPlatform,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const {
    currentUser,
    isTeamObserver,
    isGlobalObserver,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  policyIdForEdit = policyIdForEdit || 0;

  const debounceSQL = useDebouncedCallback((sql: string) => {
    let valid = true;
    const { valid: isValidated, errors: newErrors } = validateQuerySQL(sql);
    valid = isValidated;

    setErrors({
      ...newErrors,
    });
  }, 500);

  useEffect(() => {
    debounceSQL(lastEditedQueryBody);
  }, [lastEditedQueryBody]);

  const isEditMode = !!policyIdForEdit && !isTeamObserver && !isGlobalObserver;

  const isNewTemplatePolicy =
    !policyIdForEdit &&
    DEFAULT_POLICIES.find((p) => p.name === lastEditedQueryName);

  const hasSavePermissions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  const displayOrder = [
    {
      selected: isDarwinCompatible,
      displayName: "macOS",
    },
    {
      selected: isWindowsCompatible,
      displayName: "Windows",
    },
    {
      selected: isLinuxCompatible,
      displayName: "Linux",
    },
  ];

  const onLoad = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
    });

    if (policyIdForEdit || isNewTemplatePolicy) {
      setIsWindowsCompatible(!!lastEditedQueryPlatform?.includes("windows"));
      setIsDarwinCompatible(!!lastEditedQueryPlatform?.includes("darwin"));
      setIsLinuxCompatible(!!lastEditedQueryPlatform?.includes("linux"));
    } else {
      setIsWindowsCompatible(true);
      setIsDarwinCompatible(true);
      setIsLinuxCompatible(true);
    }

    // @ts-expect-error
    // the string "linkClick" is not officially in the lib but we need it
    editor.on("linkClick", (data: EditorSession) => {
      const { type, value } = data.token;

      if (type === "osquery-token") {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const onChangePolicy = (sqlString: string) => {
    setLastEditedQueryBody(sqlString);
  };

  const onInputKeypress = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key.toLowerCase() === "enter" && !event.shiftKey) {
      event.preventDefault();
      event.currentTarget.blur();
      setIsEditingName(false);
      setIsEditingDescription(false);
      setIsEditingResolution(false);
    }
  };

  const promptSavePolicy = (forceNew = false) => (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    if (isEditMode && !lastEditedQueryName) {
      return setErrors({
        ...errors,
        name: "Policy name must be present",
      });
    }

    const selectedPlatforms = [];

    const areCheckboxesUndefined = [
      isDarwinCompatible,
      isWindowsCompatible,
      isLinuxCompatible,
    ].some((val) => isUndefined(val));

    if (!areCheckboxesUndefined) {
      isDarwinCompatible && selectedPlatforms.push("darwin");
      isWindowsCompatible && selectedPlatforms.push("windows");
      isLinuxCompatible && selectedPlatforms.push("linux");
      setLastEditedQueryPlatform(selectedPlatforms.join(",") as IQueryPlatform);
    }

    if (!isEditMode || forceNew) {
      setIsNewPolicyModalOpen(true);
    } else {
      onUpdate({
        name: lastEditedQueryName,
        description: lastEditedQueryDescription,
        query: lastEditedQueryBody,
        resolution: lastEditedQueryResolution,
        platform: lastEditedQueryPlatform,
      });
    }

    setIsEditingName(false);
    setIsEditingDescription(false);
    setIsEditingResolution(false);
  };

  const renderAuthor = (): JSX.Element | null => {
    return storedPolicy ? (
      <>
        <b>Author</b>
        <div>
          <Avatar
            user={addGravatarUrlToResource({
              email: storedPolicy.author_email,
            })}
            size="xsmall"
          />
          <span>
            {storedPolicy.author_name === currentUser?.name
              ? "You"
              : storedPolicy.author_name}
          </span>
        </div>
      </>
    ) : null;
  };

  const renderLabelComponent = (): JSX.Element | null => {
    if (!showOpenSchemaActionText) {
      return null;
    }

    return (
      <Button variant="text-icon" onClick={onOpenSchemaSidebar}>
        <>
          <img alt="" src={InfoIcon} />
          Show schema
        </>
      </Button>
    );
  };

  const policyNameClasses = classnames("policy-name-wrapper", {
    [`${baseClass}--editing`]: isEditingName,
  });

  const policyDescriptionClasses = classnames("policy-description-wrapper", {
    [`${baseClass}--editing`]: isEditingDescription,
  });

  const policyResolutionClasses = classnames("policy-resolution-wrapper", {
    [`${baseClass}--editing`]: isEditingResolution,
  });

  const renderName = () => {
    if (isEditMode) {
      return (
        <>
          <div className={policyNameClasses}>
            <AutoSizeInputField
              name="policy-name"
              placeholder="Add name here"
              value={lastEditedQueryName}
              hasError={errors && errors.name}
              inputClassName={`${baseClass}__policy-name`}
              maxLength="160"
              onChange={setLastEditedQueryName}
              onFocus={() => setIsEditingName(true)}
              onBlur={() => setIsEditingName(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingName}
            />
            <a className="edit-link" onClick={() => setIsEditingName(true)}>
              <img
                className={`edit-icon ${isEditingName && "hide"}`}
                alt="Edit name"
                src={PencilIcon}
              />
            </a>
          </div>
        </>
      );
    }

    return <h1 className={`${baseClass}__policy-name no-hover`}>New policy</h1>;
  };

  const renderDescription = () => {
    if (isEditMode) {
      return (
        <>
          <div className={policyDescriptionClasses}>
            <AutoSizeInputField
              name="policy-description"
              placeholder="Add description here."
              value={lastEditedQueryDescription}
              inputClassName={`${baseClass}__policy-description`}
              maxLength="250"
              onChange={setLastEditedQueryDescription}
              onFocus={() => setIsEditingDescription(true)}
              onBlur={() => setIsEditingDescription(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingDescription}
            />
            <a
              className="edit-link"
              onClick={() => setIsEditingDescription(true)}
            >
              <img
                className={`edit-icon ${isEditingDescription && "hide"}`}
                alt="Edit name"
                src={PencilIcon}
              />
            </a>
          </div>
        </>
      );
    }

    return null;
  };

  const renderResolution = () => {
    if (isEditMode) {
      return (
        <>
          <p className="resolve-title">
            <strong>Resolve:</strong>
          </p>
          <div className={policyResolutionClasses}>
            <AutoSizeInputField
              name="policy-resolution"
              placeholder="Add resolution here."
              value={lastEditedQueryResolution}
              inputClassName={`${baseClass}__policy-resolution`}
              maxLength="500"
              onChange={setLastEditedQueryResolution}
              onFocus={() => setIsEditingResolution(true)}
              onBlur={() => setIsEditingResolution(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingResolution}
            />
            <a
              className="edit-link"
              onClick={() => setIsEditingResolution(true)}
            >
              <img
                className={`edit-icon ${isEditingResolution && "hide"}`}
                alt="Edit name"
                src={PencilIcon}
              />
            </a>
          </div>
        </>
      );
    }

    return null;
  };

  const renderPlatformCompatibility = () => {
    let chosenPlatforms = displayOrder.filter((platform) => platform.selected);
    chosenPlatforms = chosenPlatforms.length ? chosenPlatforms : displayOrder;
    const displayPlatforms = chosenPlatforms.map(
      (platform) => platform.displayName
    );

    return (
      <span className={`${baseClass}__platform-compatibility`}>
        {isEditMode ? (
          <>
            <b>Checks on:</b>
            <span className="platforms-text">
              <TooltipWrapper tipContent="To choose new platforms, please create a new policy.">
                {displayPlatforms.join(", ")}
              </TooltipWrapper>
            </span>
          </>
        ) : (
          <>
            <b>Checks on:</b>
            <div className="platforms-select">
              <Checkbox
                value={isDarwinCompatible}
                onChange={(value: boolean) => setIsDarwinCompatible(value)}
                wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
              >
                macOS
              </Checkbox>
              <Checkbox
                value={isWindowsCompatible}
                onChange={(value: boolean) => setIsWindowsCompatible(value)}
                wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
              >
                Windows
              </Checkbox>
              <Checkbox
                value={isLinuxCompatible}
                onChange={(value: boolean) => setIsLinuxCompatible(value)}
                wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
              >
                Linux
              </Checkbox>
            </div>
          </>
        )}
      </span>
    );
  };

  const renderRunForObserver = (
    <form className={`${baseClass}__wrapper`}>
      <div className={`${baseClass}__title-bar`}>
        <div className="name-description-resolve">
          <h1 className={`${baseClass}__policy-name no-hover`}>
            {lastEditedQueryName}
          </h1>
          <p className={`${baseClass}__policy-description no-hover`}>
            {lastEditedQueryDescription}
          </p>
        </div>
        <div className="author">{renderAuthor()}</div>
      </div>
      <Button
        className={`${baseClass}__toggle-sql`}
        variant="text-link"
        onClick={() => setShowQueryEditor(!showQueryEditor)}
        disabled={false}
      >
        {showQueryEditor ? "Hide SQL" : "Show SQL"}
      </Button>
      {showQueryEditor && (
        <FleetAce
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
    </form>
  );

  const renderForGlobalAdminOrAnyMaintainer = (
    <>
      <form className={`${baseClass}__wrapper`} autoComplete="off">
        <div className={`${baseClass}__title-bar`}>
          <div className="name-description-resolve">
            {renderName()}
            {renderDescription()}
            {renderResolution()}
          </div>
          <div className="author">{isEditMode && renderAuthor()}</div>
        </div>
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={onChangePolicy}
          handleSubmit={promptSavePolicy}
        />
        {renderPlatformCompatibility()}
        {renderLiveQueryWarning()}
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-policy`}
        >
          {hasSavePermissions && (
            <div className="query-form__button-wrap--save-policy-button">
              <div
                data-tip
                data-for="save-query-button"
                data-tip-disable={!(isTeamAdmin || isTeamMaintainer)}
              >
                <Button
                  className={`${baseClass}__save`}
                  variant="brand"
                  onClick={promptSavePolicy()}
                >
                  <>Save{!isEditMode && " policy"}</>
                </Button>
              </div>
            </div>
          )}
          <Button
            className={`${baseClass}__run`}
            variant="blue-green"
            onClick={goToSelectTargets}
          >
            Run
          </Button>
        </div>
      </form>
      {isNewPolicyModalOpen && (
        <NewPolicyModal
          baseClass={baseClass}
          queryValue={lastEditedQueryBody}
          onCreatePolicy={onCreatePolicy}
          setIsNewPolicyModalOpen={setIsNewPolicyModalOpen}
          platform={lastEditedQueryPlatform}
          backendValidators={backendValidators}
        />
      )}
    </>
  );

  if (isStoredPolicyLoading) {
    return <Spinner />;
  }

  if (
    isTeamObserver ||
    isGlobalObserver ||
    (policyTeamId === 0 && !isOnGlobalTeam)
  ) {
    return renderRunForObserver;
  }

  return renderForGlobalAdminOrAnyMaintainer;
};

export default PolicyForm;
