import React, { useContext, useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { IAppStoreApp, ISoftwarePackage } from "interfaces/software";

import { NotificationContext } from "context/notification";
import softwareAPI, {
  MAX_FILE_SIZE_BYTES,
  MAX_FILE_SIZE_MB,
} from "services/entities/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import FileUploader from "components/FileUploader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Card from "components/Card";
import Button from "components/buttons/Button";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import DataSet from "components/DataSet";
import TableCount from "components/TableContainer/TableCount";
import CardHeader from "components/CardHeader";
import { resultsCountClass } from "components/TableContainer/TableContainer";

import {
  baseClass as softwareDetailsBaseClass,
  descriptionListClass,
  infoClass,
} from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary/SoftwareDetailsSummary";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";
import { SELF_SERVICE_SUBHEADER } from "pages/hosts/details/cards/Software/SelfService/SelfService";
import { getFileDetails } from "utilities/file/fileUtils";

import { TitleVersionsLastUpdatedInfo } from "../SoftwareSummaryCard/TitleVersionsTable/TitleVersionsTable";
import { getErrorMessage } from "./helpers";
import PreviewSelfServiceIcon from "../../../../../assets/images/preview-self-service-icon.png";

const baseClass = "edit-icon-modal";

const ACCEPTED_EXTENSIONS = ".png";
const UPLOAD_MESSAGE =
  "The icon must be a PNG file and square, with dimensions ranging from 120x120 px to 1024x1024 px.";
const MIN_DIMENSION = 120;
const MAX_DIMENSION = 1024;

type IMessageFunc = (formData: any) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<any, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: any) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = () => {
  const FORM_VALIDATIONS: IFormValidations = {
    icon: {
      validations: [
        {
          name: "required",
          isValid: (formData: any) => {
            return true; // TODO
          },
        },
      ],
    },
  };

  return FORM_VALIDATIONS;
};

// Install type used on add but not edit
export type IEditPackageFormData = Omit<IPackageFormData, "installType">;

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

  const [editSoftwareModalClasses, setEditIconModalClasses] = useState(
    baseClass
  );
  const [isUpdatingIcon, setIsUpdatingIcon] = useState(false);

  const [formData, setFormData] = useState<any>();
  const [iconDimensions, setIconDimensions] = useState<number | null>(null);
  const [iconPreviewUrl, setIconPreviewUrl] = useState<string | null>(null);
  const [formValidation, setFormValidation] = useState<any>();
  const [navTabIndex, setNavTabIndex] = useState(0);

  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      // Enforce PNG MIME type
      if (file.type !== "image/png") {
        renderFlash("error", "Couldn't edit. Must be a PNG file.");
        return;
      }

      const reader = new FileReader();
      reader.onload = (e: any) => {
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
        img.src = e.target.result;
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

  // TODO: fix
  // Edit package API call
  const onEditPackage = async () => {
    setIsUpdatingIcon(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't edit software. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUpdatingIcon(false);
      return;
    }

    try {
      await softwareAPI.editSoftwarePackage({
        data: formData,
        orignalPackage: software as ISoftwarePackage,
        softwareId,
        teamId: teamIdForApi,
      });

      renderFlash(
        "success",
        <>
          Successfully edited <b>{formData.software?.name}</b>.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e, software as IAppStoreApp));
    }
    setIsUpdatingIcon(false);
  };

  // TODO: fix
  const onEditApp = async () => {
    setIsUpdatingIcon(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't edit software. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUpdatingIcon(false);
      return;
    }

    try {
      await softwareAPI.editSoftwarePackage({
        data: formData,
        orignalPackage: software as ISoftwarePackage,
        softwareId,
        teamId: teamIdForApi,
      });

      renderFlash(
        "success",
        <>
          Successfully edited <b>{formData.software?.name}</b>.
        </>
      );
      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e, software as IAppStoreApp));
    }
    setIsUpdatingIcon(false);
  };

  const onClickSave = () => {
    if (isSoftwarePackage) {
      onEditPackage();
    } else {
      onEditApp();
    }
  };

  const renderPreviewFleetCard = () => {
    const { name, type, versions, countsUpdatedAt } = previewInfo;
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
          <div className={softwareDetailsBaseClass}>
            {iconPreviewUrl ? (
              <img
                src={iconPreviewUrl}
                alt="Uploaded icon preview"
                style={{ width: 96, height: 96 }}
              />
            ) : (
              <SoftwareIcon name={software.name} source="apps" size="xlarge" />
            )}
            <dl className={infoClass}>
              <h1>{name}</h1>
              <dl className={descriptionListClass}>
                {!!type && <DataSet title="Type" value={type} />}

                {!!previewInfo.versions && (
                  <DataSet title="Versions" value={versions} />
                )}
              </dl>
            </dl>
          </div>
          <div className={resultsCountClass}>
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
                          620 vulnerabilities
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
          <div>
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
              <SoftwareIcon name={software.name} source="apps" size="xsmall" />
            )}
            {previewInfo.name}
          </div>
        </Card>
        <div
          className={`${baseClass}__mask-overlay ${baseClass}__mask-overlay--self-service`}
        />
      </Card>
    );
  };

  const renderForm = () => {
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
          fileDetails={
            formData && formData.icon
              ? {
                  name: formData.icon.name,
                  description: `Software icon • ${iconDimensions}x${iconDimensions}px`,
                }
              : undefined
          }
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
        className={editSoftwareModalClasses}
        title={isSoftwarePackage ? "Edit package" : "Edit app"}
        onExit={onExit}
      >
        <>
          {renderForm()}
          <ModalFooter
            primaryButtons={
              <Button type="submit" onClick={onClickSave}>
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
