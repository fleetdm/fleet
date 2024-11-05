import React, { useEffect } from "react";
import classnames from "classnames";
import Button from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";

const baseClass = "modal";

type ModalWidth = "medium" | "large" | "xlarge" | "auto";

export interface IModalProps {
  title: string | JSX.Element;
  children: JSX.Element;
  onExit: () => void;
  onEnter?: () => void;
  /**     medium 650px, large 800px, xlarge 850px, auto auto-width
   * @default "medium"
   */
  width?: ModalWidth;
  /**    isHidden can be set true to hide the modal when opening another modal
   * @default false
   */
  isHidden?: boolean;
  /**    isLoading can be set true to enable targeting elements by loading state
   * @default false
   */
  isLoading?: boolean;
  /** `isContentDisabled` can be set to true to display the modal content as disabled.
   * At the moment this will place an overlay over the modal content and make it
   * unclickable. The top right will not be disabled and will still be clickable.
   *
   * @default false
   */
  isContentDisabled?: boolean;
  /** `disableClosingModal` can be set to disable the users ability to manually
   * close the modal.
   * @default false
   * */
  disableClosingModal?: boolean;
  className?: string;
  actionsFooter?: JSX.Element;
}

const Modal = ({
  title,
  children,
  onExit,
  onEnter,
  width = "medium",
  isHidden = false,
  isLoading = false,
  isContentDisabled = false,
  disableClosingModal = false,
  className,
  actionsFooter,
}: IModalProps): JSX.Element => {
  useEffect(() => {
    const closeWithEscapeKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onExit();
      }
    };

    if (!disableClosingModal) {
      document.addEventListener("keydown", closeWithEscapeKey);
    }

    return () => {
      if (!disableClosingModal) {
        document.removeEventListener("keydown", closeWithEscapeKey);
      }
    };
  }, [disableClosingModal, onExit]);

  useEffect(() => {
    if (onEnter) {
      const closeOrSaveWithEnterKey = (event: KeyboardEvent) => {
        if (event.code === "Enter" || event.code === "NumpadEnter") {
          event.preventDefault();
          onEnter();
        }
      };

      document.addEventListener("keydown", closeOrSaveWithEnterKey);
      return () => {
        document.removeEventListener("keydown", closeOrSaveWithEnterKey);
      };
    }
  }, [onEnter]);

  const backgroundClasses = classnames(`${baseClass}__background`, {
    [`${baseClass}__hidden`]: isHidden,
  });

  const modalContainerClasses = classnames(
    className,
    `${baseClass}__modal_container`,
    `${baseClass}__modal_container__${width}`,
    {
      [`${className}__loading`]: isLoading,
    }
  );

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-wrapper-disabled`]: isContentDisabled,
  });

  const contentClasses = classnames(`${baseClass}__content`, {
    [`${baseClass}__content-disabled`]: isContentDisabled,
  });

  return (
    <div className={backgroundClasses}>
      <div className={modalContainerClasses}>
        <div className={`${baseClass}__header`}>
          <span>{title}</span>
          {!disableClosingModal && (
            <div className={`${baseClass}__ex`}>
              <Button variant="unstyled" onClick={onExit}>
                <Icon name="close" color="core-fleet-black" size="medium" />
              </Button>
            </div>
          )}
        </div>
        <div className={contentWrapperClasses}>
          {isContentDisabled && (
            <div className={`${baseClass}__disabled-overlay`} />
          )}
          <div className={contentClasses}>{children}</div>
        </div>
        {actionsFooter && (
          <div className={`${baseClass}__actions-footer`}>{actionsFooter}</div>
        )}
      </div>
    </div>
  );
};

export default Modal;
