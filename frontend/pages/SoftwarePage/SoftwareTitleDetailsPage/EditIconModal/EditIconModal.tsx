import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { IAppStoreApp, ISoftwarePackage } from "interfaces/software";
import { IInputFieldParseTarget } from "interfaces/form_field";

import { NotificationContext } from "context/notification";
import { INotification } from "interfaces/notification";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";
import mdmAppleAPI from "services/entities/mdm_apple";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import FileUploader from "components/FileUploader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Card from "components/Card";
import Button from "components/buttons/Button";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TableCount from "components/TableContainer/TableCount";
import CardHeader from "components/CardHeader";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Spinner from "components/Spinner";

import { isSafeImagePreviewUrl } from "pages/SoftwarePage/helpers";
import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary/SoftwareDetailsSummary";
import { SELF_SERVICE_SUBHEADER } from "pages/hosts/details/cards/Software/SelfService/SelfService";

import { TitleVersionsLastUpdatedInfo } from "../SoftwareSummaryCard/TitleVersionsTable/TitleVersionsTable";
import PreviewSelfServiceIcon from "../../../../../assets/images/preview-self-service-icon.png";

const baseClass = "edit-icon-modal";

const ACCEPTED_EXTENSIONS = ".png";
const MIN_DIMENSION = 120;
const MAX_DIMENSION = 1024;
const MAX_FILE_SIZE = 100 * 1024; // 100kb in bytes
const UPLOAD_MESSAGE = `The icon must be a PNG file and square, with dimensions ranging from ${MIN_DIMENSION}x${MIN_DIMENSION} px to ${MAX_DIMENSION}x${MAX_DIMENSION} px.`;
const DEFAULT_ERROR_MESSAGE = "Couldn't edit. Please try again.";

const getFilenameFromContentDisposition = (header: string | null) => {
  if (!header) return null;
  // Try to match extended encoding (RFC 5987) first
  const matchExtended = header.match(/filename\*\s*=\s*([^;]+)/);
  if (matchExtended) {
    // RFC 5987: filename*=UTF-8''something.png
    const value = matchExtended[1].trim().replace(/^UTF-8''/, "");
    return decodeURIComponent(value);
  }
  // Then standard quoted (or unquoted) filename param
  const matchStandard = header.match(/filename\s*=\s*["']?([^"';]+)["']?/);
  return matchStandard ? matchStandard[1] : null;
};

const makeFileDetails = (
  file: File,
  dimensions: number | null
): IFileDetails => ({
  name: file.name,
  description: `Software icon • ${dimensions || "?"}x${dimensions || "?"} px`,
});

interface IFormData {
  icon: File;
  display_name?: string;
}

export interface ISoftwareDisplayNameFormData {
  displayName?: string; // Edit Display name is in the edit icon modal
}

interface IFileDetails {
  name: string;
  description: string;
}

/**
 * Icon preview state management
 *   - "apiCustom": Icon fetched directly via API, for custom uploads.
 *   - "customUpload": User-selected custom icon (not yet saved to backend).
 *   - "fallback": VPP app default or generic fallback icon.
 */
type IconStatus = "customUpload" | "apiCustom" | "fallback";

/**
 * IconState keys:
 * previewUrl:     // A blob URL for the image that is currently previewed. Used as the <img src> for preview tabs.
 * formData:       // Holds the current icon File being edited/created that will be uploaded to the API.
 * dimensions:     // The pixel width/height (square) of the current icon. Used for validation and file details.
 * fileDetails:    // { name, description } for current icon file. Used for display in the FileUploader details.
 * status:         // What icon is being shown in the UI: "apiCustom": current API-fetched custom icon, "customUpload": icon chosen by FileUploader, "fallback": fallback/default icon if no custom icon
 */
interface IconState {
  previewUrl: string | null;
  formData: IFormData | null;
  dimensions: number | null;
  fileDetails: IFileDetails | null;
  status: IconStatus;
}

// Encapsulate all icon-related UI and API state here
const defaultIconState: IconState = {
  previewUrl: null,
  formData: null,
  dimensions: null,
  fileDetails: null,
  status: "apiCustom",
};

interface IEditIconModalProps {
  softwareId: number;
  teamIdForApi: number;
  software: ISoftwarePackage | IAppStoreApp;
  onExit: () => void;
  refetchSoftwareTitle: () => void;
  /** Timestamp used to force UI and cache updates after an icon change, since API will return the same URL. */
  iconUploadedAt: string;
  /** Updates the icon upload timestamp, triggering UI refetches to ensure a new custom icon appears called after successful icon update. */
  setIconUploadedAt: (timestamp: string) => void;
  installerType: "package" | "vpp";
  previewInfo: {
    type?: string;
    versions?: number;
    source?: string;
    currentIconUrl: string | null;
    /** Name used in preview UI but also for FMA default icon matching */
    name: string;
    /** Default title name used to check if name has been modified */
    titleName: string;
    countsUpdatedAt?: string;
  };
}

const EditIconModal = ({
  softwareId,
  teamIdForApi,
  software,
  onExit,
  refetchSoftwareTitle,
  iconUploadedAt,
  setIconUploadedAt,
  installerType,
  previewInfo,
}: IEditIconModalProps) => {
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);

  const isSoftwarePackage = installerType === "package";

  // Fetch current custom icon from API if applicable
  const shouldFetchCustomIcon =
    !!previewInfo.currentIconUrl &&
    previewInfo.currentIconUrl.startsWith("/api/");

  // Unmodified names default to empty Display name field
  const hasNameBeenModified = previewInfo.titleName !== previewInfo.name;
  const defaultName = hasNameBeenModified ? previewInfo.name : "";

  // Encapsulates software name and icon preview/upload/edit state
  const [displayName, setDisplayName] = useState(defaultName);
  const [iconState, setIconState] = useState<IconState>(defaultIconState);
  const [previewTabIndex, setPreviewTabIndex] = useState(0);
  const [isUpdatingSoftwareInfo, setIsUpdatingSoftwareInfo] = useState(false);
  /** Shows loading spinner only if a custom icon and its information is loading from API */
  const [isFirstLoadWithCustomIcon, setIsFirstLoadWithCustomIcon] = useState(
    shouldFetchCustomIcon
  );

  const originalIsApiCustom =
    !!previewInfo.currentIconUrl &&
    previewInfo.currentIconUrl.startsWith("/api/");
  const originalIsVpp =
    !!previewInfo.currentIconUrl &&
    !previewInfo.currentIconUrl.startsWith("/api/");
  const isCustomUpload = iconState.status === "customUpload";
  const isRemovedCustom =
    originalIsApiCustom &&
    iconState.status === "fallback" &&
    !iconState.formData;
  const canSaveIcon = isCustomUpload || isRemovedCustom;

  // Sets state after fetching current API custom icon
  const setCurrentApiCustomIcon = (
    file: File,
    width: number,
    previewUrl: string
  ) =>
    setIconState({
      previewUrl,
      formData: { icon: file, display_name: displayName },
      dimensions: width,
      fileDetails: makeFileDetails(file, width),
      status: "apiCustom",
    });

  // Sets state after a successful new custom file upload
  const setCustomUpload = (file: File, width: number, previewUrl: string) =>
    setIconState({
      previewUrl,
      formData: { icon: file },
      dimensions: width,
      fileDetails: makeFileDetails(file, width),
      status: "customUpload",
    });

  // Reset state to fallback/default icon when a current or new custom icon is removed
  const resetIconState = () => {
    // Default to VPP icon if available, otherwise fall back to default icon
    const defaultPreviewUrl =
      previewInfo.currentIconUrl &&
      !previewInfo.currentIconUrl.startsWith("/api/")
        ? previewInfo.currentIconUrl
        : null;

    setIconState({
      previewUrl: defaultPreviewUrl,
      formData: null,
      dimensions: null,
      fileDetails: null,
      status: "fallback",
    });
  };

  const { data: customIconData } = useQuery(
    ["softwareIcon", softwareId, teamIdForApi, iconUploadedAt],
    () => softwareAPI.getSoftwareIcon(softwareId, teamIdForApi),
    {
      enabled: shouldFetchCustomIcon,
      retry: false,
      select: (response) =>
        response
          ? {
              blob: response.data,
              filename: getFilenameFromContentDisposition(
                response.headers["content-disposition"]
              ),
              url: URL.createObjectURL(response.data),
            }
          : "",
    }
  );

  const onExitEditIconModal = () => {
    resetIconState(); // Ensure cached state is cleared
    onExit();
  };

  const onInputChange = ({ value }: IInputFieldParseTarget) => {
    console.log("displayName", displayName);
    setDisplayName((value as string) || "");
    // If you want live update in the formData:
    setIconState((prev) =>
      prev.formData
        ? {
            ...prev,
            formData: { ...prev.formData, display_name: value as string },
          }
        : prev
    );
  };

  console.log("displayName", displayName);

  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      // Enforce filesize limit
      if (file.size > MAX_FILE_SIZE) {
        renderFlash("error", "Couldn't edit. Icon must be 100KB or less.");
        return;
      }

      // Enforce PNG MIME type, even though FileUploader also enforces by extension
      if (file.type !== "image/png") {
        renderFlash("error", "Couldn't edit. Must be a PNG file.");
        return;
      }

      const reader = new FileReader();
      reader.onload = (e: ProgressEvent<FileReader>) => {
        const img = new Image();
        img.onload = () => {
          const { width, height } = img;
          if (
            width !== height ||
            width < MIN_DIMENSION ||
            width > MAX_DIMENSION
          ) {
            renderFlash(
              "error",
              `Couldn't edit. Icon must be square, between ${MIN_DIMENSION}x${MIN_DIMENSION}px and ${MAX_DIMENSION}x${MAX_DIMENSION}px.`
            );
            return;
          }
          const previewUrl = URL.createObjectURL(file);
          setCustomUpload(file, width, previewUrl);
        };
        if (e.target && typeof e.target.result === "string") {
          img.src = e.target.result;
        } else {
          renderFlash("error", "FileReader result was not a string.");
        }
      };
      reader.readAsDataURL(file);
    }
  };

  const onDeleteFile = () => resetIconState();

  const onTabChange = (index: number) => setPreviewTabIndex(index);

  // If there's currently a custom API icon and no new upload has happened yet,
  // populate icon info from API-fetched custom icon
  // useQuery does not handle dimension extraction, so this is required for updating
  // state with image details after loading the icon blob in the browser
  useEffect(() => {
    // Handle API custom icon blob conversion and initialization
    if (
      shouldFetchCustomIcon &&
      iconState.status === "apiCustom" &&
      customIconData &&
      !iconState.previewUrl
    ) {
      const img = new Image();
      img.onload = () => {
        fetch(customIconData.url)
          .then((res) => {
            const filename = customIconData.filename || "icon.png";
            return res.blob().then((blob) => ({ blob, filename }));
          })
          .then(({ blob, filename }) => {
            setCurrentApiCustomIcon(
              new File([blob], filename, { type: "image/png" }),
              img.width,
              customIconData.url
            );
            setIsFirstLoadWithCustomIcon(false);
          });
      };
      img.src = customIconData.url;
      return; // Don't run fallback block below on initial load
    }

    // Or handle VPP fallback initialization (only when not using API custom icon)
    if (originalIsVpp && iconState.status !== "customUpload") {
      setIconState({
        previewUrl: previewInfo.currentIconUrl,
        formData: null,
        dimensions: null,
        fileDetails: null,
        status: "fallback",
      });
    }
  }, [
    customIconData,
    iconState.status,
    shouldFetchCustomIcon,
    iconState.previewUrl,
    previewInfo.currentIconUrl,
  ]);

  const fileDetails =
    iconState.formData && iconState.formData.icon
      ? {
          name: iconState.formData.icon.name,
          description: `Software icon • ${iconState.dimensions || "?"}x${
            iconState.dimensions || "?"
          } px`,
        }
      : undefined;

  const renderPreviewFleetCard = () => {
    const {
      name,
      type,
      versions,
      source,
      currentIconUrl,
      countsUpdatedAt,
    } = previewInfo;
    return (
      <Card
        borderRadiusSize="medium"
        color="grey"
        className={`${baseClass}__preview-card`}
        paddingSize="xlarge"
      >
        <Card
          borderRadiusSize="xxlarge"
          className={`${baseClass}__preview-card__fleet`}
        >
          <SoftwareDetailsSummary
            title={displayName || name}
            name={displayName || name}
            type={type}
            source={source}
            iconUrl={
              !currentIconUrl && software.icon_url ? software.icon_url : null
            }
            versions={versions}
            iconPreviewUrl={iconState.previewUrl}
            iconUploadedAt={iconUploadedAt}
          />
          <div className={`${baseClass}__preview-results-count`}>
            <TableCount name="versions" count={versions} />
            {countsUpdatedAt && TitleVersionsLastUpdatedInfo(countsUpdatedAt)}
          </div>
          <div className={`data-table-block ${baseClass}__preview-table`}>
            <div className="data-table data-table__wrapper">
              <table className="data-table__table">
                <thead>
                  <tr role="row">
                    <th
                      className="version__header"
                      colSpan={1}
                      role="columnheader"
                    >
                      <div className="column-header">Version</div>
                    </th>
                    <th
                      className="vulnerabilities__header"
                      colSpan={1}
                      role="columnheader"
                    >
                      <div className="column-header">Vulnerabilities</div>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="single-row" role="row">
                    <td className="version__cell" role="cell">
                      88.0.1
                    </td>
                    <td className="vulnerabilities__cell" role="cell">
                      <div
                        className="vulnerabilities-cell__vulnerability-text-with-tooltip"
                        data-tip="true"
                        data-for="86"
                      >
                        <span className="text-cell w250 italic-cell">
                          20 vulnerabilities
                        </span>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </Card>
        <div
          className={`${baseClass}__mask-overlay ${baseClass}__mask-overlay--fleet`}
        />
      </Card>
    );
  };

  const renderPreviewSelfServiceCard = () => (
    <Card
      borderRadiusSize="medium"
      color="grey"
      className={`${baseClass}__preview-card`}
      paddingSize="xlarge"
    >
      <Card
        className={`${baseClass}__preview-card__self-service`}
        borderRadiusSize="xxlarge"
      >
        <CardHeader header="Self-service" subheader={SELF_SERVICE_SUBHEADER} />
        <div className={`${baseClass}__preview-img-container`}>
          <img
            className={`${baseClass}__preview-img`}
            src={PreviewSelfServiceIcon}
            alt="Preview icon on Fleet Desktop > Self-service"
          />
        </div>
        <div className={`${baseClass}__self-service-preview`}>
          {iconState.previewUrl &&
          isSafeImagePreviewUrl(iconState.previewUrl) ? (
            <img
              src={iconState.previewUrl}
              alt="Uploaded self-service icon"
              style={{
                width: 20,
                height: 20,
                borderRadius: "4px",
                overflow: "hidden",
              }}
            />
          ) : (
            // Known limitation: we cannot see VPP app icons as the fallback when a custom icon
            // is set as VPP icon is not returned by the API if a custom icon is returned
            <SoftwareIcon
              name={displayName || previewInfo.name}
              source={previewInfo.source}
              url={isSoftwarePackage ? undefined : software.icon_url} // fallback PNG icons only exist for VPP apps
              uploadedAt={iconUploadedAt}
            />
          )}
          <div className={`${baseClass}__self-service-preview-name`}>
            <TooltipTruncatedText value={displayName || previewInfo.name} />
          </div>
        </div>
      </Card>
      <div
        className={`${baseClass}__mask-overlay ${baseClass}__mask-overlay--self-service`}
      />
    </Card>
  );

  const renderForm = () => (
    <>
      <InputField
        label="Display name"
        onChange={onInputChange}
        name="displayName"
        value={displayName}
        parseTarget
        helpText={
          <>
            Optional. If left blank, Fleet will use{" "}
            <strong>{previewInfo.name}</strong>.
          </>
        }
        autofocus
      />
      <FileUploader
        label="Icon"
        canEdit
        onDeleteFile={onDeleteFile}
        graphicName="file-png"
        accept={ACCEPTED_EXTENSIONS}
        message={UPLOAD_MESSAGE}
        onFileUpload={onFileSelect}
        buttonMessage="Choose file"
        buttonType="brand-inverse-icon"
        className={`${baseClass}__file-uploader`}
        fileDetails={fileDetails}
        gitopsCompatible={false}
      />
      <h2>Preview</h2>
      <TabNav>
        <Tabs selectedIndex={previewTabIndex} onSelect={onTabChange}>
          <TabList>
            <Tab>
              <TabText>Fleet</TabText>
            </Tab>
            <Tab>
              <TabText>Self-service</TabText>
            </Tab>
          </TabList>
          <TabPanel>{renderPreviewFleetCard()}</TabPanel>
          <TabPanel>{renderPreviewSelfServiceCard()}</TabPanel>
        </Tabs>
      </TabNav>
    </>
  );

  const onClickSave = async () => {
    setIsUpdatingSoftwareInfo(true);
    try {
      const notifications: INotification[] = [];

      // Process icon change
      try {
        if (!iconState.formData?.icon) {
          await softwareAPI.deleteSoftwareIcon(softwareId, teamIdForApi);
          notifications.push({
            id: "icon-removed",
            alertType: "success",
            isVisible: true,
            message: (
              <>
                Successfully removed icon from <b>{software?.name}</b>.
              </>
            ),
            persistOnPageChange: false,
          });
        } else {
          await softwareAPI.editSoftwareIcon(
            softwareId,
            teamIdForApi,
            iconState.formData
          );
          notifications.push({
            id: "icon-edited",
            alertType: "success",
            isVisible: true,
            message: (
              <>
                Successfully edited <b>{previewInfo.name}</b>.
              </>
            ),
            persistOnPageChange: false,
          });
        }
      } catch (e) {
        const errorMessage = getErrorReason(e) || DEFAULT_ERROR_MESSAGE;
        notifications.push({
          id: "icon-error",
          alertType: "error",
          isVisible: true,
          message: errorMessage,
          persistOnPageChange: false,
        });
      }

      // Process display name change
      if (displayName !== previewInfo.name) {
        try {
          await (installerType === "package"
            ? softwareAPI.editSoftwarePackage({
                data: { displayName },
                softwareId,
                teamId: teamIdForApi,
              })
            : mdmAppleAPI.editVppApp(softwareId, teamIdForApi, {
                displayName,
              }));
          notifications.push({
            id: "name-edited",
            alertType: "success",
            isVisible: true,
            message: (
              <>
                Successfully renamed <b>{previewInfo.name}</b> to{" "}
                <b>{displayName}</b>.
              </>
            ),
            persistOnPageChange: false,
          });
        } catch (e) {
          const errorMessage = getErrorReason(e) || DEFAULT_ERROR_MESSAGE;
          notifications.push({
            id: "name-error",
            alertType: "error",
            isVisible: true,
            message: errorMessage,
            persistOnPageChange: false,
          });
        }
      }

      // Show all gathered messages
      if (notifications.length > 1) {
        renderMultiFlash({ notifications });
      } else if (notifications.length === 1) {
        renderFlash(notifications[0].alertType, notifications[0].message);
      }

      refetchSoftwareTitle();
      setIconUploadedAt(new Date().toISOString());
      onExitEditIconModal();
    } catch (e) {
      const errorMessage = getErrorReason(e) || DEFAULT_ERROR_MESSAGE;
      renderFlash("error", errorMessage);
    } finally {
      setIsUpdatingSoftwareInfo(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title={isSoftwarePackage ? "Edit package" : "Edit app"}
      onExit={onExitEditIconModal}
    >
      <>
        {isFirstLoadWithCustomIcon ? (
          <Spinner includeContainer={false} />
        ) : (
          renderForm()
        )}
        <ModalFooter
          primaryButtons={
            <Button
              type="submit"
              onClick={onClickSave}
              isLoading={isUpdatingSoftwareInfo}
              disabled={!canSaveIcon || isUpdatingSoftwareInfo}
            >
              Save
            </Button>
          }
        />
      </>
    </Modal>
  );
};

export default EditIconModal;
