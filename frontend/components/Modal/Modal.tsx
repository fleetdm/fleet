import classnames from "classnames";
import React, { useCallback, useEffect, useRef, useState } from "react";

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
  /** Called when the user presses Enter. Avoid using this on modals that
   * contain forms, reveal/copy controls, or other elements where Enter has
   * its own meaning — it will conflict with keyboard navigation. */
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

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(",");

const getFocusable = (container: HTMLElement): HTMLElement[] => {
  const nodes = container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
  return Array.from(nodes).filter((el) => {
    const style = window.getComputedStyle(el);
    return style.display !== "none" && style.visibility !== "hidden";
  });
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
  className,
}: IModalProps): JSX.Element => {
  const isDownOnBackgroundRef = useRef(false);
  const isFormDirtyRef = useRef(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLElement | null>(null);
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

  useEffect(() => {
    const onWindowBlur = () => {
      isDownOnBackgroundRef.current = false;
    };
    window.addEventListener("blur", onWindowBlur);
    return () => {
      window.removeEventListener("blur", onWindowBlur);
    };
  }, []);

  useEffect(() => {
    const { body, documentElement } = document;
    // A stacked modal would measure the already-hidden scrollbar as 0.
    if (!body.classList.contains("modal-open")) {
      const scrollbarWidth = window.innerWidth - documentElement.clientWidth;
      // Takes the place of the scrollbar that overflow: hidden removes, so the
      // page doesn't widen. Inline to win over the command palette's scroll lock.
      body.style.paddingRight = `${scrollbarWidth}px`;
      body.classList.add("modal-open");
    }
    return () => {
      // By cleanup time this modal's own background node is already
      // detached, so only unlock scroll once none remain.
      if (document.querySelectorAll(`.${baseClass}__background`).length === 0) {
        body.classList.remove("modal-open");
        body.style.removeProperty("padding-right");
      }
    };
  }, []);

  useEffect(() => {
    triggerRef.current = document.activeElement as HTMLElement | null;

    const container = containerRef.current;
    // isContentDisabled autofocuses the close button via the Button's own
    // autofocus prop, so don't fight it.
    if (container && !isContentDisabled) {
      const focusables = getFocusable(container);
      (focusables[0] ?? container).focus();
    }

    return () => {
      const trigger = triggerRef.current;
      if (trigger && document.body.contains(trigger)) {
        trigger.focus();
      }
    };
    // Mount-only: initial focus and trigger capture happen once per modal lifetime.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (isHidden) return undefined;

    const trapFocus = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;
      const container = containerRef.current;
      if (!container) return;

      const focusables = getFocusable(container);
      if (focusables.length === 0) {
        e.preventDefault();
        container.focus();
        return;
      }

      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement as HTMLElement | null;

      if (!container.contains(active)) {
        e.preventDefault();
        first.focus();
        return;
      }

      if (e.shiftKey && active === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", trapFocus);
    return () => {
      document.removeEventListener("keydown", trapFocus);
    };
  }, [isHidden]);

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
      {/* Handlers here are stopPropagation shims for the click-outside-to-close
          overlay above, not user-facing interactions on the dialog itself. */}
      {/* eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions */}
      <div
        ref={containerRef}
        className={modalContainerClasses}
        role="dialog"
        aria-modal="true"
        aria-label={typeof title === "string" ? title : undefined}
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
                variant="subdued"
                onClick={handleClose}
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
