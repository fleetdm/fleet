import React, { useCallback, useEffect, useRef, useState } from "react";
import classnames from "classnames";
import Button from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";

const baseClass = "modal";
const CLOSE_ANIMATION_MS = 100;

type ModalWidth = "medium" | "large" | "xlarge" | "auto";
//                  650px    800px      850px      auto

export interface IModalProps {
  title: string | JSX.Element;
  children: React.ReactNode;
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
}: IModalProps): JSX.Element => {
  const isDownOnBackgroundRef = useRef(false);
  const isFormDirtyRef = useRef(false);
  const [isClosing, setIsClosing] = useState(false);
  const isClosingRef = useRef(false);

  const handleClose = useCallback(() => {
    if (isClosingRef.current) return;
    isClosingRef.current = true;
    setIsClosing(true);
    setTimeout(() => {
      onExit();
    }, CLOSE_ANIMATION_MS);
  }, [onExit]);

  useEffect(() => {
    const closeWithEscapeKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleClose();
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
  }, [disableClosingModal, handleClose]);

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
    return undefined;
  }, [onEnter]);

  const backgroundClasses = classnames(`${baseClass}__background`, {
    [`${baseClass}__hidden`]: isHidden,
    [`${baseClass}__closing`]: isClosing,
  });

  const modalContainerClasses = classnames(
    className,
    `${baseClass}__modal_container`,
    `${baseClass}__modal_container__${width}`,
    {
      [`${className}__loading`]: isLoading,
      [`${baseClass}__closing`]: isClosing,
    }
  );

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-wrapper-disabled`]: isContentDisabled,
  });

  const contentClasses = classnames(`${baseClass}__content`, {
    [`${baseClass}__content-disabled`]: isContentDisabled,
  });

  const handleBackgroundMouseDown = () => {
    isDownOnBackgroundRef.current = true;
  };

  const handleBackgroundMouseUp = () => {
    if (
      !disableClosingModal &&
      isDownOnBackgroundRef.current &&
      !isFormDirtyRef.current
    ) {
      handleClose();
    }
    isDownOnBackgroundRef.current = false;
  };

  const handleContainerMouseDown = (e: React.MouseEvent) => e.stopPropagation();

  const handleContainerMouseUp = (e: React.MouseEvent) => e.stopPropagation();

  const handleContainerInput = () => {
    isFormDirtyRef.current = true;
  };

  const handleContainerClick = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    const isCheckbox =
      target instanceof HTMLInputElement && target.type === "checkbox";
    const isToggle = !!target.closest('button[role="switch"]');
    if (isCheckbox || isToggle) {
      isFormDirtyRef.current = true;
    }
  };

  return (
    <div
      className={backgroundClasses}
      style={
        {
          "--modal-close-duration": `${CLOSE_ANIMATION_MS}ms`,
        } as React.CSSProperties
      }
      onMouseDown={handleBackgroundMouseDown}
      onMouseUp={handleBackgroundMouseUp}
    >
      <div
        className={modalContainerClasses}
        tabIndex={-1} // Make focusable
        onMouseDown={handleContainerMouseDown}
        onMouseUp={handleContainerMouseUp}
        onInput={handleContainerInput}
        onClick={handleContainerClick}
      >
        <div className={`${baseClass}__header`}>
          <span>{title}</span>
          {!disableClosingModal && (
            <div className={`${baseClass}__ex`}>
              <Button
                variant="icon"
                onClick={handleClose}
                iconStroke
                autofocus={isContentDisabled}
              >
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
      </div>
    </div>
  );
};

export default Modal;
