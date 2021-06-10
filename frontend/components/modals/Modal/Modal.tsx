import React, { Component } from "react";
import classnames from "classnames";

const baseClass = "modal";

interface IModalProps {
  children: JSX.Element;
  onExit: () => void;
  title: string | JSX.Element;
  className?: string;
}

class Modal extends Component<IModalProps> {
  render(): JSX.Element {
    const { children, className, onExit, title } = this.props;
    const modalContainerClassName = classnames(
      `${baseClass}__modal_container`,
      className
    );

    return (
      <div className={`${baseClass}__background`}>
        <div className={modalContainerClassName}>
          <div className={`${baseClass}__header`}>
            <span>{title}</span>
            <div className={`${baseClass}__ex`}>
              <button className="button button--unstyled" onClick={onExit} />
            </div>
          </div>
          <div className={`${baseClass}__content`}>{children}</div>
        </div>
      </div>
    );
  }
}

export default Modal;
