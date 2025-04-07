import React from "react";

import classnames from "classnames";

import { IFileDetails } from "utilities/file/fileUtils";

import Button from "components/buttons/Button";
import { ISupportedGraphicNames } from "components/FileUploader/FileUploader";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

export type IFileDetailsSupportedGraphicNames =
  | ISupportedGraphicNames
  | "app-store"; // For VPP apps (non-editable)

interface IFileDetailsProps {
  graphicNames:
    | IFileDetailsSupportedGraphicNames
    | IFileDetailsSupportedGraphicNames[];
  fileDetails: IFileDetails;
  canEdit: boolean;
  onFileSelect?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  accept?: string;
  progress?: number;
  gitOpsModeEnabled?: boolean;
}

const baseClass = "file-details";

const FileDetails = ({
  graphicNames,
  fileDetails,
  canEdit,
  onFileSelect,
  accept,
  progress,
  gitOpsModeEnabled = false,
}: IFileDetailsProps) => {
  const infoClasses = classnames(`${baseClass}__info`, {
    [`${baseClass}__info--disabled-by-gitops-mode`]: gitOpsModeEnabled,
  });
  return (
    <div className={baseClass}>
      {/* disabling at this level preserves funcitonality of GitOpsModeTooltipWrapper around the edit icon */}
      <div className={infoClasses}>
        <Graphic
          name={
            typeof graphicNames === "string" ? graphicNames : graphicNames[0]
          }
        />
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__name`}>{fileDetails.name}</div>
          {fileDetails.platform && (
            <div className={`${baseClass}__platform`}>
              {fileDetails.platform}
            </div>
          )}
        </div>
      </div>
      {!progress && canEdit && onFileSelect && (
        <GitOpsModeTooltipWrapper
          position="left"
          tipOffset={-8}
          renderChildren={(disableChildren) => (
            <div className={`${baseClass}__edit`}>
              <Button
                disabled={disableChildren}
                className={`${baseClass}__edit-button`}
                variant="icon"
              >
                <label htmlFor="edit-file">
                  <Icon name="pencil" color="ui-fleet-black-75" />
                </label>
              </Button>
              <input
                disabled={disableChildren}
                accept={accept}
                id="edit-file"
                type="file"
                onChange={onFileSelect}
              />
            </div>
          )}
        />
      )}
      {!!progress && (
        <div className={`${baseClass}__progress-wrapper`}>
          <div className={`${baseClass}__progress-bar`}>
            <div
              className={`${baseClass}__progress-bar--uploaded`}
              style={{
                width: `${progress * 100}%`,
              }}
              title="upload progress bar"
            />
          </div>
          <div className={`${baseClass}__progress-text`}>
            {Math.round(progress * 100)}%
          </div>
        </div>
      )}
    </div>
  );
};

export default FileDetails;
