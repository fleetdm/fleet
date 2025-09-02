import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { IAppStoreApp, ISoftwarePackage } from "interfaces/software";

import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import FileUploader from "components/FileUploader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Card from "components/Card";
import Button from "components/buttons/Button";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TableCount from "components/TableContainer/TableCount";
import CardHeader from "components/CardHeader";
import TooltipTruncatedText from "components/TooltipTruncatedText";

import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary/SoftwareDetailsSummary";
import { SELF_SERVICE_SUBHEADER } from "pages/hosts/details/cards/Software/SelfService/SelfService";

import { TitleVersionsLastUpdatedInfo } from "../SoftwareSummaryCard/TitleVersionsTable/TitleVersionsTable";
import PreviewSelfServiceIcon from "../../../../../assets/images/preview-self-service-icon.png";

const baseClass = "edit-icon-modal";

const ACCEPTED_EXTENSIONS = ".png";
const MIN_DIMENSION = 120;
const MAX_DIMENSION = 1024;
const UPLOAD_MESSAGE = `The icon must be a PNG file and square, with dimensions ranging from ${MIN_DIMENSION.toString()}x${MIN_DIMENSION.toString()} px to ${MAX_DIMENSION.toString()}x${MAX_DIMENSION.toString()} px.`;
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
  if (matchStandard) return matchStandard[1];
  return null;
};

interface IIconFormData {
  icon: File;
}
interface IEditIconModalProps {
  softwareId: number;
  teamIdForApi: number;
  software: ISoftwarePackage | IAppStoreApp;
  onExit: () => void;
  refetchSoftwareTitle: () => void;
  installerType: "package" | "vpp";
  previewInfo: {
    type?: string;
    versions?: number;
    source?: string;
    currentIconUrl: string | null;
    name: string;
    countsUpdatedAt?: string;
  };
}

const EditIconModal = ({
  softwareId,
  teamIdForApi,
  software,
  onExit,
  refetchSoftwareTitle,
  installerType,
  previewInfo,
}: IEditIconModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const isSoftwarePackage = installerType === "package";

  const [isUpdatingIcon, setIsUpdatingIcon] = useState(false);

  const [formData, setFormData] = useState<IIconFormData | null>(null);
  const [iconDimensions, setIconDimensions] = useState<number | null>(null);
  const [iconPreviewUrl, setIconPreviewUrl] = useState<string | null>(null);
  const [navTabIndex, setNavTabIndex] = useState(0);

  console.log("formData", formData);

  const shouldFetchCustomIcon =
    !!previewInfo.currentIconUrl &&
    previewInfo.currentIconUrl.startsWith("/api/");

  console.log("shouldGetIconFromApi", shouldFetchCustomIcon);
  // If there's an API URL and no file already selected, fetch the existing icon
  const { data: customIconBlob, isLoading, error } = useQuery<
    Blob | undefined,
    AxiosError,
    string
  >(
    ["softwareIcon", softwareId, teamIdForApi],
    () => softwareAPI.getSoftwareIcon(softwareId, teamIdForApi),
    {
      enabled: shouldFetchCustomIcon,
      retry: false,
      select: (blob) => (blob ? URL.createObjectURL(blob) : ""),
    }
  );

  // On mount, if we have an icon URL from the API, load it into state
  useEffect(() => {
    if (customIconBlob) {
      const img = new Image();
      img.onload = () => setIconDimensions(img.width);
      img.src = customIconBlob;

      // Fetch the blob again to get the filename from headers
      fetch(customIconBlob)
        .then((res) => {
          const header = res.headers.get("Content-Disposition");
          let filename = "icon.png";
          console.log("res", res);
          if (header) {
            console.log("header", header);
            const matchQuoted = header.match(/filename=["']?([^"';]+)["']?/);
            if (matchQuoted) filename = matchQuoted[1];
          }
          return res.blob().then((blob) => ({ blob, filename }));
        })
        .then(({ blob, filename }) => {
          setFormData({
            icon: new File([blob], filename, { type: "image/png" }),
          });
        });
      setIconPreviewUrl(customIconBlob);
    }
  }, []);

  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      // Enforce PNG MIME type
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

          // All checks passed → update formData & preview
          const newData = { ...formData, icon: file };
          setFormData(newData);
          setIconDimensions(width);

          // create preview url
          const previewUrl = URL.createObjectURL(file);
          setIconPreviewUrl(previewUrl);
        };
        if (e.target && typeof e.target.result === "string") {
          img.src = e.target.result; // img.src expects a string
        } else {
          renderFlash("error", "FileReader result was not a string.");
        }
      };
      reader.readAsDataURL(file);
    }
  };

  const onDeleteFile = () => {
    setFormData(null);
    setIconDimensions(null);
    setIconPreviewUrl(null);
  };

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  const onClickSave = async () => {
    setIsUpdatingIcon(true);

    try {
      if (!formData?.icon) {
        await softwareAPI.deleteSoftwareIcon(softwareId, teamIdForApi);
        renderFlash(
          "success",
          <>
            Successfully removed icon from <b>{software?.name}</b>.
          </>
        );
      } else {
        await softwareAPI.editSoftwareIcon(softwareId, teamIdForApi, formData);
        renderFlash(
          "success",
          <>
            Successfully edited <b>{previewInfo.name}</b>.
          </>
        );
      }
      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash("error", DEFAULT_ERROR_MESSAGE);
    } finally {
      setIsUpdatingIcon(false);
    }
  };

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
            title={name}
            name={name}
            type={type}
            source={source}
            iconUrl={
              !previewInfo.currentIconUrl && software.icon_url
                ? software.icon_url
                : null
            } // Rely on iconPreviewwUrl or uploaded file so if we "trash" a custom icon we want to see a preview of the default icon
            versions={versions}
            hosts={0} // required field but not shown in isPreview
            iconPreviewUrl={iconPreviewUrl || null}
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

  const renderPreviewSelfServiceCard = () => {
    return (
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
          <CardHeader
            header="Self-service"
            subheader={SELF_SERVICE_SUBHEADER}
          />
          <div className={`${baseClass}__preview-img-container`}>
            <img
              className={`${baseClass}__preview-img`}
              src={PreviewSelfServiceIcon}
              alt="Preview icon on Fleet Desktop > Self-service"
            />
          </div>
          <div className={`${baseClass}__self-service-preview`}>
            {iconPreviewUrl ? (
              <img
                src={iconPreviewUrl}
                alt="Uploaded self-service icon"
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: "4px",
                  overflow: "hidden",
                }}
              />
            ) : (
              <SoftwareIcon
                name={software.name}
                source={previewInfo.source}
                url={isSoftwarePackage ? undefined : software.icon_url}
              />
            )}
            <TooltipTruncatedText value={previewInfo.name} />
          </div>
        </Card>
        <div
          className={`${baseClass}__mask-overlay ${baseClass}__mask-overlay--self-service`}
        />
      </Card>
    );
  };

  const renderForm = () => {
    const fileDetails =
      formData && formData.icon
        ? {
            name: formData.icon.name,
            description: `Software icon • ${iconDimensions || "?"}x${
              iconDimensions || "?"
            } px`,
          }
        : undefined;

    console.log("fileDetails", fileDetails);

    return (
      <>
        <FileUploader
          canEdit
          onDeleteFile={onDeleteFile}
          graphicName="file-png"
          accept={ACCEPTED_EXTENSIONS}
          message={UPLOAD_MESSAGE}
          onFileUpload={onFileSelect}
          buttonMessage="Choose file"
          buttonType="link"
          className={`${baseClass}__file-uploader`}
          fileDetails={fileDetails}
          gitopsCompatible={false}
        />
        <h2>Preview</h2>
        <TabNav>
          <Tabs selectedIndex={navTabIndex} onSelect={onTabChange}>
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
  };

  return (
    <>
      <Modal
        className={baseClass}
        title={isSoftwarePackage ? "Edit package" : "Edit app"}
        onExit={onExit}
      >
        <>
          {renderForm()}
          <ModalFooter
            primaryButtons={
              <Button
                type="submit"
                onClick={onClickSave}
                isLoading={isUpdatingIcon}
              >
                Save
              </Button>
            }
          />
        </>
      </Modal>
    </>
  );
};

export default EditIconModal;
