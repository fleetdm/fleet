/* eslint-disable jsx-a11y/no-noninteractive-element-to-interactive-role */
/* eslint-disable jsx-a11y/interactive-supports-focus */
import React, { useState, useContext } from "react";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { isUndefined } from "lodash";

import { addGravatarUrlToResource } from "fleet/helpers";
// @ts-ignore

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { IQueryPlatform } from "interfaces/query";

import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import NewPolicyModal from "../NewPolicyModal";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";
import QuestionIcon from "../../../../../../assets/images/icon-question-16x16@2x.png";
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
}

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
}: IPolicyFormProps): JSX.Element => {
  const isEditMode = !!policyIdForEdit;
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
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

    setIsWindowsCompatible(!!lastEditedQueryPlatform?.includes("windows"));
    setIsDarwinCompatible(!!lastEditedQueryPlatform?.includes("darwin"));
    setIsLinuxCompatible(!!lastEditedQueryPlatform?.includes("linux"));

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

  const renderName = () => {
    if (isEditMode) {
      if (isEditingName) {
        return (
          <InputField
            id="policy-name"
            type="textarea"
            name="policy-name"
            error={errors.name}
            value={lastEditedQueryName}
            placeholder="Add name here"
            inputClassName={`${baseClass}__policy-name`}
            onChange={setLastEditedQueryName}
            inputOptions={{
              autoFocus: true,
              onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
                // sets cursor to end of inputfield
                const val = e.target.value;
                e.target.value = "";
                e.target.value = val;
              },
            }}
          />
        );
      }

      return (
        <h1
          role="button"
          className={`${baseClass}__policy-name`}
          onClick={() => setIsEditingName(true)}
        >
          {lastEditedQueryName}
          <img alt="Edit name" src={PencilIcon} />
        </h1>
      );
    }

    return <h1 className={`${baseClass}__policy-name no-hover`}>New policy</h1>;
  };

  const renderDescription = () => {
    if (isEditMode) {
      if (isEditingDescription) {
        return (
          <InputField
            id="policy-description"
            type="textarea"
            name="policy-description"
            value={lastEditedQueryDescription}
            placeholder="Add description here."
            inputClassName={`${baseClass}__policy-description`}
            onChange={setLastEditedQueryDescription}
            inputOptions={{
              autoFocus: true,
              onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
                // sets cursor to end of inputfield
                const val = e.target.value;
                e.target.value = "";
                e.target.value = val;
              },
            }}
          />
        );
      }

      return (
        <span
          role="button"
          className={`${baseClass}__policy-description`}
          onClick={() => setIsEditingDescription(true)}
        >
          {lastEditedQueryDescription || "Add description here."}
          <img alt="Edit description" src={PencilIcon} />
        </span>
      );
    }

    return null;
  };

  const renderResolution = () => {
    if (isEditMode) {
      if (isEditingResolution) {
        return (
          <div className={`${baseClass}__policy-resolve`}>
            {" "}
            <b>Resolve:</b> <br />
            <InputField
              id="policy-resolution"
              type="textarea"
              name="policy-resolution"
              value={lastEditedQueryResolution}
              placeholder="Add resolution here."
              inputClassName={`${baseClass}__policy-resolution`}
              onChange={setLastEditedQueryResolution}
              inputOptions={{
                autoFocus: true,
                onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
                  // sets cursor to end of inputfield
                  const val = e.target.value;
                  e.target.value = "";
                  e.target.value = val;
                },
              }}
            />
          </div>
        );
      }

      return (
        <>
          <div className="resolve-text-wrapper">
            <b>Resolve:</b>{" "}
            <span
              role="button"
              className={`${baseClass}__policy-resolution`}
              onClick={() => setIsEditingResolution(true)}
            >
              <img alt="Edit resolution" src={PencilIcon} />
            </span>
            <br />
            <span
              role="button"
              className={`${baseClass}__policy-resolution`}
              onClick={() => setIsEditingResolution(true)}
            >
              {lastEditedQueryResolution || "Add resolution here."}
            </span>
          </div>
        </>
      );
    }

    return null;
  };

  const renderPlatformCompatibility = () => {
    const displayPlatforms = displayOrder
      .filter((platform) => platform.selected)
      .map((platform) => {
        return platform.displayName;
      });

    return (
      <span className={`${baseClass}__platform-compatibility`}>
        {isEditMode ? (
          <>
            <b>Checks on:</b>
            <span className="platforms-text">
              {displayPlatforms.join(", ")}
            </span>
            <span className={`tooltip`}>
              <span
                className={`tooltip__tooltip-icon`}
                data-tip
                data-for="query-compatibility-tooltip"
                data-tip-disable={false}
              >
                <img alt="question icon" src={QuestionIcon} />
              </span>
              <ReactTooltip
                place="bottom"
                type="dark"
                effect="solid"
                backgroundColor="#3e4771"
                id="query-compatibility-tooltip"
                data-html
              >
                <span className={`tooltip__tooltip-text`}>
                  To choose new platforms,
                  <br />
                  please create a new policy.
                </span>
              </ReactTooltip>
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
              </div>{" "}
              <ReactTooltip
                className={`save-policy-button-tooltip`}
                place="bottom"
                type="dark"
                effect="solid"
                backgroundColor="#3e4771"
                id="save-query-button"
                data-html
              >
                <div
                  className={`tooltip`}
                  style={{ width: "152px", textAlign: "center" }}
                >
                  You can only save changes to a query if you are the author.
                </div>
              </ReactTooltip>
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
