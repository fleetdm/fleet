import classnames from "classnames";
import React, { useState, useRef } from "react";

import Button from "components/buttons/Button";
import Card from "components/Card";
import FileDetails from "components/FileDetails";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "file-uploader";

// The HTML `accept` attribute only filters the native file picker; dropped
// files are not pre-filtered by the browser. Mirror the picker's behavior
// for drops so a FileUploader with `accept=".pem"` doesn't accept a dropped
// `.txt`.
const isFileAccepted = (file: File, accept?: string): boolean => {
  if (!accept) return true;
  const specs = accept
    .split(",")
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean);
  if (specs.length === 0) return true;
  const type = file.type.toLowerCase();
  const name = file.name.toLowerCase();
  return specs.some((spec) => {
    if (spec.startsWith(".")) return name.endsWith(spec);
    if (spec.endsWith("/*")) return type.startsWith(spec.slice(0, -1));
    return type === spec;
  });
};

export type ISupportedGraphicNames = Extract<
  GraphicNames,
  | "file-configuration-profile"
  | "file-sh"
  | "file-ps1"
  | "file-py"
  | "file-script"
  | "file-pdf"
  | "file-pkg"
  | "file-p7m"
  | "file-pem"
  | "file-vpp"
  | "file-png"
  | "file-json"
  | "fleet-logo"
>;

interface IFileUploaderProps {
  label?: React.ReactNode;
  graphicName: ISupportedGraphicNames | ISupportedGraphicNames[];
  message: React.ReactNode;
  title?: string;
  /** allow error state within the file uploader, as opposed to on its label */
  internalError?: string;
  additionalInfo?: string;
  /** Controls the loading spinner on the upload button */
  isLoading?: boolean;
  /** Disables the upload button */
  disabled?: boolean;
  /** A comma separated string of one or more file types accepted to upload.
   * This is the same as the html accept attribute.
   * https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/accept
   */
  accept?: string;
  /** The text to display on the upload button
   * @default "Upload"
   */
  buttonMessage?: string;
  className?: string;
  /** renders the button to open the file uploader to appear as a button or
   * a link.
   * @default "button"
   */
  buttonType?: "button" | "secondary";
  /** `"small"` lays the graphic, message and button out in a single row so
   * the uploader is about half the height of the default stacked card.
   * @default "default"
   */
  variant?: "default" | "small";
  /** renders a tooltip for the button. If `gitopsCompatible` is set to `true`
   * this tooltip will not be rendered if gitops mode is enabled. */
  buttonTooltip?: React.ReactNode;
  onFileUpload: (files: FileList | null) => void;
  /** renders the current file with the edit pencil button */
  canEdit?: boolean;
  /** renders a custom editor for the current file replacing the edit pencil button */
  customEditor?: () => React.ReactNode;
  /** if provided, replaces the default file-type Graphic shown next to the
   * file details (e.g. to display a preview thumbnail of the picked image
   * instead of a generic file icon). */
  customPreview?: React.ReactNode;
  /** renders the current file with the delete trash button */
  onDeleteFile?: () => void;
  /** if provided, will be called when the button is clicked
   * instead of opening the file selector. Useful if you want to
   * show the file selector UI but handle the file selection
   * in a modal.
   */
  onButtonClick?: () => void;
  fileDetails?: {
    name: string;
    description?: React.ReactNode;
  };
  /** Indicates that this file uploader deals with an entity that can be managed by GitOps, and so should be disabled when gitops mode is enabled */
  gitopsCompatible?: boolean;
  /** Whether or not GitOpsMode is enabled. Has no effect if `gitopsCompatible` is false */
  gitOpsModeEnabled?: boolean;
}

/**
 * A component that encapsulates the UI for uploading a file and a file selected.
 */
export const FileUploader = ({
  label,
  graphicName: graphicNames,
  message,
  title,
  internalError,
  additionalInfo,
  isLoading = false,
  disabled = false,
  accept,
  className,
  buttonMessage = "Upload",
  buttonType = "button",
  variant = "default",
  buttonTooltip,
  onButtonClick,
  onFileUpload,
  canEdit = false,
  customEditor,
  customPreview,
  onDeleteFile,
  fileDetails,
  gitopsCompatible = false,
  gitOpsModeEnabled = false,
}: IFileUploaderProps) => {
  const [isFileSelected, setIsFileSelected] = useState(!!fileDetails);
  const [isDragActive, setIsDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // When onButtonClick is set, the uploader renders no file input (file
  // selection happens elsewhere, e.g. in a modal), so drops have nowhere
  // to go. GitOps mode is a safety gate — if the click button is suppressed
  // by GitOps, drops must be suppressed too.
  const canAcceptDrop =
    !disabled &&
    !isLoading &&
    !onButtonClick &&
    !fileDetails &&
    !(gitopsCompatible && gitOpsModeEnabled);

  const classes = classnames(baseClass, className, {
    [`${baseClass}__file-preview`]: isFileSelected,
    [`${baseClass}__error`]: !!internalError,
    [`${baseClass}__drag-active`]: isDragActive && canAcceptDrop,
    [`${baseClass}--small`]: variant === "small",
  });
  const buttonVariant = buttonType === "button" ? "default" : "secondary";

  const triggerFileInput = () => {
    fileInputRef.current?.click();
  };

  const onFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const target = e.currentTarget;
    // Ensure target is the expected input element to prevent DOM manipulation
    if (target && target.type === "file") {
      const files = target.files;
      onFileUpload(files);
      setIsFileSelected(true);

      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  // Always preventDefault on drag events over the component so the browser
  // doesn't fall back to opening/downloading the file when a drop lands on
  // the Card's padding or a FileDetails preview.
  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    if (canAcceptDrop && !isDragActive) setIsDragActive(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    // dragleave fires when the pointer enters a child element too; ignore
    // those so the active state doesn't flicker.
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      setIsDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragActive(false);
    if (!canAcceptDrop) return;
    const files = e.dataTransfer.files;
    if (!files || files.length === 0) return;
    // Reject the whole drop if any file would have been filtered out by
    // the native picker. The signature returns FileList (read-only), so we
    // can't hand back a partial batch; all-or-nothing matches "picker
    // declined this file" semantics.
    const hasInvalid = Array.from(files).some(
      (file) => !isFileAccepted(file, accept)
    );
    if (hasInvalid) return;
    onFileUpload(files);
    setIsFileSelected(true);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      triggerFileInput();
    }
  };

  // The small variant has no separate upload button; the card itself is the
  // click target, so it also needs to open the picker on Space.
  const handleCardKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      triggerFileInput();
    }
  };

  const renderLabel = () => {
    return label ? (
      <div className={`${baseClass}__label form-field__label`}>{label}</div>
    ) : null;
  };
  const renderGraphics = () => {
    const graphicNamesArr =
      typeof graphicNames === "string" ? [graphicNames] : graphicNames;
    return graphicNamesArr.map((graphicName) => (
      <Graphic
        key={`${graphicName}-graphic`}
        className={`${baseClass}__graphic`}
        name={graphicName}
      />
    ));
  };

  const renderUploadButton = () => {
    let buttonMarkup = (
      <>
        {buttonMessage}
        {buttonType === "secondary" && <Icon name="upload" />}
      </>
    );
    // If we want to actual do file uploading, wrap in a label that
    // references the hidden file input. Otherwise just use a span.
    if (!onButtonClick) {
      buttonMarkup = <label htmlFor="upload-file">{buttonMarkup}</label>;
    } else {
      buttonMarkup = <span>{buttonMarkup}</span>;
    }
    // the gitops mode tooltip wrapper takes presedence over other button
    // renderings
    if (gitopsCompatible) {
      return (
        <GitOpsModeTooltipWrapper
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              className={`${baseClass}__manual-install-tooltip`}
              tipContent={buttonTooltip}
              disableTooltip={disableChildren || !buttonTooltip}
              position="top"
              showArrow
              underline={false}
            >
              <Button
                className={`${baseClass}__upload-button`}
                variant={buttonVariant}
                isLoading={isLoading}
                disabled={disabled || disableChildren}
                customOnKeyDown={!onButtonClick ? handleKeyDown : undefined}
                onClick={onButtonClick || undefined}
                tabIndex={0}
              >
                {buttonMarkup}
              </Button>
            </TooltipWrapper>
          )}
        />
      );
    }

    return (
      <TooltipWrapper
        className={`${baseClass}__upload-button`}
        position="top"
        tipContent={buttonTooltip}
        underline={false}
        showArrow
        disableTooltip={!buttonTooltip}
      >
        <Button
          className={`${baseClass}__upload-button`}
          variant={buttonVariant}
          isLoading={isLoading}
          disabled={disabled}
          customOnKeyDown={!onButtonClick ? handleKeyDown : undefined}
          onClick={onButtonClick || undefined}
          tabIndex={0}
        >
          {buttonMarkup}
        </Button>
      </TooltipWrapper>
    );
  };

  const renderTitle = () => {
    if (internalError) {
      return (
        <div className={`${baseClass}__internal-error`}>{internalError}</div>
      );
    }
    if (title) {
      return <div className={`${baseClass}__title`}>{title}</div>;
    }
    return null;
  };

  const renderFileUploader = () => {
    const isCardClickTarget = variant === "small" && !onButtonClick;
    const content = (
      <div className={`${baseClass}__content-wrapper`}>
        <div className={`${baseClass}__outer`}>
          <div className={`${baseClass}__inner`}>
            <div className={`${baseClass}__graphics`}>{renderGraphics()}</div>
            <div className={`${baseClass}__text`}>
              {renderTitle()}
              <p className={`${baseClass}__message`}>{message}</p>
              {additionalInfo && (
                <p className={`${baseClass}__additional-info`}>
                  {additionalInfo}
                </p>
              )}
            </div>
          </div>
          {!isCardClickTarget && renderUploadButton()}
          {/* If onButtonClick is provided, we're not actually uploading files here. */}
          {!onButtonClick && (
            <input
              ref={fileInputRef}
              accept={accept}
              id="upload-file"
              type="file"
              aria-label={buttonMessage}
              disabled={disabled || isLoading}
              onChange={onFileSelect}
              className="file-input-visually-hidden"
            />
          )}
        </div>
      </div>
    );

    if (isCardClickTarget) {
      return (
        <label
          htmlFor="upload-file"
          className={`${baseClass}__click-target`}
          tabIndex={disabled || isLoading ? -1 : 0}
          onKeyDown={handleCardKeyDown}
        >
          {content}
        </label>
      );
    }
    return content;
  };

  return (
    <div
      className={`${baseClass}__wrapper form-field`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {renderLabel()}
      <Card color="grey" className={classes}>
        {fileDetails ? (
          <FileDetails
            graphicNames={graphicNames}
            fileDetails={fileDetails}
            canEdit={canEdit}
            customEditor={customEditor}
            customPreview={customPreview}
            onDeleteFile={onDeleteFile}
            onFileSelect={onFileSelect}
            accept={accept}
            gitopsCompatible={gitopsCompatible}
            gitOpsModeEnabled={gitOpsModeEnabled}
            disabled={disabled}
          />
        ) : (
          renderFileUploader()
        )}
      </Card>
    </div>
  );
};

export default FileUploader;
