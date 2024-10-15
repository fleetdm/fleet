import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useRef,
} from "react";
import classnames from "classnames";
import Button from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";
import { noop } from "lodash";

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
  /** Modal defaults focus to first element for ability to tab through app
   * `disableModalAutofocus` when a specific form field should be autofocused */
  disableModalAutofocus?: boolean;
  className?: string;
}

// Context for focus handling
// Child buttons to handle their own Enter key press events when focused.
// The modal's onEnter function to be executed when Enter is pressed and no child button is focused.
const ModalContext = createContext<{
  setChildFocused: (focused: boolean) => void;
  onEnter?: () => void;
}>({ setChildFocused: noop });

export const useModalContext = () => useContext(ModalContext);

const getFirstFocusableElement = (
  container: HTMLElement | null
): HTMLElement | null => {
  if (!container) return null;
  const focusableElements = container.querySelectorAll<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  );
  return focusableElements[0] || null;
};

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
  disableModalAutofocus = false,
  className,
}: IModalProps): JSX.Element => {
  const [isChildFocused, setIsChildFocused] = useState(false);

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
      const handleEnterKey = (event: KeyboardEvent) => {
        if (
          (event.code === "Enter" || event.code === "NumpadEnter") &&
          !isChildFocused
        ) {
          event.preventDefault();
          onEnter();
        }
      };

      document.addEventListener("keydown", handleEnterKey);
      return () => {
        document.removeEventListener("keydown", handleEnterKey);
      };
    }
  }, [onEnter, isChildFocused]);

  const setChildFocused = (focused: boolean) => {
    setIsChildFocused(focused);
  };

  const modalRef = useRef(null);

  useEffect(() => {
    if (!disableModalAutofocus) {
      const firstFocusableElement = getFirstFocusableElement(modalRef.current);
      if (firstFocusableElement) {
        firstFocusableElement.focus();
      }
    }
  }, [disableModalAutofocus]);

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
    <div ref={modalRef} className={backgroundClasses}>
      <div className={modalContainerClasses}>
        <div className={`${baseClass}__header`}>
          <span>{title}</span>
          {!disableClosingModal && (
            <div className={`${baseClass}__ex`}>
              <Button
                className="button button--unstyled"
                onClick={onExit}
                onFocus={() => setChildFocused(true)}
                onBlur={() => setChildFocused(false)}
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
          <div className={contentClasses}>
            <ModalContext.Provider value={{ setChildFocused, onEnter }}>
              {children}
            </ModalContext.Provider>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Modal;
